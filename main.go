package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const MAX_CONCURRENT_JOBS = 2

type result struct {
	sumValue      int
	multiplyValue int
}

var mutex = &sync.Mutex{}

var statuscount = map[string]int{}

func lineCounter(r io.Reader) int {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count

		case err != nil:
			return count
		}
	}
}

func urlFuzzScanner(url string, directoryList []string, showStatus string) {
	// open the text file directoryList and read the lines in it
	file, err := os.Open(directoryList[0])
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	// read the lines in the text file

	file_lines, err := os.OpenFile(directoryList[0], os.O_RDONLY, 0444)
	if err != nil {
		log.Fatal(err)
	}
	defer file_lines.Close()
	count_lines := lineCounter(file_lines)

	concurrent := make(chan int, MAX_CONCURRENT_JOBS)

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		percent := (count * 100 / count_lines)
		fill := strings.Repeat("x", percent) + strings.Repeat("-", 100-percent)
		_, _ = fmt.Fprint(os.Stdout, "\r[")
		_, _ = fmt.Fprintf(os.Stdout, "%s]", fill)
		p := int(count * 100 / (count_lines + 1))
		_, _ = fmt.Fprintf(os.Stdout, "\t%d %%", p)

		word := scanner.Text()
		// check if the line is empty
		if word == "" {
			continue
		}
		concurrent <- 1
		count++
		go func(count int, url string, word string, showStatus string) {
			testUrl(url, word, showStatus)
			<-concurrent
		}(count, url, word, showStatus)
	}
	return
}

func testUrl(url string, word string, showStatus string) {
	// create a new http client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// find the wildcard in the url
	if strings.Contains(url, "#PSFUZZ#") {
		url = strings.Replace(url, "#PSFUZZ#", word, 1)
	} else {
		url = url + word
	}

	// create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	// set the user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
	// define the request with a timeout of 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// make the request
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		fmt.Println(err)
	}

	mutex.Lock()
	statuscount[resp.Status] = statuscount[resp.Status] + 1
	mutex.Unlock()

	// check the response status code
	if resp.StatusCode == 200 {
		// if the response status code is 200, print the url
		fmt.Fprint(os.Stdout, "\r"+url+" - 200 "+strings.Repeat(" ", 100)+"\n")
	} else if showStatus == "true" {
		// if the response status code is not 200, print the url and the response status code
		fmt.Fprint(os.Stdout, "\r"+url+" "+resp.Status+strings.Repeat(" ", 100)+"\n")
	}
}

func main() {
	// get url parameter from name "url" in the command line
	url := flag.String("url", "", "URL")
	// get directoryList parameter from name "directoryList" in the command line
	dirlist := flag.String("dl", "", "Directory List")
	// get status parameter from the command lline
	status := flag.String("status", "false", "show status")
	flag.Parse()
	directoryList := strings.Split(*dirlist, ",")

	// check the directory list, if the found in the url
	urlFuzzScanner(*url, directoryList, *status)
	fmt.Fprint(os.Stdout, "\n")
	fmt.Println(statuscount) // map[string]int
}
