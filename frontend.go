package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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

type ResultEntry struct {
	Host         string `json:"host"`
	CrawlVersion string `json:"crawl_version"`
	Seed         string `json:"seed"`
	IPFSHash     string `json:"ipfshash"`
	Status       string `json:"status,omitempty"` // Added status field
}

type noCache struct {
	http.Handler
}

func (n *noCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")    // HTTP 1.0
	w.Header().Set("Expires", "0")          // Proxies
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
		err := r.ParseForm()
		if err != nil {
			fmt.Println("Failed to parse form.")
			return
		}

		// Verify hCaptcha token
		hCaptchaResponse := r.Form.Get("h-captcha-response")
		if hCaptchaResponse == "" {
			http.Error(w, "Captcha verification failed.", http.StatusBadRequest)
			return
		}

		// Get hCaptcha secret key from environment variable
		hCaptchaSecret := os.Getenv("HCAPTCHA_SECRET_KEY")
		if hCaptchaSecret == "" {
			http.Error(w, "Server configuration error.", http.StatusInternalServerError)
			return
		}

		// Verify the token with hCaptcha API
		resp, err := http.PostForm("https://hcaptcha.com/siteverify", url.Values{
			"secret":   {hCaptchaSecret},
			"response": {hCaptchaResponse},
		})
		if err != nil {
			http.Error(w, "Failed to verify captcha.", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var captchaResult struct {
			Success bool `json:"success"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&captchaResult); err != nil {
			http.Error(w, "Failed to parse captcha verification response.", http.StatusInternalServerError)
			return
		}

		if !captchaResult.Success {
			http.Error(w, "Invalid captcha.", http.StatusBadRequest)
			return
		}

		version := r.Form.Get("crawl_version")
		if version == "" {
			http.Error(w, "Crawl version cannot be empty.", http.StatusBadRequest)
			return
		}

		regexpInput := r.Form.Get("regexp")
		if !isValidRegexp(regexpInput) {
			http.Error(w, "Invalid regular expression.", http.StatusBadRequest)
			return
		}

		attempts := r.Form.Get("attempts")
		depth := r.Form.Get("depth")

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

		sk := &SeekQuery{
			Uuid:         uuid.NewString(),
			Attempts:     atmpt,
			Regexp:       regexpInput,
			Depth:        depthInt,
			CrawlVersion: version,
		}
		publish_sk(sk)

		w.Header().Set("Content-Type", "application/json")
		response := struct {
			ResultsURL string `json:"results_url"`
		}{
			ResultsURL: fmt.Sprintf("results/%s.html", sk.Uuid),
		}

		json.NewEncoder(w).Encode(response)
	})

	go monitor_results("crawlseedresults")
	go monitor_queries("crawlseedqueries")

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		panic(err)
	}
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

		fileMutex.Lock()
		filename := fmt.Sprintf("results/%s.html", sr.Uuid)

		// Read existing entries or create new array
		var entries []ResultEntry
		content, err := ioutil.ReadFile(filename)
		if err == nil {
			json.Unmarshal(content, &entries)
		}

		// Create new entry
		newEntry := ResultEntry{
			Host:         sr.Host,
			CrawlVersion: sr.CrawlVersion,
			Seed:         sr.Seed,
		}

		if sr.Success {
			newEntry.IPFSHash = sr.IPFSHash
			newEntry.Status = "success"
			err = sh.Pin(sr.IPFSHash)
			if err != nil {
				fmt.Println("Failed to pin output to IPFS.")
			}
		} else {
			newEntry.Status = "gave up"
		}

		entries = append(entries, newEntry)

		// Write back to file
		jsonData, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			fileMutex.Unlock()
			continue
		}

		err = ioutil.WriteFile(filename, jsonData, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			fileMutex.Unlock()
			continue
		}

		fileMutex.Unlock()
	}
}

func isValidRegexp(input string) bool {
	re := regexp.MustCompile(`^.*?$`)
	return re.MatchString(input)
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

		ioutil.WriteFile(fmt.Sprintf("results/%s.html", sr.Uuid), []byte("[]"), 0644)
	}
}

