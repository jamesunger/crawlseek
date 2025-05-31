
package main

import (
	"fmt"
	"os/exec"
	"crypto/rand"
	"encoding/binary"
	"os"
	"bytes"
	"io"
	"encoding/json"
	"regexp"
	//ipfsapi "github.com/ipfs/go-ipfs-api"
	shell "github.com/ipfs/go-ipfs-api"
//	"flag"
)
type SeekQuery struct {
        Uuid string
        Attempts int
        Depth int
        CrawlVersion string
        Regexp string

}

type SeekResult struct {
        Uuid string
        Success bool
        CrawlVersion string
        IPFSHash string
        Host string
        Seed string
}



var sh = shell.NewShell("localhost:5001")
var hostname,_ = os.Hostname()

func main() {

	subscribe_topic("crawlseedqueries")

}

func randint64() (uint64, error) {
    var b [8]byte
    if _, err := rand.Read(b[:]); err != nil {
        return 0, err
    }
    //return int64(math.Abs(float64(binary.LittleEndian.Uint64(b[:])))), nil
    return uint64(binary.LittleEndian.Uint64(b[:])), nil
}


// Validate that the input string does not contain dangerous characters
func isValidInput(input string) bool {
        // Define a regex pattern to disallow dangerous characters (e.g., ; & | $ ` \)
        pattern := `^[\w\-\.]+$` // allows only alphanumeric characters, hyphens, dots, and underscores
        return regexp.MustCompile(pattern).MatchString(input)
}



func subscribe_topic(topic string) {
	for {
		sub, _ := sh.PubSubSubscribe(topic)
		r, _ := sub.Next()

		sk := &SeekQuery{}
		sr := &SeekResult{}

		err := json.Unmarshal(r.Data,sk)
		sr.Uuid = sk.Uuid
		if err != nil {
			fmt.Println("error unmarshaling",err)
			continue
		}





		depth := fmt.Sprintf("%d",sk.Depth)
		sr.Host = hostname
		sr.CrawlVersion = sk.CrawlVersion

		for i := 1; i <= sk.Attempts; i++ {
			fmt.Println("Attempt",i)

			seed,_ := randint64()
			seedstr := fmt.Sprintf("%d",seed)
			sr.Seed = seedstr

                        // Validate input
                        if !isValidInput(sr.CrawlVersion) || !isValidInput(seedstr) || !isValidInput(depth) {
                                fmt.Println("Input validation failed for attempt:", i)
                                break;
                        }


			// to get version ./crawl -version | head -n 1 | cut -f 3 -d ' '
			//cmd := exec.Command("/home/junger/crawl/crawl-ref/source/util/fake_pty","/home/junger/crawl/crawl-ref/source/crawl-debug", "-script", "seed_explorer.lua", "-seed", "random", "-depth", depth, "-artefacts")
			//cmdstring := fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/util/fake_pty /crawl/%s/crawl-ref/source/crawl -script seed_explorer.lua -seed %s -depth %s",sr.CrawlVersion, sr.CrawlVersion, seedstr,depth)
			//fmt.Println("Cmdring:",cmdstring)
			cmd := exec.Command(fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/util/fake_pty",sr.CrawlVersion),fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/crawl", sr.CrawlVersion), "-script", "seed_explorer.lua", "-seed", seedstr, "-depth", depth)
			fmt.Println("Cmd path:",cmd.Path)
			cmd.Dir = fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source", sr.CrawlVersion)

			cmd.Env = []string{ "TERM=vt100" }
			stderr,err := cmd.StderrPipe()
			if err != nil {
				fmt.Println("Error failed to open pipe", err)
				continue;
			}
			if err := cmd.Start(); err != nil {
				fmt.Println("Failed to read err pipe:", err)
				continue
			}

			out,_ := io.ReadAll(stderr)


			fmt.Println(string(out))

			if match,_ := regexp.Match(sk.Regexp,out); match {
				fmt.Println("matched")
				sr.Success = true
			} else {
				fmt.Println("nomatch")
				continue;
			}
			r := bytes.NewReader(out)

			hash,err := sh.Add(r)
			if err != nil {
				fmt.Println("Failed to add output to IPFS.")
				break;
			}
			err = sh.Pin(hash)
			if err != nil {
				fmt.Println("Failed to pin output to IPFS.")
				break;
			}

			fmt.Println("IPFS HASH:", hash)
			sr.IPFSHash = hash
			break;
		}


		response,err := json.Marshal(sr)
		if err != nil {
			fmt.Println("Error marshaling response", err)
		}
		resp := sh.PubSubPublish("crawlseedresults",string(response))
		if resp != nil {
			fmt.Println("Sent",resp)
		}

		//time.Sleep(10 * time.Second)
	}

}

