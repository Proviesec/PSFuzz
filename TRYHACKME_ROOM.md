# TryHackMe Room: PSFuzz – Web Fuzzing for Security Testing

This document provides the full content structure for a TryHackMe room about PSFuzz. Use it when creating the room in the TryHackMe content editor. Room content is in **English** (international audience).

---

## Room metadata (for room creation)

- **Room title:** PSFuzz – Fast Web & Directory Fuzzing
- **Category:** Security Tools / Web Security / Red Team
- **Difficulty:** Easy to Medium
- **Duration:** 1–2 hours (estimated)
- **Tags:** fuzzing, directory-discovery, web-security, penetration-testing, go

---

## Room description (short, for listing)

Learn PSFuzz: a concurrent web and directory fuzzer written in Go. Master wordlists, filters, recursion, response modules, and safe scanning for penetration tests and bug bounties.

---

## Room description (full, for room intro)

### Welcome to the PSFuzz room

**PSFuzz** is a high-performance web and directory fuzzer written in Go. It is designed for security testers, red teams, and bug hunters who need fast, flexible endpoint and directory discovery with minimal setup.

In this room you will:

- Understand what fuzzing is and why it matters for web security
- Install and run PSFuzz with basic and advanced options
- Use wordlists, placeholders (`FUZZ`), and filters (status, size, content)
- Enable recursion to discover nested directories automatically
- Use response modules (fingerprinting, CORS, link extraction) to enrich results
- Apply presets and safe scanning practices

**Prerequisites:** Basic familiarity with HTTP, status codes, and the command line. Optional: experience with other fuzzing tools.

**Legal reminder:** Only use PSFuzz on targets you are authorized to test.

---

## Task 1 – What is PSFuzz?

### Description

PSFuzz is a **concurrent web fuzzer** that sends many HTTP requests to a target URL by replacing placeholders (such as `FUZZ`) with values from a wordlist. It helps you discover hidden directories, files, and parameters that might be missed by normal browsing.

**Why use it?**

- **Single binary:** No Python/Ruby stack; one Go binary runs anywhere.
- **Familiar workflow:** Similar flags and workflow to other fuzzing tools (`-u`, `-w`, `-mc`, `-fc`, etc.) so you can switch easily.
- **Built-in analysis:** Technology fingerprinting, CORS checks, URL/link extraction, and optional AI verdicts—no need for a second tool.
- **Recursion:** Automatically scan into discovered directories with configurable depth and strategy.
- **Audit log:** Record every request/response to a file for later analysis without rescanning.

**Requirements:** Go 1.21+ to build from source, or use a pre-built binary.

### Questions

1. **What is the main programming language PSFuzz is written in?**  
   - **Answer format:** One word (e.g. the language name).  
   - **Answer:** `Go`

2. **What kind of tools does PSFuzz share a similar CLI and workflow with?**  
   - **Answer format:** Two words (e.g. "other fuzzing tools").  
   - **Answer:** `other fuzzing tools`

3. **Name one built-in feature that analyzes responses (e.g. technology detection or CORS).**  
   - **Answer format:** One word or short phrase (e.g. fingerprint, cors, urlextract, links, ai).  
   - **Accept any of:** `fingerprint` / `cors` / `urlextract` / `links` / `ai`

---

## Task 2 – Build and first run

### Description

You can build PSFuzz from source or use a pre-built binary. The typical workflow is:

1. Clone or download the repo (or use the room’s attack machine).
2. Build: `go build -o psfuzz .` (or `make build`).
3. Run a minimal scan: `./psfuzz -u <TARGET_URL>/FUZZ -w default -o scan`

The placeholder `FUZZ` in the URL is replaced by each word from the wordlist. The `default` wordlist is built-in (common paths). Output goes to `scan.txt` (and optionally `scan.json` with `-of json`).

**Try it:** If you have a lab target (e.g. a TryHackMe machine), run:

```bash
./psfuzz -u http://TARGET_IP/FUZZ -w default -o scan
```

Then inspect `scan.txt` for discovered paths.

### Questions

1. **What is the placeholder keyword you put in the URL so PSFuzz replaces it with wordlist values?**  
   - **Answer format:** Uppercase keyword.  
   - **Answer:** `FUZZ`

2. **Which flag do you use to set the output file base name (e.g. so results go to scan.txt)?**  
   - **Answer format:** Single flag with hyphen (e.g. -something).  
   - **Answer:** `-o`

