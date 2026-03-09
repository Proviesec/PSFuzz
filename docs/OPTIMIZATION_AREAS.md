# PSFuzz — Optimization Areas (Architecture, Go, Security)

Analysis from three perspectives: **Software Architecture**, **Go expertise**, and **Cybersecurity**. Each section lists concrete improvements with impact and effort.

---

## 1. Software Architecture

### 1.1 Config as a God Object

**Current:** `config.Config` has 150+ fields. It is loaded, validated, and passed into engine, filter, httpx, output, and modules. Every new feature adds more fields. *Future idea: see [IDEAS.md](../IDEAS.md#architecture-future-improvements).*

**Impact:** Hard to test (need to fill many fields), easy to create invalid combinations, unclear which parts of config are used where.

**Recommendations:**

- **Split by domain:** Introduce smaller structs that are composed, e.g. `config.ScanConfig` (URLs, wordlist, concurrency, depth), `config.FilterConfig` (status/length/words, regex, duplicates), `config.HTTPConfig` (timeout, proxy, headers, auth), `config.OutputConfig`, `config.ModuleConfig` (already exists). Top-level `Config` embeds or references these. Validation and defaults stay next to each sub-config.
- **Optional builder or presets:** For tests and programmatic use, offer `config.NewScanConfig()` with minimal required fields and optional setters, so tests don’t depend on the full flat struct.

### 1.2 Engine.Run() Monolith

**Current:** `Run()` in `engine.go` is long and does: wordlist resolution, calibration, resume load, report init, login, audit file, channel setup, producer goroutine, N workers, resume checkpoint, final report assembly. All in one function.

**Impact:** Hard to unit-test individual phases, difficult to add features (e.g. different producer strategies) without touching the same blob.

**Recommendations:**

- **Extract phases:** e.g. `prepareRun(ctx, cfg) (*runParams, error)` for wordlists, calibration, resume, report init; `runLoop(ctx, params) (*Report, error)` for producer + workers + shutdown. `Run()` becomes a short orchestration that calls these. Each phase can be tested and evolved separately.
- **Explicit run state:** Introduce a `runParams` or `ScanState` struct that holds `visited`, `mu`, `inflight`, `report`, `st` (runState), etc. Pass this through instead of many separate arguments and closures. Reduces the number of variables captured in `enqueue` and worker logic.

### 1.3 Module System and Config Coupling

**Current:** Modules get `modules.Config` (AIPrompt, AIProvider, etc.). Engine calls `modules.Enabled(&cfg.ModuleConfig)` and caches analyzers. Clean interface (`Analyzer`), but module config is still one big bag.

**Recommendation:** Keep the `Analyzer` interface. Optionally allow each module to declare required config (e.g. a small struct or validated options) so that `Enabled()` and module init can validate once at startup and avoid nil or invalid config at analyze time.

### 1.4 Clear Package Boundaries

**Current:** `internal/engine` imports config, encoder, filter, httpx, modules. It knows about report format, resume format, and output-related concepts. Largely one-way: engine → others.

**Recommendations:**

- Keep engine free of `output` and file formats (report struct is fine as a data DTO). All writing stays in `output` with only `Report` and `Config` passed in.
- Consider a small `internal/scan` package that owns only “run a scan and return a Report” (orchestration + engine), and have `main` or a thin CLI layer do config load, output write, and extracted-URLs file. That keeps “what we found” separate from “how we persist it.”

---

## 2. Go Expertise

### 2.1 Config Mutation During Run

**Current:** In `applyAutoCalibration`, the code does `addExactRange(&e.cfg.FilterLengthNot, length)` and similar for `FilterWordsNot` and `FilterLinesNot`. So the shared `config.Config` is mutated during the scan.

**Impact:** If config were ever reused for a second run (e.g. in a server or test), the second run would see the first run’s calibration. It also makes reasoning about “config at start” harder.

**Recommendation:** Either copy the relevant filter slices at the start of `Run()` and mutate the copies, or introduce an explicit “runtime filter state” that calibration updates instead of `e.cfg`.

### 2.2 Error Wrapping Consistency

**Current:** Some places use `fmt.Errorf("...: %w", err)`, others `errors.New("...")` or `fmt.Errorf("...")` without `%w`. So error chains are inconsistent.

**Recommendation:** Where an underlying error exists, always use `%w` so callers can use `errors.Is` / `errors.As`. Reserve `errors.New` for leaf errors with no cause.

### 2.3 Interfaces for Testability

**Current:** Engine uses `*httpx.Client` and calls `e.client.Do`, `e.client.Replay`, `e.client.SetScale`, `e.client.Close`. There is a `Doer` interface but it’s not the main dependency.

**Recommendation:** Define a small interface used by engine, e.g. `type HTTPDoer interface { Do(ctx, spec) (*DoResult, error); Close() }`, and optionally extend it with `SetScale(float64)` and `Replay(...)` if engine needs them. Engine then depends on this interface; tests can inject a mock or fake server. Same idea for “write report” if you ever want to test without writing to disk: depend on an interface that takes `*Report` and returns an error.

### 2.4 Producer Shutdown

**Current:** The producer closes the task channel only after all enqueued tasks have completed, using a `sync.WaitGroup` (no spin-loop). *Implemented.*

**Recommendation:** Use a `sync.WaitGroup`: workers call `wg.Done()` when they finish a task (or when they exit). Producer (or a small coordinator) calls `wg.Wait()` in a goroutine and then closes the channel, or use a “done” channel that the last worker closes so the producer can `select` on it. This avoids busy-wait and is easier to reason about.

**Current:** `MaxTime` and `MaxTimeJob` are applied; context is passed through. Good.

**Recommendation:** Ensure `cfg.Timeout` is never 0 when building the HTTP client (or document that 0 means “no timeout”). Validation in `validateConfig` can enforce `Timeout > 0` (or a sentinel like “use default”) to avoid accidental unbounded waits.

---

## 3. Cybersecurity

### 3.1 Safe Defaults and Validation

**Current:** SafeMode blocks loopback and private IPs. Redirect targets are validated (SafeMode/scope, http(s) only). Timeout and MaxResponseSize have defaults/caps. See [README.md](../README.md#security) for the user-facing description.

### 3.2 Secrets in Config and Audit Log

**Current:** Proxy credentials, Basic Auth, and login credentials live in `Config`. Audit log writes request headers (including possibly `Authorization`, `Cookie`, or custom headers with secrets) and body.

**Recommendations:**

- **Audit log:** Redact or omit sensitive headers when writing the audit entry (e.g. same list as in explore AI: `Authorization`, `Cookie`, `X-Api-Key`, etc.). Optionally redact request/response body when it looks like a login form or token.
- **Config and env:** Prefer reading proxy/auth secrets from environment variables when possible (e.g. `PROXY_USER` / `PROXY_PASS`), and document that saving config with `-save-config` should not persist secrets (or persist placeholders like `$PROXY_PASS`).

### 3.3 Scope and AllowedHosts

**Current:** `AllowedHosts` is an allowlist; if set, only those hosts are allowed. Good.

**Recommendations:**

- Document that when using a proxy, the proxy itself is not in scope; only the target host is validated. Ensure DNS resolution for the target host (in SafeMode) uses the same view as the tool (e.g. no surprise split-horizon that could allow bypass).
- If you add “scope by URL prefix” later, validate that redirects stay within the same scheme and host (or within a configured scope), not just “any host in allowed list.”

### 3.4 Resilience Against Malicious Targets

**Current:** Body size is limited by `MaxResponseSize`; explore probe uses `readBodyWithLimit`. Good.

**Recommendations:**

- Ensure every HTTP response body read path uses a limit (no raw `io.ReadAll` without `LimitReader` or equivalent).
- For wordlists fetched from URLs (e.g. explore or default wordlist), enforce a maximum download size (e.g. 50 MiB) to avoid a malicious URL serving a 10 GB “wordlist.”
- Rate limiting: you already have ThrottleRPS and WAF adaptive. Consider a global “max requests per host per minute” option to reduce the risk of accidentally DoS’ing a single host.

### 3.5 Dependency and Supply Chain

**Current:** Few dependencies (golang.org/x/net, golang.org/x/text). Good.

**Recommendations:**

- Pin versions in `go.mod` and run `go mod verify` in CI.
- Optionally add a `go.sum` check or Dependabot/Renovate for dependency updates, and document that users should re-build from a trusted source rather than run pre-built binaries from untrusted locations.

---

## 4. Performance / Concurrency

### 4.1 Default Concurrency

**Current:** PSFuzz uses **40 concurrent workers** by default (`-c 40`), which gives good throughput for directory and path fuzzing. Use `-c` to override (e.g. lower for stealth, higher for local or rate-limited targets). Presets override this: quick and stealth use 5, thorough uses 30.

---

## Summary Table

| Area              | Priority | Effort | Suggestion                                      |
|-------------------|----------|--------|-------------------------------------------------|
| Architecture      | High     | Medium | Split Config into domain sub-structs            |
| Architecture      | Medium   | Medium | Break up Engine.Run into phases + run state     |
| Go                | Medium   | Low    | Stop mutating e.cfg in applyAutoCalibration     |
| Go                | Low      | Low    | Consistent error wrapping with %w               |
| Go                | Medium   | Low    | HTTP client behind interface for tests          |
| Security          | High     | Low    | Redact sensitive headers in audit log           |
| Security          | Low      | Low    | Limit wordlist download size from URLs          |

Implementing the high-priority security items and the config-mutation fix gives the best payoff with limited effort. Larger refactors (config split, Run() phases) are tracked as **future architecture ideas** in [IDEAS.md](../IDEAS.md#architecture-future-improvements).
