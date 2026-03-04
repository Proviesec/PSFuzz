# PSFuzz: Fast Web & Directory Fuzzing in a Single Go Binary

**One tool for endpoint discovery, tech fingerprinting, CORS checks, and link-driven crawling—no Python stack required.**

---

Finding hidden directories and endpoints is a staple of web security testing. You need something fast, scriptable, and capable of more than just status codes: technology hints, CORS misconfigurations, and URLs buried in responses. **PSFuzz** is a concurrent web fuzzer written in Go that does exactly that—with an ffuf-compatible CLI, built-in response modules, recursion, and a single binary you can drop anywhere.

In this article we’ll walk through what PSFuzz is, where to get it, and how to use it with practical examples—including every response module. If you prefer learning by doing, there’s also a **TryHackMe room** and the full source on **GitHub**; links are at the end.

[IMAGE: Terminal showing PSFuzz banner and a short scan output, or a simple diagram “Target → PSFuzz → Wordlist → Discovered paths + module data”]

---

## What is PSFuzz?

PSFuzz is a **concurrent web and directory fuzzer**. You give it a URL with a placeholder (e.g. `https://example.com/FUZZ`), a wordlist, and optional filters. It replaces the placeholder with each word, sends requests, and collects matches. Unlike “dumb” fuzzers that only report status and size, PSFuzz can run **response modules** on every match: technology fingerprinting, CORS analysis, URL/link extraction, and optional AI-powered verdicts.

**Why it stands out:**

- **Single binary:** Written in Go; no Python/Ruby runtime. Build once, run anywhere.
- **ffuf-compatible:** Same ideas (FUZZ placeholder, wordlists, `-mc`/`-fc`, `-w`, `-x`). Easy to switch from ffuf or use both.
- **Modules built in:** Fingerprint, CORS, urlextract, links, and AI—no second tool to chain.
- **Recursion:** Automatically scan into discovered directories with configurable depth and strategy.
- **Audit log:** Record every request/response to NDJSON for later analysis without rescanning.

**Where to find it:**

