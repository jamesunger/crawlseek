package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"os"
	"net"
	"regexp"
	"strconv"
	"strings"
	"crawlexec"
	"bytes"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/redis/go-redis/v9"
	"context"
)

var sh = shell.NewShell("localhost:5001")
var rdb *redis.Client
var ctx = context.Background()

func init() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379",redisHost),
		Password: redisPassword,
		Username: "crawlseek",
		DB:       0, // use default DB
	})
}

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

func getClientIP(r *http.Request) string {
    // 1. Check X-Real-IP header
    ip := r.Header.Get("X-Real-IP")
    if ip != "" {
        return ip
    }

    // 2. Check X-Forwarded-For header
    ip = r.Header.Get("X-Forwarded-For")
    if ip != "" {
        // X-Forwarded-For may contain multiple IPs
        // The first IP is the original client IP
        ips := strings.Split(ip, ",")
        if len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }

    // 3. Get IP from RemoteAddr
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        // RemoteAddr may not contain port
        return r.RemoteAddr
    }
    return ip
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

		remoteIP := getClientIP(r)
		// Verify the token with hCaptcha API
		resp, err := http.PostForm("https://hcaptcha.com/siteverify", url.Values{
			"secret":   {hCaptchaSecret},
			"remoteip":   {remoteIP},
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
			ResultsURL: fmt.Sprintf("results/%s.json", sk.Uuid),
		}

		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/results/", func(w http.ResponseWriter, r *http.Request) {
		uuid := strings.TrimPrefix(r.URL.Path, "/results/")
		uuid = strings.TrimSuffix(uuid, ".json")

		val, err := rdb.Get(ctx, uuid).Result()
		if err != nil {
			http.Error(w, "Result not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(val))
	})

	go monitor_results("crawlseedresults")
	go monitor_queries("crawlseedqueries")

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		panic(err)
	}
}

func processResultData(data []byte) {
	sr := &SeekResult{}
	err := json.Unmarshal(data, sr)
	if err != nil {
		fmt.Println("Error unmarshaling", err)
		return
	}

	fmt.Println("Got result", sr.Uuid)

	// Read existing entries or create new array
	var entries []ResultEntry
	val, err := rdb.Get(ctx, sr.Uuid).Result()
	if err == nil {
		json.Unmarshal([]byte(val), &entries)
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

	// Write back to redis
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	err = rdb.Set(ctx, sr.Uuid, jsonData, 0).Err()
	if err != nil {
		fmt.Println("Error writing to redis:", err)
		return
	}
}

func monitor_results(topic string) {
	sub, err := sh.PubSubSubscribe(topic)
	if err != nil {
		fmt.Println("Error subscribing to results topic", err)
		panic(err)
	}

	for {

		r, _ := sub.Next()
		if err != nil {
			fmt.Println("Error getting next message", err)
			sub, err = sh.PubSubSubscribe(topic)
			if err != nil {
				fmt.Println("Error resubscribing to results topic", err)
				panic(err)
			}
			continue
		}

		// Launch a goroutine to process the data
		go processResultData(r.Data)
	}
	sub.Cancel()
}

func processQueryData(data []byte) {
	sk := &SeekQuery{}
	err := json.Unmarshal(data, sk)
	if err != nil {
		fmt.Println("Error unmarshaling", err)
		return
	}


	err = rdb.Set(ctx, sk.Uuid, []byte("[]"), 0).Err()
	if err != nil {
		fmt.Println("Error writing to redis:", err)
	}


	findCrawlSeed(sk)

}



func findCrawlSeed(sk *SeekQuery) {
        hostname, _ := os.Hostname()

	sr := &SeekResult{}
        sr.Uuid = sk.Uuid
        sr.Host = hostname
        sr.CrawlVersion = sk.CrawlVersion

        foundSeed,reportBytes := crawlexec.SearchCrawl(sk.CrawlVersion, sk.Depth, sk.Regexp, sk.Attempts)

        if foundSeed == "" {
                fmt.Println("nomatch")
        } else {
                sr.Success = true
                sr.Seed = foundSeed
        }

        r := bytes.NewReader(reportBytes)


        hash, err := sh.Add(r)
        if err != nil {
        fmt.Println("Failed to add output to IPFS.")
                return
        }

        err = sh.Pin(hash)
        if err != nil {
                fmt.Println("Failed to pin output to IPFS.")
                return
        }

        sr.IPFSHash = hash

        response, err := json.Marshal(sr)
        if err != nil {
                fmt.Println("Error marshaling response", err)
                return
        }

        resp := sh.PubSubPublish("crawlseedresults", string(response))
        if resp != nil {
                fmt.Println("Sent", resp)
        }

}




func monitor_queries(topic string) {
	sub, err := sh.PubSubSubscribe(topic)
	if err != nil {
		fmt.Println("Error subscribing to queries topic", err)
		panic(err)
	}

	for {

		r, err := sub.Next()
		if err != nil {
			fmt.Println("Error getting next message", err)
			sub, err = sh.PubSubSubscribe(topic)
			if err != nil {
				fmt.Println("Error resubscribing queries to topic", err)
				panic(err)
			}
			continue
		}
		// Launch a goroutine to process the data
		go processQueryData(r.Data)
	}
	sub.Cancel()
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

