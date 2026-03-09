# PSFuzz 1.0 – Cheat Sheet

Main command reference: copy-paste examples, common fuzzer-style usage, and PSFuzz-specific flags.

## Typical workflows (copy & paste)

```bash
# Classic dirbust with tech detection + JSON report
./psfuzz -u https://target.tld/FUZZ -w default -modules fingerprint,cors -of json -o scan

# Spider: follow links from every page, depth 2
./psfuzz -u https://target.tld/ -w list.txt -modules links -enqueue-module-urls links -D 2 -of json -o scan

# Log everything for later analysis (audit)
./psfuzz -u https://target.tld/FUZZ -w list.txt -audit-log audit.ndjson -o scan

# 403 bypass + recursion + only 200s with modules
./psfuzz -u https://target.tld/FUZZ -w list.txt -b -D 2 -mc 200 -modules fingerprint,links -enqueue-module-urls links -of json -o scan

# Burp as proxy: send traffic through; only matches to replay proxy
./psfuzz -u https://target.tld/FUZZ -w list.txt -x http://127.0.0.1:8080 -replay-proxy http://127.0.0.1:8080 -of json -o scan

# Explore AI: probe target once, get wordlist/extensions recommendation from OpenAI (then run a real scan with suggested -w/-e/-mc)
export OPENAI_API_KEY=sk-...
./psfuzz -u https://target.tld/ -explore-ai
```

## Quick Start (3 commands)

```bash
go build -o psfuzz .
./psfuzz -u https://target.tld/FUZZ -w default -o scan
# → scan.txt with findings; add -of json for scan.json
```

## Presets

One flag applies a set of defaults; you can still override any option with CLI flags.

```bash
# Quick: fast smoke test (fav wordlist, 5 min limit, fingerprint only, low concurrency)
./psfuzz -u https://target.tld/FUZZ -preset quick -o scan

# Stealth: low profile (rate limit, delay, random UA, jitter) to reduce WAF/blocking
./psfuzz -u https://target.tld/FUZZ -w list.txt -preset stealth -o scan

# Thorough: full discovery (high concurrency, all modules, default extensions, depth 2, JSON)
./psfuzz -u https://target.tld/FUZZ -w list.txt -preset thorough -o scan

# Override preset: e.g. stealth but higher concurrency
./psfuzz -u https://target.tld/FUZZ -preset stealth -c 15 -o scan
```

| Preset    | Concurrency | Wordlist | Maxtime | Modules              | Other                          |
|-----------|-------------|----------|---------|----------------------|---------------------------------|
| **quick**   | 5           | fav      | 300 s   | fingerprint          | output txt                      |
| **stealth** | 5           | (unchanged) | (unchanged) | (unchanged)     | rate 10/s, delay 0.5–1s, random UA, jitter |
| **thorough**| 30          | (unchanged) | (unchanged) | fingerprint,cors,urlextract,links | ext-defaults, depth 2, json |

## Explore AI

When `-explore-ai` is set, the **first URL** is probed once; **fingerprint, headers, and response (body)** are sent to an **AI backend** of your choice. The AI is given SecLists ([SecLists CMS](https://github.com/danielmiessler/SecLists/tree/master/Discovery/Web-Content/CMS) and common.txt, plus fav/default payload URLs) and returns **JSON** with either a single **wordlist_url** or multiple **wordlist_urls**. If the AI returns **wordlist_urls** (e.g. nginx + wordpress + fav), PSFuzz fetches each URL, **merges** the wordlists and **removes duplicates**, then runs the scan with that combined payload.

**Providers:** Choose with `-explore-ai-provider openai | ollama | gemini`. Default: **openai** (needs `OPENAI_API_KEY`). **Ollama** uses local models (default `http://localhost:11434`, model `llama3.1`; no API key). **Gemini** needs `GEMINI_API_KEY` or `GOOGLE_API_KEY`. Override URL with `-explore-ai-endpoint`, model with `-explore-ai-model`.

**Resolution order for the payload:** 1) **AI `wordlist_urls`** (merge + dedupe), 2) **AI `wordlist_url`**, 3) `-explore-ai-wordlist` (map), 4) `-explore-ai-wordlists-dir` (file in dir), 5) built-in defaults.

