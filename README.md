![PSFuzz](https://user-images.githubusercontent.com/6010786/176360134-adc6d195-60b0-4628-af06-b6b42afaffae.png)
![](https://us-central1-progress-markdown.cloudfunctions.net/progress/75)
# PSFuzz - ProvieSec Fuzz Scanner - Web path discovery
[![License](https://img.shields.io/badge/license-MIT-_red.svg)](https://opensource.org/licenses/MIT)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/dwisiswant0/go-dork/issues)
[![Twitter](https://img.shields.io/twitter/follow/proviesec?label=Follow)](https://twitter.com/proviesec)

<a href="https://proviesec.org/">
    <img src="https://avatars.githubusercontent.com/u/92156402?s=400&u=7fe0dbb9085a37818ee8c2b061432a9a69cbff42&v=4" alt="Proviesec logo" title="Proviesec" align="right" height="60" />
</a>
<a href="https://www.buymeacoffee.com/proviesec" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>

Table of Contents
------------
* [Introduction](#introduction)
* [Installation](#installation--usage)
* [Wordlists](#wordlists-important)
* [Options](#options)
* [How to use](#how-to-use)
* [Todos](#todos)
* [Tips](#tips)
* [License](#license)

Introduction 
------------

:star: Star us on GitHub — it motivates a lot! :star:

Web path discovery

Discover with ProSecFuzz hidden files and directories on a web server.

## Disclaimer: DONT BE A JERK!
Needless to mention, please use this tool very very carefully. The authors won't be responsible for any consequences. 

# Installation & Usage

```go get https://github.com/Proviesec/PSFuzz```

Wordlists
---------------
**Summary:**
  - the Wordlist is a text file, each line is a path.
  - Here you get suitable lists: https://github.com/Proviesec/directory-payload-list

Options
---------------
**Required**
* `-u`/`-url` Example: `-u https://www.google.com`
 
 **Optional**
* `-o`/`-output` Example: `-o google_output` Default: output.txt
* `-d`/`-dirlist` Example: `-d list.txt` Default is https://raw.githubusercontent.com/Proviesec/directory-payload-list/main/directory-full-list.txt
* `-s`/`-status` Example: `-s true` Default:false only Status Code 200 
* `-c`/`-concurrency` Example:  `-c 5 `
* `-fscn`/`-filterStatusCodeNot`  Example: `-fscn 404`  Don't show response status code 404
* `-fsc`/`-filterStatusCode` Example: `-fsc 200,301` Show only response status code 200 and 301
* `-fl`/`-filterLength` Example: `-fl 122,1234,1235,1236` or `-fl 122,1234-1236` Show only the response with this length (or length range)
* `-fln`/`-filterLengthNot` Example: `-fln 122,1234,1235,1236` or `-fln 122,1234-1236` Show not this response with this length (or length range)
* `-fm`/`-filterMatchWord` Example: `-fm admin`
* `-rah`/`-requestAddHeader` Example: `-rah Host:127.0.0.1`
* `-b`/`-bypass` Example: `-b true` -> bypass status code: 401,402,403
* `-g`/`-generate_payload` Example: `-g 100` -> generate a,aa,ab,abc,aaa,abb,bbc
* `-od` /`-onlydomains` Example: `-od true` Show only domains in the outputfile (no status code)
* `-t` /`-filterTestLength` Example: `-t true` make a test request and check if any other request has the same length, if yes, then skip the result for this request
* `fws` / `filterWrongStatus200` Example: `-fws true` - Don´t show: in title: "Access Gateway", "Not Found", "Error"/"ERROR", "403", "Bad Request" ,"Forbidden", "500", "Internal Server Error" and body length <= 1



# Attack Configuration / Patterns


# Response Analysis 

# Example
```
go run main.go -url https://www.google.com/ -d dir-full.txt -c 2 -o testest -s true -fscn 404,301,302

go run main.go -url https://www.google.com/ -d dir-full.txt -c 2 -o googletest -s true -fl 122,1565-1569 -fln 1566-1568
```

![image](https://user-images.githubusercontent.com/6010786/180856727-0d8791af-6076-417c-94a8-05bc786b5a4d.png)

![image](https://user-images.githubusercontent.com/6010786/180856025-6922fc14-9baf-4ba7-b5c0-6d2073c5b0c2.png)

# Todos

General
- [x] Multi requests
- [x] Optional param output
- [x] check https or http
- [x] Logo and Version output
- [ ] Tryhackme room
- [ ] help mode ( -h )
- [ ] optional config file
    - [ ] load config 
    - [ ] save config
    - [ ] yaml file 
    - [ ] config for "dont show" in title/body
- [ ] Proxy
- [ ] throttle
- [ ] detect "too many requests" 
- [ ] Output
    - [x] TXT
    - [ ] CSV
    - [ ] Json
    - [ ] HTML
- [x] Progress bar
- [ ] list of sites
- [ ] Parameter
    - [ ] random payload generator 
    - [ ] choice of dirlist from proviesec github repo
    - [ ] subdomain list from proviesec github repo
    - [ ] Port List
    - [x] Length
    - [x] Length range show and not show
    - [x] Response Status List show
    - [x] Response Status Range show
    - [x] Response Status List not show
    - [x] Response Status Range not show
    - [x] Filter content type 
    - [x] Words match list title/page
    - [x] Set Optional Header
    - [ ] Set request Timeout
    - [ ] Add Cookies
    - [ ] quite Mode 
    - [ ] random user-agent
    - [x] show only the urls 
    - [x] add user agent
    - [ ] username /password basic Auth 

Attack
- [x] make GET requests 
- [ ] make POST requests 
- [ ] try PUT/DELETE/PATCH
- [x] Wordlist txt parameter 
- [x] Wildcard parameter 
- [ ] List of URLs
- [ ] depth by dir
- [ ] Word list 
    - [ ] Automatic Word list for any file html,txt, php.. 
         - [ ] payload generator, include, start or end with specific word and max length 
         - [ ] file ending as parameter list 
    - [ ] get list from any url 
    - [x] get list from proviesec github account default
    - [ ] multiple word lists 
- [ ] Crlf scan
- [ ] open redirect scan
- [ ] fuzzing parameter (from a-z)
- [ ] fuzzing http verbs
- [ ] Wordlist formats, upper lower 

Response Analysis
- [x] show response status 
- [ ] show possibile parameter
- [ ] dump the response in files 
- [ ] Fingerprint Software (Wordpress/Apache/nginx etc.)
- [ ] CORS analyse
- [ ] bypass
    - [ ] 403 Bypass, config 
    - [ ] Status bypass
- [ ] Words match list title/page/header 
    - [ ] output the match line 
- [x] Show positiv false: status 200, but title 404
- [ ] Show possible block response, after x requests "403 or too many request" 
- [x] Show possible false 200, same length of a random site
- [ ] Intilligence
   - [x] Automatically detect false 200 (really 404) 
   - [ ] too many rediretcs and then restart again, with the exclusion of
   - [ ] Show the most unique target 
- [x] Show titel of Page
- [x] Show Response Body Length
- [ ] show content type 
- [ ] Fingerprint check 
- [ ] fuzz Parameter check (normal Response vs. with paramter)
- [ ] show reflected cookie 
- [ ] show reflected params
- [ ] show reflected base64
- [ ] compare two scans 
    - [ ] save scan
    - [ ] load scan
- [x] Redirect handler - 301... -> Can be activated via parameter
    - [ ] Show Redirect URL
    - [ ] Skip Status filter if redirect true (via parameter) 
       
       
# Example
`go run main.go -url https://www.google.com -d list.txt -s true -c 2`
