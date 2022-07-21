package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

var url string
var dirlist = flag.String("dirlist", "", "Directory List")
var showStatus string
var redirect string
var concurrency int
var output string
var filterStatusCode string
var filterStatusCodeList []string
var filterStatusNot string
var filterStatusNotList []string

func init() {
	// get url parameter from name "url" in the command line
	flag.StringVar(&url, "url", "", "URL")
	flag.StringVar(&url, "u", "", "URL")

	// get directoryList parameter from name "directoryList" in the command line
	flag.StringVar(dirlist, "d", "", "Directory List")

	// get status parameter from the command lline
	flag.StringVar(&showStatus, "showStatus", "false", "show status")
	flag.StringVar(&showStatus, "s", "false", "show status")

	// get concurrency parameter from the command line
	flag.StringVar(&redirect, "redirect", "true", "redirect")
	flag.StringVar(&redirect, "3", "true", "redirect")

	// get concurrency parameter from the command line
	flag.IntVar(&concurrency, "concurrency", 1, "concurrency")
	flag.IntVar(&concurrency, "c", 1, "concurrency")

	// get output parameter from the command line

	flag.StringVar(&output, "output", "", "output")
	flag.StringVar(&output, "o", "", "output")

	// get list of show status codes from the command line
	flag.StringVar(&filterStatusCode, "filterStatusCode", "", "Show only this status code")
	flag.StringVar(&filterStatusCode, "fsc", "", "Show only this status code")

	// get list of status code to be filtered from the command line
	flag.StringVar(&filterStatusNot, "filterStatusCodeNot", "", "Show not this status code")
	flag.StringVar(&filterStatusNot, "fscn", "", "Show not this status code")

	flag.Parse()

	filterStatusCodeList = strings.Split(filterStatusCode, ",")
	filterStatusNotList = strings.Split(filterStatusNot, ",")

	// If the identified URL has neither http or https infront of it. Create both and scan them.
	if !strings.Contains(url, "http://") && !strings.Contains(url, "https://") {
		url = "https://" + url
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func GetResponseDetails(response *http.Response) (string, int) {
	// Get the response body as a string
	dataInBytes, _ := ioutil.ReadAll(response.Body)
	pageContent := string(dataInBytes)

	// Find a substr
	titleStartIndex := strings.Index(pageContent, "<title>")
	if titleStartIndex == -1 {
		return "No title element found", len(pageContent)
	}
	// <title> = length = 7
	titleStartIndex += 7

	// Find the index of the closing tag
	titleEndIndex := strings.Index(pageContent, "</title>")
	if titleEndIndex == -1 {
		return "No closing tag for title found.", len(pageContent)
	}
	pageTitle := string([]byte(pageContent[titleStartIndex:titleEndIndex]))
	return "Page title:" + pageTitle, len(pageContent)
}

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

func urlFuzzScanner(directoryList []string) {
	// open the text file directoryList and read the lines in it
	file, err := os.Open(directoryList[0])
	if err != nil {
		fmt.Fprint(os.Stdout, "\r"+err.Error()+strings.Repeat(" ", 100)+"\n")
	}
	defer file.Close()
	// read the lines in the text file

	file_lines, err := os.OpenFile(directoryList[0], os.O_RDONLY, 0444)
	if err != nil {
		fmt.Fprint(os.Stdout, "\r"+err.Error()+strings.Repeat(" ", 100)+"\n")
	}
	defer file_lines.Close()
	count_lines := lineCounter(file_lines)
	if concurrency <= 0 {
		concurrency = MAX_CONCURRENT_JOBS
	}

	concurrent := make(chan int, concurrency)

	scanner := bufio.NewScanner(file)
	count := 0

	if output == "" {
		output = "output.txt"
	} else {
		output = output + ".txt"
	}

	file_create, err := os.Create(output) // Truncates if file already exists, be careful!
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file_create.Close()

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
		go func(count int, url string, showStatus string) {
			// find the wildcard in the url
			if strings.Contains(url, "#PSFUZZ#") {
				url = strings.Replace(url, "#PSFUZZ#", word, 1)
			} else {
				url = url + word
			}
			// write the result to the file
			testUrl(url, showStatus, file_create, false)
			<-concurrent
		}(count, url, showStatus)
	}
	return
}

func testUrl(url string, showStatus string, file_create *os.File, redirected bool) {

	// create a new http client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	// create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprint(os.Stdout, "\r"+err.Error()+strings.Repeat(" ", 100)+"\n")
		return
	}
	// set the user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36 (zz99)")
	// define the request with a timeout of 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// make the request
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		fmt.Fprint(os.Stdout, "\r"+err.Error()+strings.Repeat(" ", 100)+"\n")
		return
	}

	mutex.Lock()
	statuscount[resp.Status] = statuscount[resp.Status] + 1
	mutex.Unlock()

	// create output string variable
	var outputString string
	if (contains(filterStatusCodeList, strconv.Itoa(resp.StatusCode)) || showStatus == "true") && !contains(filterStatusNotList, strconv.Itoa(resp.StatusCode)) {
		title, length := GetResponseDetails(resp)
		if strings.Contains(title, "404") {
			title = title + " -- possibile a 404"
		}
		if redirected {
			outputString = "redirected to "
		}
		outputString = outputString + url + " - " + resp.Status + " " + strings.Repeat(" ", 100) + "\n" + title + " " + strconv.Itoa(length) + "\n"
		// convert resp.ContentLength to string
		fmt.Fprint(os.Stdout, "\r"+outputString)
		if redirected {
			fmt.Fprint(os.Stdout, "redirected to ")
		}
		if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently { //status code 302
			redirUrl, _ := resp.Location()
			testUrl(redirUrl.String(), showStatus, file_create, true)
		}
	}
	_, err = file_create.WriteString(outputString)
}

func main() {
	directoryList := strings.Split(*dirlist, ",")

	// check the directory list, if the found in the url
	urlFuzzScanner(directoryList)
	fmt.Fprint(os.Stdout, "\n")
	fmt.Println(statuscount) // map[string]int
}
