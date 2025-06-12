package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	shell "github.com/ipfs/go-ipfs-api"
)

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

var sh = shell.NewShell("localhost:5001")

func randint64() (uint64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return uint64(binary.LittleEndian.Uint64(b[:])), nil
}

func isValidInput(input string) bool {
	pattern := `^[\w\-\.]+$` // allows only alphanumeric characters, hyphens, dots, and underscores
	return regexp.MustCompile(pattern).MatchString(input)
}

func processMessage(r *shell.Message) {
	hostname, _ := os.Hostname()
	
	sk := &SeekQuery{}
	sr := &SeekResult{}

	err := json.Unmarshal(r.Data, sk)
	if err != nil {
		fmt.Println("error unmarshaling", err)
		return
	}

	sr.Uuid = sk.Uuid
	sr.Host = hostname
	sr.CrawlVersion = sk.CrawlVersion

	for i := 1; i <= sk.Attempts; i++ {
		fmt.Println("Attempt", i)

		seed, _ := randint64()
		seedstr := fmt.Sprintf("%d", seed)
		sr.Seed = seedstr

		if !isValidInput(sr.CrawlVersion) || !isValidInput(seedstr) || !isValidInput(fmt.Sprintf("%d", sk.Depth)) {
			fmt.Println("Input validation failed for attempt:", i)
			break
		}

		cmd := exec.Command(fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/util/fake_pty", sr.CrawlVersion), fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/crawl", sr.CrawlVersion), "-script", "seed_explorer.lua", "-seed", seedstr, "-depth", fmt.Sprintf("%d", sk.Depth))
		fmt.Println("Command execution: ", cmd)
		cmd.Dir = fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source", sr.CrawlVersion)

		cmd.Env = []string{"TERM=vt100"}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Println("Error failed to open pipe", err)
			continue
		}
		if err := cmd.Start(); err != nil {
			fmt.Println("Failed to read err pipe:", err)
			continue
		}

		out, _ := io.ReadAll(stderr)

		match, _ := regexp.Match(sk.Regexp, out)
		if match {
			fmt.Println("matched")
			sr.Success = true
		} else {
			fmt.Println("nomatch")
			continue
		}
		r := bytes.NewReader(out)

		hash, err := sh.Add(r)
		if err != nil {
			fmt.Println("Failed to add output to IPFS.")
			break
		}
		err = sh.Pin(hash)
		if err != nil {
			fmt.Println("Failed to pin output to IPFS.")
			break
		}

		sr.IPFSHash = hash
		break
	}

	response, err := json.Marshal(sr)
	if err != nil {
		fmt.Println("Error marshaling response", err)
	}
	resp := sh.PubSubPublish("crawlseedresults", string(response))
	if resp != nil {
		fmt.Println("Sent", resp)
	}
}

func subscribe_topic(topic string) {
	for {
		sub, err := sh.PubSubSubscribe(topic)
		if err != nil {
			fmt.Println("Failed to subscribe to topic:", err)
			continue
		}

		r, err := sub.Next()
		if err != nil {
			fmt.Println("Failed to get next message:", err)
			sub.Cancel()
			continue
		}

		// Process message in a goroutine
		go processMessage(r)

		// Close the subscription after getting the message
		sub.Cancel()
	}
}

func main() {
	subscribe_topic("crawlseedqueries")
}