**Next-level behaviour:**
- **Sensitive headers** (Cookie, Authorization, X-API-Key, etc.) are **redacted** before sending to the AI so tokens/sessions are never leaked.
- **Profile** `-explore-ai-profile quick|balanced|thorough`: quick = small/fast wordlists, thorough = larger wordlists and more extensions. Default: balanced.
- **Focus areas & next steps:** The AI can return optional `focus_areas` (e.g. "wp-admin, plugin versions") and `next_steps` (e.g. "recursion on /wp-content"); they are printed after the recommendation.
- **Robustness:** If the API returns invalid JSON, PSFuzz asks the model once to fix it and retries.
- **Cache:** Explore AI results are cached per target and **per provider** (normalized URL + provider) for **1 hour** under the user cache dir (e.g. `~/.cache/psfuzz/explore-ai` on Linux). A cache hit skips the probe and API call; output shows `(from cache)`. Use `-explore-ai-no-cache` to bypass (always call API).

```bash
# Quick recon (small wordlist)
./psfuzz -u https://target.tld/FUZZ -explore-ai -explore-ai-profile quick -o scan

# Thorough (larger wordlist, more extensions)
./psfuzz -u https://target.tld/FUZZ -explore-ai -explore-ai-profile thorough -o scan
```

**Custom lists:** Put files in a directory and use `-explore-ai-wordlists-dir`, or map types to paths/URLs with `-explore-ai-wordlist wordpress:./my-wp.txt,nginx:https://...`. The AI can still suggest a type; your map/dir overrides the AI’s URL when the type matches.

```bash
# OpenAI (default)
export OPENAI_API_KEY=sk-...
./psfuzz -u https://target.tld/FUZZ -explore-ai -o scan

# Ollama (local, no key)
./psfuzz -u https://target.tld/FUZZ -explore-ai -explore-ai-provider ollama -explore-ai-model llama3.1 -o scan

# Gemini
export GEMINI_API_KEY=...
./psfuzz -u https://target.tld/FUZZ -explore-ai -explore-ai-provider gemini -o scan

# Override with your dir: wordlists in ./wordlists (wordpress.txt, typo3.txt, …) override defaults
./psfuzz -u https://target.tld/FUZZ -explore-ai -explore-ai-wordlists-dir ./wordlists -o scan

# Override with custom map (checked first)
./psfuzz -u https://target.tld/FUZZ -explore-ai -explore-ai-wordlist "wordpress:./my-wp.txt,typo3:https://..." -o scan
# Config file: "exploreAIWordlistMap": {"wordpress": "./wordlists/wordpress.txt", "typo3": "https://..."}
```

**Own lists / URLs:** Use `-explore-ai-wordlist name:path_or_url,name2:url` to override the AI’s `wordlist_url` for specific types. The map is checked after the AI’s `wordlist_url`, then dir, then built-in defaults.

**SecLists CMS** has many lists; the AI can reference any. Example sources:

