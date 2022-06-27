package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const MAX_CONCURRENT_JOBS = 15

type result struct {
	sumValue      int
	multiplyValue int
}

var mutex = &sync.Mutex{}

var statuscount = map[string]int{}

var clear map[string]func() //create a map for storing clear funcs

func init() {
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func CallClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
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
	scanner := bufio.NewScanner(file)
	wg := sync.WaitGroup{}

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		// check if the line is empty
		if line == "" {
			continue
		}
		wg.Add(1)
		count++
		go func(count int, url string, line string, showStatus string) {
			testUrl(url, line, showStatus)
			job(count)
			defer wg.Done()
		}(count, url, line, showStatus)
	}
	wg.Wait()
	return
}

func testUrl(url string, line string, showStatus string) {
	// create a new http client
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
	// get status parameter from the command lline
	status := os.Args[3]

	// check the directory list, if the found in the url
	urlFuzzScanner(url, directoryList, status)

	fmt.Println(statuscount) // map[string]int
}

func job(index int) {
	// fmt.Println(index, "begin doing something")
	// time.Sleep(time.Duration(rand.Intn(1) * int(time.Second)))
	// fmt.Println(index, "done")
}
