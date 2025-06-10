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

type noCache struct {
    http.Handler
}

func (n *noCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache") // HTTP 1.0
    w.Header().Set("Expires", "0") // Proxies
    n.Handler.ServeHTTP(w, r)
}



func main() {


        fileServer := http.FileServer(http.Dir("."))
        noCacheFS := &noCache{fileServer}
        http.Handle("/", noCacheFS)
		
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

		// Set response header to application/json
		w.Header().Set("Content-Type", "application/json")
		
		// Create response object
		response := struct {
			ResultsURL string `json:"results_url"`
		}{
			ResultsURL: fmt.Sprintf("results/%s.html", sk.Uuid),
		}
		
		// Encode and send JSON response
		json.NewEncoder(w).Encode(response)
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
			if _, err := f.WriteString(fmt.Sprintf("%s: %s %s %s\n", sr.Host, sr.CrawlVersion, sr.Seed, sr.IPFSHash)); err != nil {
				fmt.Println("Error writing to result :", err)
			}

			err = sh.Pin(sr.IPFSHash)
			if err != nil {
				fmt.Println("Failed to pin output to IPFS.")
			}
			f.Close()

		} else {
			f, err := os.OpenFile(fmt.Sprintf("results/%s.html", sr.Uuid),
				os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println("Error opening file for append", err)
				continue
			}
			if _, err := f.WriteString(fmt.Sprintf("<p>%s: %s gave up</p>\n", sr.Host, sr.CrawlVersion)); err != nil {
				fmt.Println("Error writing to fail result:", err)
			}
			f.Close()
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

		resultsURL := fmt.Sprintf("Generating %s...\n", sr.Uuid)
		ioutil.WriteFile(fmt.Sprintf("results/%s.html", sr.Uuid), []byte(resultsURL), 0644);
	}
}

