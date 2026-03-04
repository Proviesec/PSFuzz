# Fuzzing Guides vs. PSFuzz

This document maps two popular fuzzing guides to PSFuzz: what they recommend, whether PSFuzz already supports it, and what could be added. Use it to prioritize features and document workarounds.

**Guides:**

1. **DEV – A Summary of Fuzzing Tools and Dictionaries for Bug Bounty Hunters**  
   [dev.to/tutorialboy/a-summary-of-fuzzing-tools-and-dictionaries-for-bug-bounty-hunters-2n3k](https://dev.to/tutorialboy/a-summary-of-fuzzing-tools-and-dictionaries-for-bug-bounty-hunters-2n3k)  
   Covers: Wfuzz, Ffuf, GoBuster; wordlist/dictionary sources (SecLists, FuzzDB, fuzz.txt, PayloadsAllTheThings, etc.).

2. **Medium – The 50 Ultimate Fuzzing Guide for Bug Bounty Hunters**  
   [medium.com/@pankajkryadav1/the-50-ultimate-fuzzing-guide-for-bug-bounty-hunters-mastering-fuzzing-9f70e5474dc5](https://medium.com/@pankajkryadav1/the-50-ultimate-fuzzing-guide-for-bug-bounty-hunters-mastering-fuzzing-9f70e5474dc5)  
   FFUF-focused: recursive fuzzing, HTTP methods, parameters, subdomains, Burp, API/WebSocket, encodings, WAF bypass, file upload, timing, JSON/GraphQL/XML, JWT, SSRF, SSTI, Nuclei, automation, etc.

---

## Legend

| Status | Meaning |
|--------|--------|
| **Yes** | Supported in PSFuzz today; see example. |
| **Partial** | Partially supported or workaround; gap noted. |
| **No** | Not supported; candidate for implementation. |

---

## 1. DEV Guide – Tools & Dictionaries

### 1.1 Tools (Wfuzz, Ffuf, GoBuster)

| Topic | Status | PSFuzz |
|-------|--------|--------|
| FUZZ placeholder + wordlist | **Yes** | `-u https://target.com/FUZZ -w wordlist.txt` |
| Replace FUZZ with wordlist values | **Yes** | Same as ffuf/wfuzz. |
| Recursive fuzzing | **Yes** | `-D 3`, `-recursion-strategy default\|greedy` |
| GoBuster-style dir / vhost | **Yes** | Dir: same URL pattern. VHost: `-vhost -w subdomains.txt` |
| GoBuster DNS subdomain mode | **Partial** | No DNS resolution mode. Use `-vhost` (Host header fuzzing) or external tool (e.g. subfinder) + `-list`. |

### 1.2 Dictionaries / Wordlist Sources (DEV)

The guide recommends these **dictionaries** (use with `-w <path>` or merge into your wordlist):

| Source | Purpose | Use with PSFuzz |
|--------|---------|-----------------|
| [SecLists](https://github.com/danielmiessler/SecLists) | Discovery, payloads, usernames, etc. | `-w /path/to/SecLists/Discovery/Web-Content/...` |
| [FuzzDB](https://github.com/fuzzdb-project/fuzzdb) | Attack patterns, discovery | `-w /path/to/fuzzdb/discovery/...` |
| [fuzz.txt (Bo0oM)](https://github.com/Bo0oM/fuzz.txt) | Potentially dangerous files | `-w /path/to/fuzz.txt` |
| [PayloadsAllTheThings](https://github.com/swisskyrepo/PayloadsAllTheThings) | XSS, SQLi, etc. | `-w` for param/endpoint fuzzing |
| [big-list-of-naughty-strings](https://github.com/minimaxir/big-list-of-naughty-strings) | Edge-case strings | `-w` for input fuzzing |
| [AwesomeXSS](https://github.com/s0md3v/AwesomeXSS) | XSS payloads | `-w` |
| [Port Swigger XSS Cheat Sheet](https://portswigger.net/web-security/cross-site-scripting/cheat-sheet) | Reference | Manual / custom wordlist |
| [bl4de/dictionaries](https://github.com/bl4de/dictionaries) | Wordlists | `-w` |
| [Open-Redirect-Payloads](https://github.com/cujanovic/Open-Redirect-Payloads) | Redirect payloads | `-w` |
| [EdOverflow bugbounty-cheatsheet](https://github.com/EdOverflow/bugbounty-cheatsheet) | Reference | — |
| [payloadbox/xss-payload-list](https://github.com/payloadbox/xss-payload-list) | XSS | `-w` |
| [WordList (orwagodfather)](https://github.com/orwagodfather/WordList) | Wordlists | `-w` |
| [Bug-Bounty-Wordlists](https://github.com/Karanxa/Bug-Bounty-Wordlists) | Wordlists | `-w` |
| [Assetnote wordlists](https://wordlists.assetnote.io/) | Wordlists | Download, then `-w` |
| [OneListForAll](https://github.com/six2dez/OneListForAll) | Combined list | `-w` |

**Doc gap:** README/CHEATSHEET do not yet list “recommended wordlist sources”. Adding a short “Wordlist sources” section would align with this guide.

---

## 2. Medium “50 Ultimate” Guide – Technique Mapping

### 2.1 Discovery & Recursion

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 1 | Custom wordlists & recursive fuzzing | **Yes** | `-w yourlist.txt -D 3` |
| 12 | Recursion depth control | **Yes** | `-D 3`, `-recursion-strategy default\|greedy` |
| 24 | Hidden directory & file discovery | **Yes** | Same; `-e php,txt`, `-ext-defaults` |
| 34 | Subdomain fuzzing (Host header) | **Yes** | `-vhost -w subdomains.txt -u https://target/` |
| 36 | Amass + fuzzer (multi-target) | **Partial** | `-list file` accepts URL list; no stdin for URL list. Workaround: `amass enum -d target.com -o out.txt` then `-list out.txt` (if one URL per subdomain). For “base URL + subdomain” you need a small script to build the list. |

### 2.2 HTTP Methods & Headers

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 2 | Uncommon HTTP methods (PUT, DELETE, TRACE) | **Yes** | `-X PUT`, `-verbs GET,POST,PUT,DELETE`, `-auto-verb` |
| 11 | Fuzz hidden HTTP headers | **Yes** | `-H "Custom-Header: FUZZ"` |
| 26 | Burp Collaborator / OOB (blind SSRF/RCE) | **Partial** | No built-in OOB. Use payload wordlist that triggers your Collaborator URL; check Collaborator manually. |

### 2.3 Parameters & Body

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 3 | Parameter-based fuzzing (multiple params) | **Yes** | `-w params.txt:PARAM -u "https://target?PARAM=val"`, multi-wordlist |
| 8 | Multi-wordlist (clusterbomb, pitchfork) | **Yes** | `-w a.txt:A -w b.txt:B -mode clusterbomb\|pitchfork\|sniper` |
| 18 | JSON API fuzzing | **Yes** | `-X POST -d '{"param":"FUZZ"}' -H "Content-Type: application/json"` |
| 21 | Parameter pollution (HPP) | **Partial** | Two keywords in URL/body (e.g. `param=FUZZ&param=FUZZ2`) with two wordlists. |
| 23 | XML/XXE fuzzing | **Yes** | `-d @request.xml` loads body from file; FUZZ (and other keywords) in the file are replaced per request. |
| 33 | GraphQL fuzzing | **Yes** | `-X POST -d '{"query":"{ FUZZ }"}' -H "Content-Type: application/json"` |
| 35 | Login/credential fuzzing | **Yes** | `-w users.txt:USER -w pass.txt:PASS` + `-X POST -d "username=USER&password=PASS"` or use `-login-url` for session then fuzz. |
| 37 | JWT fuzzing | **Yes** | `-H "Authorization: Bearer FUZZ" -w jwt_payloads.txt` |
| 47 | Multi-step forms / JSON APIs | **Yes** | Same as JSON/body fuzzing. |

### 2.4 Filters, Performance, Stealth

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 9 | Advanced matchers/filters (status, size, regex) | **Yes** | `-mc`, `-fc`, `-ms`, `-fs`, `-mr`, `-fr`, `-bw`, `-is`, `-nd` |
| 10 | Performance (threads, delay) | **Yes** | `-c`, `-p 0.5-1.5`, `-rate`, `-preset stealth` |
| 13 | Rate limiting evasion (random delay) | **Yes** | `-p 0.1-2.0`, `-preset stealth`, `-jitter` |
| 16 | Response time analysis (blind SSRF, etc.) | **Yes** | `-mt`, `-ft` (match/filter by time in ms) |
| 17 | Time-based SQLi (SLEEP, BENCHMARK) | **Yes** | `-w sqli_payloads.txt`, filter by time `-mt` or match by content |
| 27 | Rate limiting / throttling bypass | **Yes** | Delay, low `-c`, proxy, `-waf-adaptive` |
| 41 | Timing attacks (blind SQLi) | **Yes** | `-mt` / `-ft` |

### 2.5 WAF, Encoding, Bypass

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 19 | WAF bypass (custom encodings, e.g. urlencode) | **Yes** | `-enc FUZZ:urlencode`, `-enc FUZZ:doubleurlencode`, `-enc FUZZ:base64encode` (and htmlencode, base64decode). See CHEATSHEET. |
| 5, 31 | Burp integration (proxy, session) | **Yes** | `-x http://127.0.0.1:8080`, `-replay-proxy`, `-insecure`, `-login-url` for session |
| 403/401 bypass | — | **Yes** | `-b` (bypass variants), `-waf-adaptive`, `-awc` (auto-wildcard) |

### 2.6 File & Upload

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 15 | File upload fuzzing (file=@FUZZ, extensions) | **No** | No multipart/form-data with file from wordlist (e.g. `file=@/path/FUZZ`). **High-value add for upload testing.** |
| 25 | LFI/RFI fuzzing (file param) | **Yes** | `-w lfi_payloads.txt -u "https://target/page?file=FUZZ"` |
| 45 | File upload (extensions, MIME) | **No** | Same as #15; no multipart file fuzzing. |

### 2.7 Vulnerability-Specific Fuzzing

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 20 | Template injection (SSTI) | **Yes** | `-w ssti_payloads.txt -u "https://target?template=FUZZ"` or POST body |
| 22 | Chained vulns (SSRF + metadata) | **Yes** | Wordlist with metadata URLs; fuzz param |
| 39 | SSRF fuzzing | **Yes** | `-w ssrf_payloads.txt -u "https://target?url=FUZZ"` |
| 42 | Chaining (exposed key → internal API) | **Yes** | `-H "Authorization: Bearer <key>"` + fuzz endpoints |
| 43 | Privilege escalation / RBAC | **Yes** | `-b "session=low_priv_cookie"` + `-w admin_endpoints.txt` |
| 44 | SSTI (Twig, Jinja) | **Yes** | Same as #20 |
| 46 | Cloud metadata (SSRF) | **Yes** | Wordlist with metadata URLs |
| 48 | Unauthenticated endpoints | **Yes** | `-w api_endpoints.txt -b "session=expired"` or no cookie |
| 49 | Mobile API fuzzing | **Yes** | Same as API fuzzing; capture with Burp/mitmproxy, then `-request` or `-u` + `-H` |

### 2.8 Infrastructure & Cloud

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 28 | S3 / cloud bucket fuzzing | **Yes** | `-u https://s3.amazonaws.com/FUZZ` or similar |
| 4 | DNS subdomain (DNS mode) | **Partial** | No DNS resolution. Use `-vhost` (HTTP) or external DNS tool + `-list`. |

### 2.9 Body from File & Encodings

| Topic | Status | PSFuzz |
|-------|--------|--------|
| Body from file with FUZZ inside | **Yes** | `-d @path` loads body template from file; FUZZ and other keywords substituted per request. |
| Payload encoders (urlencode, base64, double) | **Yes** | `-enc KEYWORD:enc1,enc2` (see CHEATSHEET). |

### 2.10 Automation & Integration

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 14 | Nuclei + httpx (pipe targets) | **Partial** | Export URLs via `-of json`, `-extracted-urls-file`; user pipes to Nuclei/httpx. No built-in “run Nuclei on results”. IDEAS: “Nuclei / template run”. |
| 30 | Cron / Jenkins / GitHub Actions | **Yes** | Run `psfuzz ... -o scan` in pipeline; `-of json` for parsing |
| 32 | Interlace (parallel targets) | **Partial** | `-list targets.txt` for multiple URLs; no Interlace-specific feature. User can run Interlace with psfuzz as command. |
| 38 | FFUF → Nuclei pipeline | **Partial** | Same as #14: export URL list, then `nuclei -l urls.txt` |

### 2.11 WebSockets & Other

| # | Technique | Status | PSFuzz |
|---|-----------|--------|--------|
| 6, 29 | WebSocket fuzzing | **No** | HTTP only; no WebSocket client. Out of scope for current design. |
| 7 | Dynamic wordlist (CeWL) | **Partial** | Not built-in. User runs CeWL, then `-w cewl_output.txt`. |
| 40 | Custom plugins | **No** | Modules are built-in only; no plugin API. |

### 2.12 Reflection / “Reflexive” Match

| Topic | Status | PSFuzz |
|-------|--------|--------|
| Match only if FUZZ value is reflected in response | **No** | vaf has this. Could add as filter or module (e.g. `-reflective` or module `reflection`). |

---

## 3. Summary: What PSFuzz Already Covers

From both guides, PSFuzz **already supports**:

- FUZZ + wordlists, multiple keywords, clusterbomb/pitchfork/sniper  
- Recursion with depth and strategy  
- Custom HTTP methods and header fuzzing  
- JSON/GraphQL/XML (inline body), POST body, parameters  
- Status/size/regex/time filters, block-words, interesting-strings  
- Rate/delay, jitter, WAF-adaptive, 403 bypass, presets (quick/stealth/thorough)  
- Proxy, replay proxy, Burp-friendly (insecure, cookies)  
- Login/session, VHost fuzzing  
- Response time match/filter (timing attacks, blind SQLi)  
- LFI/RFI, SSRF, SSTI, JWT, credential fuzzing, S3-style URL fuzzing  
- Multi-target via `-list`  
- Export (JSON, etc.) for Nuclei/httpx pipelines  

---

## 4. Gaps and Recommended Additions

### 4.1 High priority (guides + TOOL_COMPARISON)

| Feature | Guides | Notes |
|---------|--------|--------|
| **Payload encoders** | ~~Medium #19~~ | *Implemented.* `-enc KEYWORD:enc1,enc2` (urlencode, doubleurlencode, base64encode, etc.). |
| **Multipart file upload fuzzing** | Medium #15, #45 | `-F "file=@FUZZ"` or similar: wordlist = file paths or extension; tool sends multipart with file. |
| **Body from file with FUZZ** | ~~Medium #23, ROADMAP~~ | *Implemented.* `-d @request.xml` loads body from file; FUZZ and other keywords are replaced per request. |

### 4.2 Medium priority (guides + IDEAS)

| Feature | Guides | Notes |
|---------|--------|--------|
| **Recommended wordlist docs** | DEV | Short section in README or CHEATSHEET: SecLists, FuzzDB, fuzz.txt, PayloadsAllTheThings, Assetnote, etc. |
| **Reflexive / reflection filter** | vaf, TOOL_COMPARISON | Only match if the fuzzed value appears in response body (XSS/reflection checks). |
| **URL list from stdin** | Medium #36 | `-list -` or pipe: `cat urls.txt \| psfuzz -list -` for Amass/httpx-style pipelines. |
| **Autocalibration** | TOOL_COMPARISON | Auto baseline size/code (e.g. first N requests) to suggest or apply filters. |

### 4.3 Lower priority / ideas

| Feature | Guides | Notes |
|---------|--------|--------|
| **DNS mode** | Medium #4, DEV (GoBuster) | Native DNS subdomain enumeration. Larger change; `-vhost` covers HTTP subdomain fuzzing. |
| **Nuclei hook** | Medium #14, #38, IDEAS | Optional “after scan” run Nuclei on found URLs. |
| **Secrets/headers modules** | ROADMAP | Already planned (secrets, headers); aligns with “find sensitive data” in guides. |
| **Open redirect / SSRF probe module** | IDEAS | Optional module to probe for redirect/SSRF. |

### 4.4 Out of scope (for now)

- **WebSocket client** (Medium #6, #29): Different protocol; keep HTTP-focused.  
- **Custom plugin API** (Medium #40): Modules are built-in; plugin API is a larger design.  
- **Built-in CeWL**: External tool; document “use CeWL then `-w`” in CHEATSHEET.

---

## 5. References

- [DEV – Fuzzing Tools and Dictionaries for Bug Bounty Hunters](https://dev.to/tutorialboy/a-summary-of-fuzzing-tools-and-dictionaries-for-bug-bounty-hunters-2n3k)  
- [Medium – The 50 Ultimate Fuzzing Guide for Bug Bounty Hunters](https://medium.com/@pankajkryadav1/the-50-ultimate-fuzzing-guide-for-bug-bounty-hunters-mastering-fuzzing-9f70e5474dc5)  
- [TOOL_COMPARISON.md](TOOL_COMPARISON.md) – PSFuzz vs. ffuf, wfuzz, vaf, etc.  
- [ROADMAP.md](ROADMAP.md) – Planned features  
- [IDEAS.md](IDEAS.md) – North-star ideas  
