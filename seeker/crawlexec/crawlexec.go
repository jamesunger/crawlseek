package crawlexec

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"regexp"
)

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

func SearchCrawl(crawlversion string, depth int, regexpstr string, attempts int) (string,[]byte) {

	for i := 1; i <= attempts; i++ {
		fmt.Println("Attempt", i)

		seed, _ := randint64()
		seedstr := fmt.Sprintf("%d", seed)

		if !isValidInput(crawlversion) || !isValidInput(seedstr) || !isValidInput(fmt.Sprintf("%d", depth)) {
			fmt.Println("Input validation failed for attempt:", i)
			break
		}

		cmd := exec.Command(fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/util/fake_pty", crawlversion), fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source/crawl", crawlversion), "-script", "seed_explorer.lua", "-seed", seedstr, "-depth", fmt.Sprintf("%d", depth))
		fmt.Println("Command execution: ", cmd)
		cmd.Dir = fmt.Sprintf("/crawl/%s/crawl/crawl-ref/source", crawlversion)

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

		match, _ := regexp.Match(regexpstr, out)
		if match {
			return seedstr, out
		}

	}

	return "", []byte("")

}