3. **Which flag do you use to request JSON output in addition to or instead of TXT?**  
   - **Answer format:** Flag and value (e.g. -x json).  
   - **Answer:** `-of json`

---

## Task 3 – Wordlists and filters

### Description

- **Wordlist:** `-w wordlist.txt` uses a file; `-w default` uses the built-in list. You can use multiple wordlists with keywords: `-w users.txt:USER,pass.txt:PASS` and placeholders `USER`, `PASS` in the URL.
- **Match filter (status):** `-mc 200,301` only shows these status codes; `-fc 404` hides 404.
- **Size:** `-ms 100` minimum size, `-fs 50` filter out size 50.
- **Content:** `-fr "not found"` filter out responses containing that text; `-mr "admin"` only keep matches with “admin”.
- **Block words:** `-bw "not found,denied"` excludes responses containing these phrases.

**Example:** Only show 200 and 301, hide 404:

```bash
./psfuzz -u http://TARGET/FUZZ -w default -mc 200,301 -fc 404 -o scan
```

### Questions

1. **Which flag do you use to include only certain HTTP status codes (e.g. 200 and 301)?**  
   - **Answer format:** Two-letter flag (e.g. -xx).  
   - **Answer:** `-mc`

2. **Which flag do you use to exclude responses that contain a given string (e.g. "not found")?**  
   - **Answer format:** Two-letter flag.  
   - **Answer:** `-fr`

3. **How do you tell PSFuzz to exclude responses with status code 404?**  
   - **Answer format:** Flag and value (e.g. -xx 404).  
   - **Answer:** `-fc 404`

---

## Task 4 – Recursion

### Description

Recursion lets PSFuzz automatically scan **into** discovered directories. When a URL looks like a directory (e.g. ends with `/`) and returns an “interesting” status (e.g. 200, 301, 302, 403), PSFuzz can add that path as a new base and run the wordlist again.

- **Depth:** `-D 2` means up to 2 levels of recursion (0 = base only).
- **Strategy:** `-recursion-strategy default` recurses only on configured status codes; `-recursion-strategy greedy` recurses on every match.
- **Status codes for recursion:** Controlled by options like `-rsc` (e.g. `-rsc "200,301,302,403"`).

**Example:**

```bash
./psfuzz -u http://TARGET/FUZZ -w default -D 2 -recursion-strategy default -o scan
```

Results in the report include `depth=N` (0 = base, 1+ = recursive level).

### Questions

1. **Which flag sets the maximum recursion depth (e.g. 2 for two levels)?**  
   - **Answer format:** Single letter flag (e.g. -X).  
   - **Answer:** `-D`

2. **What are the two possible values for the recursion strategy flag?**  
   - **Answer format:** Two words separated by comma, lowercase.  
   - **Answer:** `default,greedy`

3. **In the report output, how is the recursion level indicated for each result?**  
   - **Answer format:** keyword=N (e.g. depth=N).  
   - **Answer:** `depth=N` or `depth=0`, `depth=1`, etc.

---

## Task 5 – Response modules

### Description

Response modules run on **every match** that passes your filters and add extra data to the report.

| Module        | Purpose |
|---------------|---------|
| **fingerprint** | Detects technologies (e.g. nginx, PHP, WordPress) from headers and body. |
| **cors**        | Evaluates CORS headers (e.g. permissive `*`, credentials). |
| **urlextract**  | Parses URLs from body and Location header. |
| **links**       | Extracts HTML links (href, action, src) and resolves to absolute URLs. |
| **ai**          | Sends status, URL, and body to OpenAI for a short security verdict (requires `OPENAI_API_KEY`). |

Enable modules with `-modules fingerprint,cors,links`. Full module data appears in JSON/NDJSON output; TXT/HTML show a short summary.

**Link-driven discovery:** Use `-enqueue-module-urls links` (or `urlextract,links`) so that discovered URLs are added to the scan queue—like a built-in spider.

### Questions

1. **Which module detects technologies (e.g. nginx, PHP) from the response?**  
   - **Answer format:** One word, lowercase.  
   - **Answer:** `fingerprint`

2. **Which flag do you use to enable multiple modules (e.g. fingerprint and cors)?**  
   - **Answer format:** Flag name with hyphen.  
   - **Answer:** `-modules`

3. **Which module requires the OPENAI_API_KEY environment variable to work?**  
   - **Answer format:** One word, lowercase.  
   - **Answer:** `ai`

