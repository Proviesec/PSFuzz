![PSFuzz](https://user-images.githubusercontent.com/6010786/176360134-adc6d195-60b0-4628-af06-b6b42afaffae.png)
![](https://us-central1-progress-markdown.cloudfunctions.net/progress/50)
# PSFuzz - ProvieSec Fuzz Scanner - Web path discovery
[![License](https://img.shields.io/badge/license-MIT-_red.svg)](https://opensource.org/licenses/MIT)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/dwisiswant0/go-dork/issues)

<a href="https://proviesec.org/">
    <img src="https://avatars.githubusercontent.com/u/92156402?s=400&u=7fe0dbb9085a37818ee8c2b061432a9a69cbff42&v=4" alt="Proviesec logo" title="Proviesec" align="right" height="60" />
</a>
<a href="https://www.buymeacoffee.com/proviesec" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>

# Introduction 

:star: Star us on GitHub — it motivates a lot! :star:

Web path discovery

Discover with ProSecFuzz hidden files and directories on a web server.
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
* -u/-url
* -d
 
 **Optional**
* -s/-status
* -c/-concurrency

# Example
![image](https://user-images.githubusercontent.com/6010786/177131582-183f6f8f-559f-43d6-b6d9-566e5cbf8a75.png)


# Todos

- [x] Multi requests
- [x] Optional param output
- [ ] Redirect handler - 301... -> Can be activated via parameter
- [ ] Output
    - [ ] TXT
    - [ ] HTML
- [x] Wildcard parameter
- [ ] Pausing progress
- [x] Progress bar
- [ ] List of URLs
- [ ] Parameter
    - [ ] Port List
    - [ ] Length
    - [ ] Response Status List 
- [ ] 403 Bypass
- [ ] Status bypass
- [ ] Show positiv false: status 200, but title 404
- [ ] Show possible false 200, same length of startsite... 
- [x] Show titel of Page
- [ ] Fingerprint check 
- [ ] compare two scans 
    - [ ] save scan
    - [ ] load scan
- [ ] optional config file
    - [ ] load config 
    - [ ] save config
    - [ ] yaml file 
      

# Example
go run main.go -url https://www.google.com -d list.txt -s true -c 2