- **GitHub:** [github.com/Proviesec/PSFuzz](https://github.com/Proviesec/PSFuzz) — source, issues, releases.
- **TryHackMe:** A dedicated room walks you through installation, wordlists, filters, recursion, modules, and a hands-on challenge. Search for “PSFuzz” on TryHackMe or use the room link from the repo README once published.

---

## Install and first run

**Requirements:** Go 1.21+ (see `go.mod`). Clone the repo and build:

```bash
git clone https://github.com/Proviesec/PSFuzz.git
cd PSFuzz
go build -o psfuzz .
```

Or use the Makefile: `make build`.

A minimal first scan:

```bash
./psfuzz -u https://example.com/FUZZ -w default -o scan
```

- `-u` — Target URL; `FUZZ` is the placeholder replaced by each wordlist entry.
- `-w default` — Built-in wordlist of common paths (or use your own file, e.g. `-w wordlist.txt`).
- `-o scan` — Output base name; you get `scan.txt` (and with `-of json` you get `scan.json`).

[IMAGE: Screenshot of `scan.txt` with a few lines: URL, status, size, maybe one line with fingerprint summary]

---

## Wordlists and filters

You can stick to the built-in `default` wordlist or supply your own. Filtering keeps results manageable:

- **Status:** `-mc 200,301` only show those codes; `-fc 404` hide 404.
- **Size:** `-ms 100` minimum size; `-fs 50` filter out responses of size 50.
- **Content:** `-fr "not found"` exclude pages containing that text; `-mr "admin"` only keep matches containing “admin”.
- **Block words:** `-bw "not found,denied"` exclude responses containing any of those phrases.

Example: only 200 and 301, no 404:

```bash
./psfuzz -u https://target.example.com/FUZZ -w default -mc 200,301 -fc 404 -o scan
```

For multiple wordlists and placeholders (e.g. user + password), use keywords: `-w users.txt:USER,pass.txt:PASS` and put `USER` and `PASS` in the URL. See the repo’s [CHEATSHEET.md](https://github.com/Proviesec/PSFuzz/blob/main/CHEATSHEET.md) for more.

---

## Response modules: the real differentiator

Response modules run on **every match** that passes your filters. They attach extra data to each result—technology, CORS, URLs, or an AI verdict. Enable them with `-modules` (comma-separated); full data appears in JSON/NDJSON, and a short summary in TXT/HTML.

### 1. Fingerprint

Detects technologies from headers and body (e.g. nginx, PHP, WordPress, React). No config needed.

```bash
./psfuzz -u https://example.com/FUZZ -w default -modules fingerprint -of json -o scan
```

In `scan.json`, each result has `module_data.fingerprint.technologies` — a list of strings. Use it to map stacks and prioritize interesting tech.

[IMAGE: Snippet of JSON showing `module_data.fingerprint.technologies: ["nginx","php"]` for one URL]

### 2. CORS

Reads CORS-related headers from the response (no extra request with `Origin`). Surfaces things like `access_control_allow_origin`, `access_control_allow_credentials`, and a `potentially_permissive` hint.

```bash
./psfuzz -u https://example.com/FUZZ -w default -modules cors -of json -o scan
```

Useful for quickly spotting permissive or credential-bearing CORS configs.

### 3. urlextract

Parses URLs from the response body (regex) and the `Location` header, deduplicates and normalizes. Output is a `urls` list per result.

```bash
./psfuzz -u https://example.com/FUZZ -w default -modules urlextract -of json -o scan
```

Good for API responses or redirects that expose internal or interesting URLs.

### 4. links

Extracts HTML links from `href`, `action`, and `src`, resolves them to absolute URLs. Also outputs a `urls` list. Combined with **enqueue** (below), this becomes a built-in spider.

```bash
./psfuzz -u https://example.com/ -w default -modules links -of json -o scan
```

To **feed discovered links back into the scan queue** (link-driven discovery), use `-enqueue-module-urls links` (or `urlextract`, or both):

```bash
./psfuzz -u https://example.com/ -w default -modules links -enqueue-module-urls links -D 2 -of json -o scan
```

Discovered URLs are queued with depth +1, so you get multi-level discovery without a separate crawler.

### 5. AI

Sends status, URL, and a truncated body to the OpenAI API and attaches a short security verdict. Requires `OPENAI_API_KEY`. Customize the prompt with `-ai-prompt`; placeholders: `{{status}}`, `{{method}}`, `{{url}}`, `{{body}}`.

```bash
export OPENAI_API_KEY=sk-...
./psfuzz -u https://example.com/FUZZ -w default -mc 200 -modules ai -ai-prompt "Security expert: {{url}} status {{status}}. One sentence.\n\n{{body}}" -of json -o scan
```

Tip: use `-mc 200` (or other codes) so the AI module runs only on interesting responses and you save API calls.

---

## Recursion and depth

Recursion lets PSFuzz scan **into** discovered directories. When a URL looks like a directory (e.g. ends with `/`) and returns an “interesting” status (200, 301, 302, 403), PSFuzz can recurse into it with the same wordlist.

- **Depth:** `-D 2` = up to 2 levels of recursion (0 = base only).
- **Strategy:** `-recursion-strategy default` (recurse only on configured status codes) or `greedy` (recurse on every match).

Example:

```bash
./psfuzz -u https://example.com/FUZZ -w default -D 2 -recursion-strategy default -modules fingerprint -of json -o scan
```

Each result in the report includes `depth=N` (0 = base, 1+ = recursive). See [RECURSION.md](https://github.com/Proviesec/PSFuzz/blob/main/RECURSION.md) in the repo for details.

[IMAGE: Example lines from scan.txt or JSON showing depth=0 and depth=1 for different URLs]

---

## Presets and safety

**Presets** apply a bundle of defaults with one flag:

- **quick** — Fast smoke test (fav wordlist, time limit, fingerprint only, low concurrency).
- **stealth** — Low profile (rate limit, delay, random UA, jitter) to reduce WAF/blocking.
- **thorough** — Full discovery (higher concurrency, multiple modules, extensions, depth 2, JSON).

Example:

```bash
./psfuzz -u https://target.example.com/FUZZ -preset stealth -o scan
```

**Safety:** Safe mode is on by default: loopback and private IPs are blocked. Use `-safe=false` only for authorized local targets. For Burp or self-signed certs use `-insecure` or `-k` in lab environments only.

---

## Proxy, audit log, and output formats

- **Proxy:** Route all traffic through Burp: `-x http://127.0.0.1:8080`. Use `-replay-proxy` to send only matching requests to another proxy.
- **Audit log:** Log every request/response to NDJSON for later analysis: `-audit-log audit.ndjson` (optionally `-audit-max-body 51200` to cap body size).
- **Output:** `-of txt` (default), `json`, `html`, `csv`, `ndjson`, `ffufjson`.

Example with proxy and JSON:

```bash
./psfuzz -u https://target.example.com/FUZZ -w default -x http://127.0.0.1:8080 -of json -o scan
```

---

## Putting it together: one full example

Classic dirbust with tech detection, CORS, and recursion, output as JSON:

```bash
./psfuzz -u https://target.example.com/FUZZ -w default -D 2 -mc 200,301,302,403 -modules fingerprint,cors -of json -o scan
```

Spider-style: follow links from every page, depth 2:

```bash
./psfuzz -u https://target.example.com/ -w default -modules links -enqueue-module-urls links -D 2 -of json -o scan
```

[IMAGE: Final screenshot of scan.json or scan.txt with a few findings and module summaries]

---

## Where to go from here

- **GitHub:** [github.com/Proviesec/PSFuzz](https://github.com/Proviesec/PSFuzz) — source, CHEATSHEET, MODULES.md, RECURSION.md, config examples, and contribution guidelines.
- **TryHackMe:** The PSFuzz room gives you step-by-step tasks and questions on installation, flags, filters, recursion, modules, and a practical challenge. Ideal if you learn by doing.
- **Docs in the repo:** README (overview), CHEATSHEET (copy-paste commands), MODULES (modules in depth), RECURSION (depth and strategy), plus DOCKER and TESTING for build/CI.

Use PSFuzz only on systems you are authorized to test. Happy fuzzing.
