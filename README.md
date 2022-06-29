![PSFuzz](https://user-images.githubusercontent.com/6010786/176360134-adc6d195-60b0-4628-af06-b6b42afaffae.png)

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

Wordlists
---------------
**Summary:**
  - the Wordlist is a text file, each line is a path.
  
Options
---------------

# Example
![image](https://user-images.githubusercontent.com/6010786/176218589-4f5f2204-fe84-4ed5-aef6-35c04d286d53.png)
![image](https://user-images.githubusercontent.com/6010786/176218657-490a1260-dac7-4764-a9c2-778c6b066f55.png)


# Todos

- [x] Multi requests
- [x] Optional param output
- [ ] Output TXT
- [ ] Wildcard parameter
- [ ] Pausing progress
- [x] Progress bar
- [ ] List of URLs
- [ ] Parameter
    - [ ] Port List
    - [ ] Length
    - [ ] Response Status
- [ ] 403 Bypass

# Example
go run main.go -url https://www.google.com -dl list.txt -status true

