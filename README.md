# PSFuzz 1.0

Concurrent web fuzzer in Go for endpoint and directory discovery. ffuf-compatible CLI with extra features (timing, recursion strategy, HTTP/2, VHost, response-analysis modules).

**Requirements:** Go 1.21+ (see `go.mod`). Build from source; or use a pre-built binary if provided for your platform.

## Build and Run

```bash
go build -o psfuzz .
./psfuzz -u https://example.com
```

Or with Makefile: `make build` then `./psfuzz -u https://example.com`.

## Get started in 3 commands

```bash
go build -o psfuzz .                    # build
./psfuzz -u https://example.com/FUZZ -w default -o scan   # run (replace URL with your target)
# → results in scan.txt; use -of json for scan.json
```

Need more control? `./psfuzz -h` for all flags. **Presets:** `-preset quick` (fast smoke), `-preset stealth` (low profile), `-preset thorough` (full discovery). Examples: [CHEATSHEET.md](CHEATSHEET.md).

## What makes PSFuzz 1.0 stand out

- **Single binary, ffuf-compatible:** Same idea as ffuf (FUZZ, wordlists, filters), same flag names where it makes sense (`-mc`, `-fc`, `-w`, `-request`, `-x`, `-replay-proxy`). No Python/Ruby stack required.
- **Response modules built in:** Technology fingerprinting, CORS check, URL/link extraction, optional OpenAI verdict—no second tool. Results in all output formats (JSON, HTML, NDJSON).
- **Audit log:** Write every request/response to a file (NDJSON); filter or analyze after the run without rescanning.
- **Link-driven discovery:** Feed links from HTML (and URLs from body) into the scan queue (`-enqueue-module-urls links`), limit depth—spider mode without a separate tool.
- **Explore AI:** Probe the base URL once (fingerprint + headers), ask OpenAI for a **wordlist and extensions recommendation** (e.g. WordPress → wordpress paths, TYPO3 → typo3). One flag (`-explore-ai`); requires `OPENAI_API_KEY`. Then run your scan with the suggested options.
- **Recursion with strategy:** `default` (only on configured status codes) or `greedy` (follow every match). Bypass variants for 403/401, WAF-adaptive slowdown, auto-wildcard per host.
- **Timing & control:** Max duration overall and per task, resume, stop on status/matches/errors. HTTP/2, VHost fuzzing, proxy and replay proxy.

Details and examples: [CHEATSHEET.md](CHEATSHEET.md), [MODULES.md](MODULES.md), [RECURSION.md](RECURSION.md).

## Quick reference

**See [CHEATSHEET.md](CHEATSHEET.md)** for copy-paste commands and ffuf-style examples.

## Main usage

```bash
./psfuzz -u https://target.tld -w default -c 20 -D 2 -of json -o scan
```

Output files (base name `scan`): `scan.txt`, `scan.json`, `scan.html`, `scan.csv`, `scan.ndjson` (depending on `-of`).

## Architecture

- `main.go` – CLI entrypoint
- `internal/config` – flags, config file, validation
- `internal/httpx` – HTTP client, retries, throttling, safe-mode
- `internal/filter` – status/length/content/regex/dedupe
- `internal/engine` – task queue, workers, recursion, report
- `internal/output` – TXT, JSON, HTML, CSV, NDJSON, FFUF-JSON
- `internal/modules` – response analyzers (fingerprint, cors, ai, urlextract, links)

## Flags (overview)

