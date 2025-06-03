package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
	"strings"
	"sync"
	//ipfsapi "github.com/ipfs/go-ipfs-api"
	shell "github.com/ipfs/go-ipfs-api"
	// "flag"
)

var sh = shell.NewShell("localhost:5001")
var fileMutex sync.Mutex // Create a mutex for file access synchronization


type SeekQuery struct {
	Uuid         string
	Attempts     int
	Depth        int
	CrawlVersion string
	Regexp       string
}

type SeekResult struct {
	Uuid         string
	Success      bool
	CrawlVersion string
	IPFSHash     string
	Host         string
	Seed         string
}

func main() {

	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	http.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		hash, ok := r.URL.Query()["resulthash"]
		if !ok {
			fmt.Println("Failed to get resulthash param.")
			return
		}
		rcl, err := sh.Cat(hash[0])
		if err != nil {
			fmt.Println("Failed to download hash.")
			return
		}

		_, err = io.Copy(w, rcl)
		if err != nil {
			fmt.Println("Failed to copy to HTTP response.")
			return
		}

	})

	http.HandleFunc("/enqueue", func(w http.ResponseWriter, r *http.Request) {
		// enqueue the data
		err := r.ParseForm()
		if err != nil {
			fmt.Println("Failed to parse form.")
			return
		}

		// Basic input validation
		version := r.Form.Get("crawl_version")
		if version == "" {
			http.Error(w, "Crawl version cannot be empty.", http.StatusBadRequest)
			return
		}

		regexpInput := r.Form.Get("regexp")
		// Simple regex check for allowed characters in regex (optional based on your requirements)
		if !isValidRegexp(regexpInput) {
			http.Error(w, "Invalid regular expression.", http.StatusBadRequest)
			return
		}

		attempts := r.Form.Get("attempts")
		depth := r.Form.Get("depth")

		// Ensure that attempts and depth are valid numbers
		atmpt, err := strconv.Atoi(attempts)
		if err != nil || atmpt <= 0 || atmpt > 20 {
			http.Error(w, "Attempts must be a positive number and less than 20.", http.StatusBadRequest)
			return
		}

		depthInt, err := strconv.Atoi(depth)
		if err != nil || depthInt < 0 || depthInt > 15 {
			http.Error(w, "Depth must be a non-negative number and less than 15.", http.StatusBadRequest)
			return
		}

		sk := &SeekQuery{Uuid: uuid.NewString(),
			Attempts:     atmpt,
			Regexp:       regexpInput,
			Depth:        depthInt,
			CrawlVersion: version}
		publish_sk(sk)

		fmt.Fprintf(w, `<p>Query is published and results are collected here.</p>
		<p>Redirecting in 5 seconds...</p>
		<meta http-equiv="refresh" content="5; url=/results/%s.html" />
		<a href="/results/%s.html">If you are not redirected, click here.</a>`, sk.Uuid, sk.Uuid)

	})

	go monitor_results("crawlseedresults")
	go monitor_queries("crawlseedqueries")
	//go monitor_stale_files("results")

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		panic(err)
	}

}

func isValidRegexp(input string) bool {
	// Adjust this regex based on what you consider a valid regex input
	// This is a very simple check for demonstration purposes only
	re := regexp.MustCompile(`^.*?$`)
	return re.MatchString(input)
}

func monitor_results(topic string) {
	for {
		sub, _ := sh.PubSubSubscribe(topic)
		r, _ := sub.Next()

		sr := &SeekResult{}
		err := json.Unmarshal(r.Data, sr)
		if err != nil {
			fmt.Println("Error unmarshaling", err)
			continue
		}

		//fmt.Println("Got result", string(sr.Output))
		fmt.Println("Got result", sr.Uuid)



		// Lock the file for writing
                fileMutex.Lock()
		if sr.Success {

			f, err := os.OpenFile(fmt.Sprintf("results/%s.html", sr.Uuid),
				os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println("Error opening file for success append", err)
				continue
			}
			defer f.Close()
			if _, err := f.WriteString(fmt.Sprintf("<p>%s: %s %s <a href=\"/result?resulthash=%s\">%s</a></p>\n", sr.Host, sr.CrawlVersion, sr.Seed, sr.IPFSHash, sr.IPFSHash)); err != nil {
				fmt.Println("Error writing to result :", err)
			}

			err = sh.Pin(sr.IPFSHash)
			if err != nil {
				fmt.Println("Failed to pin output to IPFS.")
			}

		} else {
			f, err := os.OpenFile(fmt.Sprintf("results/%s.html", sr.Uuid),
				os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println("Error opening file for append", err)
				continue
			}
			defer f.Close()
			if _, err := f.WriteString(fmt.Sprintf("<p>%s: %s gave up</a></p>\n", sr.Host, sr.CrawlVersion)); err != nil {
				fmt.Println("Error writing to fail result:", err)
			}
		}
		fileMutex.Unlock()

	}

}

func publish_sk(sk *SeekQuery) error {

	payload, err := json.Marshal(sk)
	if err != nil {
		fmt.Println("Failed to marshal seek query.")
		return err
	}

	resp := sh.PubSubPublish("crawlseedqueries", string(payload))
	if resp != nil {
		fmt.Println(fmt.Sprintf("err sent: %s", resp))
		return resp
	} else {
		fmt.Println(string(payload))
		return nil
	}

}

func monitor_queries(topic string) {
	for {
		sub, err := sh.PubSubSubscribe(topic)
		if err != nil {
			fmt.Println("Error subscribing to topic", err)
			panic(err)
		}
		r, _ := sub.Next()

		sr := &SeekQuery{}
		err = json.Unmarshal(r.Data, sr)
		if err != nil {
			fmt.Println("Error unmarshaling", err)
			continue
		}

		ioutil.WriteFile(fmt.Sprintf("results/%s.html", sr.Uuid), []byte(fmt.Sprintf("<p>Version: %s</p></p>", sr.CrawlVersion)), 0644)
	}
}

func monitor_stale_files(dir string) {
	for {
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			filepath := fmt.Sprintf("%s/%s",dir,file.Name())
			if strings.HasSuffix(filepath,".html") {
				// Check if the .done file exists
				doneFilePath := filepath + ".done"
				if _, err := os.Stat(doneFilePath); os.IsNotExist(err) {
					// Check the modification time of the HTML file
					if file.ModTime().Before(oneMinuteAgo) {

						f, err := os.OpenFile((filepath),
							os.O_APPEND|os.O_WRONLY, 0644)
						if err != nil {
							fmt.Println("Error opening file for append", err)
							continue
						}
						defer f.Close()
						f.WriteString("This job never received any results from workers. Perhaps they are offline or were during the request.")



						fdone, err := os.OpenFile((doneFilePath),
							os.O_WRONLY, 0644)
						if err != nil {
							fmt.Println("Error opening done file", err)
							continue
						}
						fdone.WriteString("never got results")
						defer fdone.Close()



					}
				}
			}
		}

	}


	 time.Sleep(30 * time.Second)

}
