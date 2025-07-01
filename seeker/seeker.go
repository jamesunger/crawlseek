package main

import (
	"fmt"
	"os"
	"encoding/json"
	"bytes"
	"crawlexec"

	shell "github.com/ipfs/go-ipfs-api"
)


var sh = shell.NewShell("localhost:5001")

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


func processSeekMessage(data []byte) {
	hostname, _ := os.Hostname()

	sk := &SeekQuery{}
	sr := &SeekResult{}
        err := json.Unmarshal(data, sk)
        if err != nil {
               	fmt.Println("error unmarshaling", err)
               	return
        }

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


func subscribe_topic(topic string) {
	sub, err := sh.PubSubSubscribe(topic)
	if err != nil {
		fmt.Println("Failed to subscribe to topic:", err)
		panic(err)
	}

	for {
		r, err := sub.Next()
		if err != nil {
			fmt.Println("Failed to get next message:", err)
			sub, err = sh.PubSubSubscribe(topic)
			if err != nil {
				fmt.Println("Failed to resubscribe to topic:", err)
				panic(err)
			}
			continue
		}

		// Process message in a goroutine


		go processSeekMessage(r.Data)

		// Close the subscription after getting the message
	}
	sub.Cancel()
}

func main() {
	subscribe_topic("crawlseedqueries")
}