| Type     | Example (SecLists) |
|----------|---------------------|
| Common   | `.../Discovery/Web-Content/common.txt` |
| WordPress| `.../Discovery/Web-Content/CMS/wordpress.fuzz.txt` |
| Drupal   | `.../Discovery/Web-Content/CMS/drupal.txt` |
| Joomla   | `.../CMS/joomla-themes.fuzz.txt`, `joomla-plugins.fuzz.txt` |
| TYPO3    | `.../CMS/top-3.txt` |
| Nginx    | `.../CMS/nginx.txt` |
| Others   | [SecLists CMS](https://github.com/danielmiessler/SecLists/tree/master/Discovery/Web-Content/CMS): coldfusion, django, sharepoint, siteminder, sap, etc. |

**SecLists Discovery/Web-Content (generic / directories):**

| List | Use case |
|------|----------|
| `common.txt` | Sensitive files, .git, .env, .well-known, common paths |
| `directory-list-1.0.txt` | Small set of common directories |
| `directory-list-2.3-small.txt` | Small directory list |
| `directory-list-2.3-medium.txt` | Medium directory list |
| `raft-small-directories.txt` | Small raft directories |
| `raft-large-directories.txt` | Large raft (admin, wp-content, uploads, etc.) |
| `quickhits.txt` | Quick-hit paths |

Base URL: `https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/` + filename.

**More wordlist sources:**

| Project | Description / URL |
|---------|-------------------|
| [SecLists](https://github.com/danielmiessler/SecLists) | Discovery, Fuzzing, Passwords, Payloads; [Web-Content](https://github.com/danielmiessler/SecLists/tree/master/Discovery/Web-Content), [CMS](https://github.com/danielmiessler/SecLists/tree/master/Discovery/Web-Content/CMS) |
| [FuzzDB](https://github.com/fuzzdb-project/fuzzdb) | Discovery, attack patterns, LFI/RFI payloads |
| [PayloadsAllTheThings](https://github.com/swisskyrepo/PayloadsAllTheThings) | Bypasses, LFI, XSS, etc. (often used as reference; paths under `Discovery` or similar) |
| [Assetnote wordlists](https://wordlists.assetnote.io/) | High-quality discovery lists (e.g. `httparchive_*`); copy URLs into `-explore-ai-wordlist` or use as `wordlist_url` |
| PSFuzz built-in | `-w default` and `-w fav` (Proviesec payload lists) |

Use `-explore-ai-wordlist type:url` to point any type at your own path or URL.

Optional: use `-login-url` etc. if the target requires auth before the probe.

**Possible future improvements:** Cache AI result per target to avoid repeated API calls; "follow-up" mode that sends first-scan results back to the AI for a second recommendation (e.g. deeper wordlist or recursion); multi-URL probing when using `-list` so the AI sees several targets and can suggest per-cluster wordlists.

**How to test Explore AI**

1. **Build and set API key**
   ```bash
   go build -o psfuzz .
   export OPENAI_API_KEY=sk-your-key-here
   ```

2. **AI decides everything** (no wordlist config; built-in default list is used)
   ```bash
   ./psfuzz -u https://example.com/FUZZ -explore-ai -o scan
   ```
   You see the AI recommendation, then the scan starts with the default wordlist for that type (e.g. SecLists wordpress.fuzz.txt when the AI suggests "wordpress"), plus suggested extensions and status codes.

3. **Override with a local directory**
   ```bash
   mkdir -p wordlists
   echo -e "wp-admin\nwp-login\nwp-includes" > wordlists/wordpress.txt
   ./psfuzz -u https://example.com/FUZZ -explore-ai -explore-ai-wordlists-dir wordlists -o scan
   ```
   If the AI suggests "wordpress", the scan uses `wordlists/wordpress.txt` (your file overrides the default).

4. **Custom list via URL** (map a type to a remote wordlist)
   ```bash
   ./psfuzz -u https://example.com/FUZZ -explore-ai -explore-ai-wordlist "generic:https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/common.txt" -o scan
   ```
   When the AI suggests "generic", PSFuzz uses that URL as the wordlist and runs the scan.

Use a real target (e.g. a WordPress test site or any URL you’re allowed to scan) so the fingerprint and AI suggestion make sense.

## URL List

```bash
./psfuzz -list targets.txt -w list.txt
```

## Common fuzzer-style basics

```bash
# classic FUZZ
./psfuzz -u https://target.tld/FUZZ -w wordlist.txt

# multiple wordlists with keywords
./psfuzz -u https://target.tld/USER/PASS -w users.txt:USER,pass.txt:PASS

# input modes
./psfuzz -u https://target.tld/USER/PASS -w users.txt:USER,pass.txt:PASS -mode clusterbomb
./psfuzz -u https://target.tld/USER/PASS -w users.txt:USER,pass.txt:PASS -mode pitchfork
./psfuzz -u https://target.tld/USER/PASS -w users.txt:USER,pass.txt:PASS -mode sniper
```

## Filtering & Matching

```bash
# status include/exclude
./psfuzz -u https://target.tld/FUZZ -w list.txt -mc 200,301-302 -fc 404

# size/word/line filters
./psfuzz -u https://target.tld/FUZZ -w list.txt -ms 200 -fs 50
./psfuzz -u https://target.tld/FUZZ -w list.txt -mw 10 -fw 3
./psfuzz -u https://target.tld/FUZZ -w list.txt -ml 5 -fls 2

# regex + text‑only regex
./psfuzz -u https://target.tld/FUZZ -w list.txt -fr "not found"
./psfuzz -u https://target.tld/FUZZ -w list.txt -mr "admin" -mrt

# response time match/filter (ms)
./psfuzz -u https://target.tld/FUZZ -w list.txt -mt 0-300
./psfuzz -u https://target.tld/FUZZ -w list.txt -ft 1000-999999

# near-duplicate filter
./psfuzz -u https://target.tld/FUZZ -w list.txt -nd -nd-len 20 -nd-words 5 -nd-lines 5

# block words (exclude)
./psfuzz -u https://target.tld/FUZZ -w list.txt -bw "not found,access denied"

# interesting strings (only keep matches)
./psfuzz -u https://target.tld/FUZZ -w list.txt -is "admin,secret,token"

# dedupe
./psfuzz -u https://target.tld/FUZZ -w list.txt -fd -dt 5
```

## Extensions

```bash
./psfuzz -u https://target.tld/FUZZ -w list.txt -e php,txt,zip
./psfuzz -u https://target.tld/FUZZ -w list.txt -ext-defaults
```

## Wordlist Case

```bash
./psfuzz -u https://target.tld/FUZZ -w list.txt -wordlist-case lower
```

## Encoders

Encode payloads per keyword for WAF bypass or parameter fuzzing (e.g. URL-encode query values). Format: `-enc KEYWORD:enc1,enc2`. Multiple `-enc` flags are merged (e.g. `-enc FUZZ:urlencode -enc PARAM:base64encode`).

Available encoders: `urlencode`, `doubleurlencode`, `base64encode`, `base64decode`, `htmlencode`, `htmldoubleencode`.

```bash
# URL-encode the FUZZ value (e.g. for query params)
./psfuzz -u "https://target.tld/page?q=FUZZ" -w list.txt -enc FUZZ:urlencode

# Double URL-encode (WAF bypass)
./psfuzz -u https://target.tld/FUZZ -w list.txt -enc FUZZ:doubleurlencode

# Base64 then URL-encode (chained)
./psfuzz -u "https://target.tld/api?data=FUZZ" -w list.txt -enc FUZZ:base64encode,urlencode

# Multiple keywords
./psfuzz -u "https://target.tld/?USER=FUZZ&token=PARAM" -w users.txt:USER,payloads.txt:PARAM -enc USER:urlencode -enc PARAM:base64encode
```

Config: `"encoders": "FUZZ:urlencode;PARAM:base64encode"` (semicolon separates keywords).

## 403/401/402 Bypass

```bash
# enable bypass variants
./psfuzz -u https://target.tld/admin -w list.txt -b

# keep bypass controlled
./psfuzz -u https://target.tld/admin -w list.txt -b -bypass-budget 3 -bypass-ratio 0.3
```

## WAF & Wildcard

```bash
# adaptive slowdown when WAF is suspected
./psfuzz -u https://target.tld/FUZZ -w list.txt -waf-adaptive -waf-threshold 20 -waf-factor 2.5

# auto‑wildcard baseline per host
./psfuzz -u https://target.tld/FUZZ -w list.txt -awc
```

## HTTP Controls

```bash
# custom method / data / headers
./psfuzz -u https://target.tld/FUZZ -w list.txt -X POST -d "name=FUZZ" -H "Content-Type:application/x-www-form-urlencoded"

# body from file: load template from file, substitute FUZZ (and other keywords) per request
./psfuzz -u https://target.tld/api -w list.txt -X POST -d @body.json -H "Content-Type: application/json"
# body.json example: {"user":"FUZZ","role":"admin"}

# raw request file
./psfuzz -request request.txt -u https://target.tld -w list.txt

# random user-agent
./psfuzz -u https://target.tld/FUZZ -w list.txt -random-ua

# verbs
./psfuzz -u https://target.tld/FUZZ -w list.txt -verbs GET,POST
./psfuzz -u https://target.tld/FUZZ -w list.txt -auto-verb
```

## Proxy & Replay

```bash
# proxy (http/socks5)
./psfuzz -u https://target.tld/FUZZ -w list.txt -x http://127.0.0.1:8080
./psfuzz -u https://target.tld/FUZZ -w list.txt -x socks5://127.0.0.1:1080

# replay only matches
./psfuzz -u https://target.tld/FUZZ -w list.txt -replay-proxy http://127.0.0.1:8081 -replay-on-match=true
```

## Timing & Recursion Strategy

```bash
# max scan duration (seconds), then exit
./psfuzz -u https://target.tld/FUZZ -w list.txt -maxtime 3600

# max duration per task (e.g. per recursion branch)
./psfuzz -u https://target.tld/FUZZ -w list.txt -maxtime-job 120

# recursion: default (by status) vs greedy (every match)
./psfuzz -u https://target.tld/FUZZ -w list.txt -D 3 -recursion-strategy default
./psfuzz -u https://target.tld/FUZZ -w list.txt -D 3 -recursion-strategy greedy
```

## HTTP/2 & VHost

```bash
# use HTTP/2
./psfuzz -u https://target.tld/FUZZ -w list.txt -http2

# VHost fuzzing: first wordlist value as Host header
./psfuzz -u https://target.tld/FUZZ -w subdomains.txt -vhost
```

## Response Modules (fingerprint, CORS, AI, urlextract, links)

See **[MODULES.md](MODULES.md)** for usage, output (TXT vs JSON), and recommendations.

```bash
# technology detection + CORS + OpenAI verdict (needs OPENAI_API_KEY)
./psfuzz -u https://target.tld/FUZZ -w list.txt -modules fingerprint,cors,ai -of json -o scan

# custom AI prompt (placeholders: {{status}}, {{method}}, {{url}}, {{body}})
./psfuzz -u https://target.tld/FUZZ -w list.txt -modules ai -ai-prompt "Security expert: status {{status}}, {{url}}. Anything unusual? One sentence.\n\n{{body}}"

# AI module with Ollama (local, no key) or Gemini
./psfuzz -u https://target.tld/FUZZ -w list.txt -modules ai -ai-provider ollama -o scan
./psfuzz -u https://target.tld/FUZZ -w list.txt -modules ai -ai-provider gemini -o scan   # needs GEMINI_API_KEY

# extract URLs from body + Location (urlextract) and HTML links href/action/src (links)
./psfuzz -u https://target.tld/FUZZ -w list.txt -modules urlextract,links -of json -o scan
```

## Audit log (save all requests/responses)

For post-run analysis or re-filtering without rescanning: write every request/response as one NDJSON line to a file.

```bash
# log full traffic (one request + response per line)
./psfuzz -u https://target.tld/FUZZ -w list.txt -audit-log audit.ndjson -o scan

# limit body size per entry (e.g. 50 KB) to save space
./psfuzz -u https://target.tld/FUZZ -w list.txt -audit-log audit.ndjson -audit-max-body 51200 -o scan
```

Config: `"auditLog": "/path/to/audit.ndjson"`, `"auditMaxBodySize": 51200` (0 = unlimited).

## Link-driven discovery (spider mode)

Extract links from HTML (and URLs from body) and automatically add them to the scan queue—like “extract links” in other tools, combined with your wordlist.

```bash
# follow links from every found page, depth 2
./psfuzz -u https://target.tld/ -w list.txt -modules links -enqueue-module-urls links -D 2 -of json -o scan

# enqueue URLs from body + Location (urlextract) and HTML links (links)
./psfuzz -u https://target.tld/ -w list.txt -modules urlextract,links -enqueue-module-urls urlextract,links -D 2 -of json -o scan

# only scan 200s and enqueue links from those (less noise)
./psfuzz -u https://target.tld/ -w list.txt -mc 200 -modules links -enqueue-module-urls links -D 2 -o scan
```

Config: `"enqueueModuleUrls": "urlextract,links"`. Enqueued URLs get depth+1 and the same scope/visited logic as recursion.

## Extracted URLs to file (no enqueue)

Write all URLs discovered by modules (urlextract, links, etc.) to a file, one per line, without adding them to the scan queue.

```bash
./psfuzz -u https://target.tld/FUZZ -w list.txt -modules links,urlextract -extracted-urls-file found_urls.txt -of json -o scan
```

Config: `"extractedUrlsFile": "/path/to/urls.txt"` (empty = off).

## Config file

```bash
# load config (CLI flags override config)
./psfuzz -cf config.json -w list.txt -o scan

# save current settings as config (includes auditLog, enqueueModuleUrls)
./psfuzz -u https://target.tld/FUZZ -w list.txt -audit-log audit.ndjson -enqueue-module-urls links -save-config my-config.json
# Note: login user, password, and body are never written to the config file (use CLI or env for credentials).
```

Example `config.json` (excerpt): `auditLog`, `auditMaxBodySize`, `enqueueModuleUrls`, `modules`—see [README](README.md#config-file) and `config.example.json`.

## Resume & Stop

```bash
# resume
./psfuzz -u https://target.tld/FUZZ -w list.txt -resume resume.json -resume-every 500

# stop conditions
./psfuzz -u https://target.tld/FUZZ -w list.txt -sf 403 -sa 50 -se
```

## TLS skip (self-signed / Burp MITM)

```bash
# skip TLS certificate verification
./psfuzz -u https://target.tld/FUZZ -w list.txt -insecure
./psfuzz -u https://target.tld/FUZZ -w list.txt -k
```

## Login (session for all requests)

Perform one login request; all Set-Cookie values are then sent with every fuzz request.

```bash
# form login (fields username + password)
./psfuzz -u https://target.tld/FUZZ -w list.txt -login-url https://target.tld/login -login-user admin -login-pass secret -o scan

# custom body (e.g. JSON or different field names)
./psfuzz -u https://target.tld/api/FUZZ -w list.txt -login-url https://target.tld/api/login -login-method POST -login-body '{"email":"a@b.com","password":"secret"}' -login-content-type "application/json" -o scan
```

Config: `loginUrl`, `loginMethod`, `loginUser`, `loginPass`, `loginBody`, `loginContentType`, `insecureSkipVerify`.

## Safety & Performance

```bash
# safe mode off for local testing
./psfuzz -u http://127.0.0.1/FUZZ -w list.txt -safe=false

# allowed hosts
./psfuzz -u https://target.tld/FUZZ -w list.txt -allow-hosts target.tld,api.target.tld

# exclude paths
./psfuzz -u https://target.tld/FUZZ -w list.txt -exclude-paths js,img,static

# rate/timeout/delay
./psfuzz -u https://target.tld/FUZZ -w list.txt -rate 50 -timeout 10 -p 0.1-0.3

# cap response size
./psfuzz -u https://target.tld/FUZZ -w list.txt -max-size 1048576

# minimum response size
./psfuzz -u https://target.tld/FUZZ -w list.txt -min-size 200

# jitter profiling
./psfuzz -u https://target.tld/FUZZ -w list.txt -jitter -jitter-threshold 800 -jitter-factor 1.2

# dump responses
./psfuzz -u https://target.tld/FUZZ -w list.txt -dump -dump-dir dumps
```

## Output Formats

```bash
./psfuzz -u https://target.tld/FUZZ -w list.txt -of txt
./psfuzz -u https://target.tld/FUZZ -w list.txt -of json
./psfuzz -u https://target.tld/FUZZ -w list.txt -of html
./psfuzz -u https://target.tld/FUZZ -w list.txt -of csv
./psfuzz -u https://target.tld/FUZZ -w list.txt -of ndjson
./psfuzz -u https://target.tld/FUZZ -w list.txt -of compatjson
```