4. **Which flag feeds URLs discovered by modules (e.g. links) back into the scan queue?**  
   - **Answer format:** Long flag with hyphens.  
   - **Answer:** `-enqueue-module-urls`

---

## Task 6 – Presets and safety

### Description

**Presets** apply a set of defaults with one flag:

- **quick:** Fast smoke test (fav wordlist, time limit, fingerprint only, low concurrency).
- **stealth:** Low profile (rate limit, delay, random UA, jitter) to reduce WAF/blocking.
- **thorough:** Full discovery (higher concurrency, all modules, extensions, depth 2, JSON).

Example: `./psfuzz -u http://TARGET/FUZZ -preset stealth -o scan`

**Safety:**

- **Safe mode** (`-safe=true` by default): blocks loopback, private, and link-local addresses. Use `-safe=false` only for authorized local targets.
- **TLS:** Use `-insecure` or `-k` to skip certificate verification (e.g. for Burp or self-signed certs)—only in lab environments.
- **Scope:** `-allow-hosts host1,host2` restricts which hosts are allowed.

### Questions

1. **Which preset is designed for a “low profile” to reduce WAF or blocking?**  
   - **Answer format:** One word, lowercase.  
   - **Answer:** `stealth`

2. **By default, does PSFuzz allow scanning loopback or private IP addresses?**  
   - **Answer format:** Yes or No.  
   - **Answer:** `No`

3. **Which flag do you use to skip TLS certificate verification (e.g. for Burp)?**  
   - **Answer format:** One word flag (e.g. -something).  
   - **Answer:** `-insecure` or `-k`

---

## Task 7 – Proxy, audit log, and output formats

### Description

- **Proxy:** Send all traffic through Burp or another proxy: `-x http://127.0.0.1:8080`. Use `-replay-proxy` to send only matching requests to a second proxy.
- **Audit log:** Record every request and response to a file (NDJSON) for later analysis: `-audit-log audit.ndjson`. Optionally limit body size with `-audit-max-body 51200`.
- **Output formats:** `-of txt` (default), `json`, `html`, `csv`, `ndjson`, `ffufjson`.

**Example with proxy and JSON:**

```bash
./psfuzz -u http://TARGET/FUZZ -w default -x http://127.0.0.1:8080 -of json -o scan
```

### Questions

1. **Which flag do you use to send traffic through an HTTP proxy (e.g. Burp at 127.0.0.1:8080)?**  
   - **Answer format:** Single letter flag.  
   - **Answer:** `-x`

2. **Which flag writes every request and response to an NDJSON file for later analysis?**  
   - **Answer format:** Flag name with hyphen.  
   - **Answer:** `-audit-log`

3. **Name two possible output formats (e.g. txt and one other).**  
   - **Answer format:** Two formats separated by comma, lowercase.  
   - **Accept any two of:** txt, json, html, csv, ndjson, ffufjson

---

## Task 8 – Challenge (optional)

### Description

Use a TryHackMe target (or an allowed lab machine) and:

1. Run a scan with recursion depth 2 and the default wordlist.
2. Enable the `fingerprint` and `links` modules and JSON output.
3. Find a path that returns status 200 and note the technology detected (if any) or an interesting path.

**Deliverable:** Either the full URL of one discovered path that returns 200, or the technology string reported by the fingerprint module for that URL. (Room author can define a specific flag or answer based on the lab.)

### Questions

1. **Provide the full URL of one discovered path that returns HTTP 200.**  
   - **Answer format:** Full URL (depends on lab).  
   - **Note:** Room author should replace with actual lab URL or accept pattern.

2. **What technology did the fingerprint module report for that URL (if any)?**  
   - **Answer format:** e.g. nginx, php, WordPress (depends on lab).  
   - **Note:** Room author should set expected answer based on lab.

---

## Summary for room author

- **Tasks:** 8 (Tasks 1–7 instructional, Task 8 optional challenge).
- **Questions:** 2–4 per task; answers are short (single word, flag, or short phrase).
- **Flags:** If you use a custom flag format (e.g. THM{...}), add a “Flag format” note in the room and adapt Task 8 accordingly.
- **Lab:** For Task 8, attach a vulnerable machine or use a static demo target and set the expected answers in the question editor.
- **Copy-paste:** Code blocks are ready for the room; you can add “Click to copy” in the TryHackME editor.

---

*Room content for PSFuzz. Use only on authorized targets.*
