package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func urlFuzzScanner(url string, directoryList []string) {
	// open the text file directoryList and read the lines in it
	file, err := os.Open(directoryList[0])
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	// read the lines in the text file
	scanner := bufio.NewScanner(file)
	// loop through the lines in the text file
	for scanner.Scan() {
		// get the line in the text file
		line := scanner.Text()
		// append the line to the url
		test_url := url + "/" + line
		// make the request to the url
		resp, err := http.Get(test_url)
		if err != nil {
			fmt.Println(err)
		}
		// check the response status code
		if resp.StatusCode == 200 {
			// if the response status code is 200, print the url
			fmt.Println(test_url + " - 200")
		} else {
			// if the response status code is not 200, print the url and the response status code
			fmt.Println(test_url + " " + resp.Status)
		}
	}
}

func main() {
	// get url parameter from name "url" in the command line
	url := os.Args[1]
	// get directoryList parameter from name "directoryList" in the command line
	directoryList := strings.Split(os.Args[2], ",")
	// check the directory list, if the found in the url
	urlFuzzScanner(url, directoryList)
}
