# PSFuzz Testing Guide

## Automated Tests

Run all tests:

```bash
go test ./...
```

Run with race detector:

```bash
go test -race ./...
```

Run vet:

```bash
go vet ./...
```

## Current Test Suites

- `internal/config/config_test.go`
  - config-file + CLI override precedence
  - range parser behavior
  - legacy flag compatibility
  - presets: quick, stealth, thorough; unknown preset error; CLI overrides preset; case-insensitive
- `internal/config/wordlist_test.go`
  - ParseWordlistSpecs (path, path:KEY, empty)
  - ResolveWordlists with local file and invalid path (error wrapping)
- `internal/engine/engine_test.go`
  - template: applyTemplate (FUZZ, #PSFUZZ#, multiple keywords), applyHeaderTemplate, urlTemplateCoversAllKeywords
  - recursion depth behavior under concurrency
  - safe-mode blocking behavior
- `internal/engine/raw_test.go`
  - parseRawRequest: valid request (POST/GET), CRLF body, invalid request line, empty file
- `internal/filter/filter_test.go`
  - status, length, content-type, block-words, min-size filters
  - FilterStatusNot, ShowStatus, regex, duplicate and near-duplicate filtering
- `internal/httpx/httpx_test.go`
  - client creation, validateTarget (safe mode, allowed hosts, ctx)
  - Do() success and context cancellation
- `internal/modules/registry_test.go`
  - Run with empty/context-done, Enabled dedup and unknown skip
  - fingerprint (headers case-insensitive), CORS, AI without API key
- `internal/output/output_test.go`
  - Write JSON with module_data (fingerprint, cors)
  - zero-request report (TXT), NDJSON
  - Write HTML (title + table), CSV (header + row), compat JSON (results + config.method)
  - unsupported format returns error

## Integration Check (manual)

```bash
printf 'admin\nlogin\n' > /tmp/psfuzz_words.txt
./psfuzz -u https://example.com -w /tmp/psfuzz_words.txt -c 5 -of json -o sample
```

Validate outputs:

- `sample.txt`
- `sample.json`

## Parameter test script (flags and modules)

The script `scripts/test_all_params.sh` runs PSFuzz with many parameter combinations (rate, delay, headers, filters, output formats, modules, audit, extracted-urls-file, enqueue, bypass, WAF, config file, presets, etc.) to verify flags and modules work. Each run is short (`-maxtime`). **Run from the project root** (where `config.example.json` and `./psfuzz` live).

```bash
# From project root: build then run
go build -o psfuzz .
./scripts/test_all_params.sh
```

Optional environment variables:

- `PSFUZZ_BIN` – binary path (default: `./psfuzz`)
- `PSFUZZ_BASE_URL` – target URL (default: `https://example.com`)
- `PSFUZZ_PARAMTEST_OUT` – output directory (default: `./paramtest_out`)
- `PSFUZZ_WORDLIST` – wordlist name (default: `fav`)

Exit code 0 = all tests passed, 1 = one or more failed. Outputs are written under `paramtest_out/` (ignored by git). You can also run `make test-params` from the project root (after `make build`).

## Benchmark: PSFuzz with 1000-word list (redirect on/off)

The script `scripts/bench_redirect_1k.sh` runs PSFuzz with a 1000-entry wordlist (`bench/wordlist_1k.txt`), with redirect following on or off, **4 runs per variant**. It writes a summary and any stderr to `bench_out_1k/`. Requires `jq` in PATH.

```bash
go build -o psfuzz .
./scripts/bench_redirect_1k.sh
```

Optional: `BENCH_TARGET=https://example.com` (default), `BENCH_OUT=./bench_out_1k`. Redirect following is off by default; the script runs both with and without `-r` for comparison.

## Recommended CI Steps

```bash
go test ./...
go test -race ./...
go vet ./...
```
