package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const MAX_CONCURRENT_JOBS = 30

type result struct {
	sumValue      int
	multiplyValue int
}

var statuscount = map[string]int{}

func urlFuzzScanner(url string, directoryList []string, showStatus string, resultChan chan result) {
	// open the text file directoryList and read the lines in it
	file, err := os.Open(directoryList[0])
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	// read the lines in the text file
	scanner := bufio.NewScanner(file)
	waitChan := make(chan struct{}, MAX_CONCURRENT_JOBS)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		// check if the line is empty
		if line == "" {
			continue
		}
		// get the line in the text file

		waitChan <- struct{}{}
		count++
		go func(count int, url string, line string, showStatus string, resultChan chan result) {
			testUrl(url, line, showStatus, resultChan)
			job(count)
			<-waitChan
		}(count, url, line, showStatus, resultChan)
	}
}

func testUrl(url string, line string, showStatus string, resultChan chan result) {
	// create a new http client
	// client := &http.Client{}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	// create a new request
	url = url + "/" + line
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	// set the user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
	// make the request w
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	statuscount[resp.Status] = statuscount[resp.Status] + 1
	// check the response status code
	if resp.StatusCode == 200 {
		// if the response status code is 200, print the url
		fmt.Println(url + " - 200")
	} else if showStatus == "true" {
		// if the response status code is not 200, print the url and the response status code
		fmt.Println(url + " " + resp.Status)
	}
	res := result{sumValue: 1, multiplyValue: resp.StatusCode}
	resultChan <- res
	// return the status code of the response
}

func main() {
	// get url parameter from name "url" in the command line
	url := os.Args[1]
	resultChan := make(chan result, 1)
	// get directoryList parameter from name "directoryList" in the command line
	directoryList := strings.Split(os.Args[2], ",")
	// get status parameter from the command lline
	status := os.Args[3]

	// check the directory list, if the found in the url
	urlFuzzScanner(url, directoryList, status, resultChan)

	res := <-resultChan
	fmt.Printf("Sum Value: %d\n", res.sumValue)
	fmt.Printf("Multiply Value: %d\n", res.multiplyValue)
	fmt.Println(statuscount) // map[string]int
	close(resultChan)
}

func job(index int) {
	//fmt.Println(index, "begin doing something")
	time.Sleep(time.Duration(rand.Intn(10) * int(time.Second)))
	//	fmt.Println(index, "done")
}
