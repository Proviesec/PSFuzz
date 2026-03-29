# PSFuzz

Fast web fuzzer in Go for directory and endpoint discovery. Single binary, common fuzzer-style CLI (`FUZZ`, `-w`, `-mc`, `-x`), with built-in response modules, recursion strategies, and optional AI-driven wordlist selection.

## Disclaimer: DONT BE A JERK!
Needless to mention, please use this tool very very carefully. The authors won't be responsible for any consequences. 

**Requirements:** Go 1.21+

## Install

```bash
go build -o psfuzz .
# or: make build
```

## Usage

```bash
./psfuzz -u https://example.com/FUZZ -w default -o scan
# → scan.txt; use -of json for scan.json
```

With modules and recursion:

```bash
./psfuzz -u https://target/FUZZ -w wordlist.txt -c 20 -D 2 -modules fingerprint,cors,links -enqueue-module-urls links -of json -o scan
```

Full flag reference: `./psfuzz -h`. Copy-paste examples: [CHEATSHEET.md](CHEATSHEET.md).

## Features

- **Familiar CLI:** `FUZZ` placeholder, wordlists, filters (`-mc`, `-fc`, `-ms`, `-fr`), proxy (`-x`), replay proxy, raw request file
- **Response modules:** fingerprint, CORS, headers, secrets, auth, AI verdict, URL/link extraction — output in TXT, JSON, HTML, CSV, NDJSON, compat JSON
- **Link-driven discovery:** enqueue URLs from HTML/body (`-enqueue-module-urls links`), depth limit
- **Explore AI:** one-shot probe + AI wordlist/extensions suggestion (OpenAI, Ollama, or Gemini); then run scan with suggested `-w`/`-e`
- **Recursion:** `-recursion-strategy default|greedy`, 403/401 bypass variants, WAF-adaptive slowdown, per-host wildcard
- **Control:** `-maxtime` / `-maxtime-job`, resume, stop on status/matches/errors, HTTP/2, VHost, audit log (NDJSON)

## Response modules

| Module | Description |
|--------|-------------|
| `fingerprint` | Tech detection (nginx, PHP, WordPress, etc.) |
| `cors` | CORS header evaluation |
| `headers` | Security headers (CSP, HSTS, X-Frame-Options, Set-Cookie) |
| `secrets` | Secret patterns in body/headers (AWS, JWT, etc.) |
| `auth` | Login form, 401, redirect-to-login, session cookies |
| `ai` | AI verdict (openai \| ollama \| gemini); `-ai-prompt`, `-ai-provider` |
| `urlextract` | URLs from body + Location |
| `links` | HTML links → absolute URLs; use with `-enqueue-module-urls links` |

Details: [MODULES.md](MODULES.md).

## Config

`-cf config.json`. Load order: config → preset → CLI. Example: `config.example.json`. Options: `auditLog`, `enqueueModuleUrls`, `extractedUrlsFile`, etc.

## Security

- **Safe mode** (default): blocks loopback, private and link-local IPs. Redirect targets validated (no `file://` or internal IPs). `-safe=false` for local use.
- **Timeouts:** `-timeout` 0 → 30s. `-max-size` 0 → 10 MiB body cap.
- **TLS:** `-insecure` / `-k` to skip certificate verification.
- **Scope:** `-allow-hosts host1,host2`. Login: `-login-url`, `-login-user`, `-login-pass` or `-login-body`.

## Documentation

| Doc | Description |
|-----|--------------|
| [CHEATSHEET.md](CHEATSHEET.md) | Commands and examples |
| [MODULES.md](MODULES.md) | Response modules |
| [RECURSION.md](RECURSION.md) | Recursion and strategy |
| [DOCKER.md](DOCKER.md) | Docker build and run |
| [TESTING.md](TESTING.md) | Tests and param script |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributing |
| [ROADMAP.md](ROADMAP.md) | Planned features |
| [IDEAS.md](IDEAS.md) | Future ideas |
| [CHANGELOG.md](CHANGELOG.md) | Release history |

CI: `go build`, `go test`, `go vet` on push/PR ([.github/workflows/ci.yml](.github/workflows/ci.yml)).

## Project layout

```
main.go              # CLI
internal/config      # Flags, config file, validation
internal/encoder     # Payload encoders (urlencode, base64, etc.)
internal/httpx       # HTTP client, safe-mode, redirect checks
internal/engine      # Task queue, workers, recursion, report
internal/filter      # Status/length/regex/dedupe
internal/llm         # LLM client (OpenAI, Ollama, Gemini) for AI/Explore
internal/output      # TXT, JSON, HTML, CSV, NDJSON, compat JSON
internal/modules     # Response analyzers
```

## License

[MIT License](LICENSE) (Copyright Proviesec)
