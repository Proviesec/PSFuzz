package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func urlFuzzScanner(url string, directoryList []string, showStatus string) {
	// open the text file directoryList and read the lines in it
	file, err := os.Open(directoryList[0])
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	// read the lines in the text file
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		// get the line in the text file
		line := scanner.Text()
		// check if the line is empty
		if line == "" {
			continue
		}
		// test the lin a go rountine
		go testUrl(url, line, showStatus)
	}
}

func testUrl(url string, line string, showStatus string) {
	// create a new http client
	client := &http.Client{}
	// create a new request
	url = url + "/" + line
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	
	// set the user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
	
	// make the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	
	// check the response status code
	if resp.StatusCode == 200 {
		// if the response status code is 200, print the url
		fmt.Println(url + " - 200")
	} else if showStatus == "true" {
		// if the response status code is not 200, print the url and the response status code
		fmt.Println(url + " " + resp.Status)
	}
}

func main() {
	// get url parameter from name "url" in the command line
	url := os.Args[1]
	// get directoryList parameter from name "directoryList" in the command line
	directoryList := strings.Split(os.Args[2], ",")
	start := time.Now()
	// get status parameter from the command lline
	status := os.Args[3]

	// check the directory list, if the found in the url
	urlFuzzScanner(url, directoryList, status)

	time.Sleep(30 * time.Second)
	elapsed := time.Since(start)
	log.Println(elapsed)
}