- **Wordlists:** `-w`, `-w list.txt:FUZZ`, `-e`, `-ext-defaults`, `-mode`, `-list`, `-enc` (payload encoders)
- **Filtering:** `-mc`/`-fc`, `-ms`/`-fs`, `-mw`/`-fw`, `-nd`, `-bw`, `-is`, `-fr`
- **HTTP:** `-X`, `-data` (body or `@file`), `-H`, `-request`, `-x` (proxy), `-http2`, `-vhost`, `-insecure` / `-k` (skip TLS verify)
- **Timing:** `-maxtime`, `-maxtime-job`, `-timeout` (request timeout), `-recursion-strategy default|greedy`, `-rsc` (recursion status codes)
- **Modules:** `-modules fingerprint,cors,ai,urlextract,links`, `-ai-prompt "..."`, `-ai-provider openai|ollama|gemini`, `-enqueue-module-urls urlextract,links`, `-extracted-urls-file <path>` (write all extracted URLs to file, one per line)
- **Explore AI:** `-explore-ai` — probe base URL, get wordlist/extensions recommendation from OpenAI (requires `OPENAI_API_KEY`). Use `-explore-ai-wordlists-dir <dir>` to auto-select from local files, or `-explore-ai-wordlist name:path_or_url` to use your own lists/URLs (e.g. SecLists). See [CHEATSHEET.md](CHEATSHEET.md#explore-ai) for example wordlist URLs.
- **Audit:** `-audit-log <path>`, `-audit-max-body <bytes>` (NDJSON request/response log)
- **Output:** `-of txt|json|html|csv|ndjson|ffufjson`, `-o`, `-save-config`
- **Resume/Stop:** `-resume`, `-resume-every`, `-sf`, `-se`, `-sa`
- **Login:** `-login-url`, `-login-user`, `-login-pass` (or `-login-body` for custom form); session cookies applied to all requests.
- **Presets:** `-preset quick|stealth|thorough` (apply a set of defaults; CLI overrides)
- **Other:** `-D` (depth), `-c` (concurrency), `-safe`, `-allow-hosts`, `-cf` (config file)

Full list: `./psfuzz -h`.

## Response modules

| Module       | Description |
|-------------|-------------|
| `fingerprint` | Technology detection (headers/body): e.g. nginx, PHP, WordPress. |
| `cors`      | CORS header evaluation (e.g. permissive `*`, credentials). |
| `ai`        | AI security verdict (status + URL + body). Provider: **openai** (default) \| **ollama** \| **gemini**. Keys: `OPENAI_API_KEY`, or `GEMINI_API_KEY`/`GOOGLE_API_KEY`, or none for Ollama. Custom prompt: `-ai-prompt` or config `aiPrompt`; placeholders: `{{status}}`, `{{method}}`, `{{url}}`, `{{body}}`. |
| `urlextract` | Parses URLs from response body and Location header; deduplicates. Output: `urls` list per result. |
| `links` | Extracts HTML links (`href`, `action`, `src`), resolves to absolute URLs. Output: `urls` list. Use with `-enqueue-module-urls links` to enqueue discovered URLs. |

Example: `./psfuzz -u https://example.com/FUZZ -w wordlist.txt -modules fingerprint,cors,ai -of json -o scan`

**Details and usage:** [MODULES.md](MODULES.md) — when modules run, where data appears (TXT vs JSON), and how to combine with filters and AI.

## Config file

Use `-cf config.json`. A minimal `config.example.json` is included; copy and edit as needed. Load order: config file (if `-cf`) → preset (if `-preset`) → CLI; CLI overrides the rest. Example:

```json
{
  "url": "https://example.com",
  "dirlist": "default",
  "concurrency": 20,
  "depth": 2,
  "output": "scan",
  "outputFormat": "json",
  "safeMode": true,
  "modules": "fingerprint,cors,ai,urlextract,links",
  "maxtime": 0,
  "maxtimeJob": 0,
  "recursionStrategy": "default",
  "http2": false,
  "vhost": false,
  "aiPrompt": "",
  "aiProvider": "openai",
  "aiEndpoint": "",
  "aiModel": "",
  "auditLog": "",
  "auditMaxBodySize": 0,
  "enqueueModuleUrls": "",
  "extractedUrlsFile": ""
}
```

- **auditLog:** Path to NDJSON file for full request/response audit (empty = off).
- **auditMaxBodySize:** Max body size per audit entry in bytes (0 = unlimited).
- **enqueueModuleUrls:** Comma-separated module names whose `urls` output is queued for scanning (e.g. `urlextract,links`).
- **extractedUrlsFile:** Path to write all extracted URLs (from any module with `urls` output) to a file, one per line; no enqueue. Empty = off.

## Security

- **Safe mode** (`-safe=true` by default): blocks loopback/private/link-local. Use `-safe=false` for local or authorized targets only.
- **TLS skip:** `-insecure` or `-k` skips TLS certificate verification (e.g. self-signed certs, Burp MITM).
- **Login:** `-login-url https://target/login -login-user admin -login-pass secret` performs one login request and uses the session cookies for all fuzz requests. Use `-login-body` for custom form fields or JSON.
- Optional scope: `-allow-hosts host1,host2`.
- Retries: `-retry`, `-retry-backoff-ms`.

## Further docs

- **[CHEATSHEET.md](CHEATSHEET.md)** – main command reference
- **[MODULES.md](MODULES.md)** – response modules (fingerprint, CORS, AI): usage and evaluation
- **[RECURSION.md](RECURSION.md)** – recursion and `-recursion-strategy`
- **[FUZZING_GUIDES.md](FUZZING_GUIDES.md)** – mapping of bug-bounty fuzzing guides to PSFuzz; gaps and feature ideas
- **[TOOL_COMPARISON.md](TOOL_COMPARISON.md)** – comparison with ffuf, wfuzz, and other fuzzers
- **[DOCKER.md](DOCKER.md)** – Docker build and run
- **[TESTING.md](TESTING.md)** – how to run tests, parameter test script (`scripts/test_all_params.sh`), and what is covered
- **[CONTRIBUTING.md](CONTRIBUTING.md)** – how to contribute  
  **CI:** GitHub Actions runs `go build`, `go test`, and `go vet` on push and pull requests (see [.github/workflows/ci.yml](.github/workflows/ci.yml)).
- **[ROADMAP.md](ROADMAP.md)** – planned features (e.g. response modules for later releases)
- **[IDEAS.md](IDEAS.md)** – north-star ideas (reporting, UX, integrations) for future “extra class” features
- **[CHANGELOG.md](CHANGELOG.md)** – release history

## Legal

Use only on systems you are authorized to test.
