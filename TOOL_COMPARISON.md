# PSFuzz vs. Other Fuzzers and Resources

Comparison of PSFuzz with related tools and payload sources. Use this to see where PSFuzz fits, what it does better, and what might be missing or improved.

**Scope:** PSFuzz is a **web/directory/endpoint fuzzer** (discovery, recursion, response analysis). It is *not* a JavaScript engine fuzzer, a generic API schema fuzzer, or a payload dictionary—those are compared only for context.

---

## 1. Direct Fuzzer Comparisons

### vs. **ffuf** (Fast web fuzzer, Go) — [github.com/ffuf/ffuf](https://github.com/ffuf/ffuf)

| Aspect | ffuf | PSFuzz |
|--------|------|--------|
| **CLI / compatibility** | FUZZ keyword, wordlists, `-mc`/`-fc`, `-w`, `-request`, `-x` | Same; PSFuzz aims for ffuf-compatible flags and behaviour. |
| **Encoders** | `-enc` (e.g. urlencode, b64encode per keyword) | **Present:** `-enc KEYWORD:enc1,enc2` (urlencode, doubleurlencode, base64encode, base64decode, htmlencode, htmldoubleencode). |
| **Input** | `-w`, `-input-cmd` (external mutator, e.g. radamsa) | Wordlists + URL/remote + generated; **no `-input-cmd` / external mutator**. |
| **Autocalibration** | `-ac`, `-acc`, `-ach` (auto size/code filters) | **Missing.** Filters are manual. |
| **Interactive mode** | Pause, adjust filters, resume, save | **Missing.** |
| **Recursion** | `-recursion`, `-recursion-depth`, `-recursion-strategy` | **Present:** `-D`, `-recursion-strategy default|greedy`. |
| **Timing** | `-maxtime`, `-maxtime-job` | **Present:** Same idea. |
| **Response analysis** | None built-in | **Better:** fingerprint, CORS, urlextract, links, optional AI verdict. |
| **Audit / replay** | Replay proxy | **Better:** Full audit log (NDJSON) + replay proxy. |
| **Discovery** | Recursion only | **Better:** Link-driven discovery (`-enqueue-module-urls links`), spider-like. |
| **WAF / bypass** | — | **Better:** 403/401 bypass variants, WAF-adaptive slowdown, auto-wildcard. |
| **Session** | Manual cookies | **Better:** `-login-url` / `-login-body` for session cookies. |
| **Presets** | — | **Better:** `-preset quick|stealth|thorough`. |

**Summary:** PSFuzz is stronger on **response modules**, **audit log**, **link-driven discovery**, **WAF/403 handling**, and **login/session**. It currently lacks **encoders**, **input-cmd**, **autocalibration**, and **interactive mode**.

---

### vs. **wfuzz** (Web application fuzzer, Python) — [github.com/xmendez/wfuzz](https://github.com/xmendez/wfuzz)

| Aspect | wfuzz | PSFuzz |
|--------|-------|--------|
| **Runtime** | Python, pip install | Single Go binary. |
| **FUZZ / payloads** | FUZZ keyword, payloads = “sources of data”, plugins | FUZZ (and custom keywords), wordlists; no plugin system. |
| **Encoders** | Encoder chains (e.g. urlencode) | **Missing.** |
| **Scope** | General web fuzzing (params, auth, dirs, headers) | Dirs/endpoints, recursion, VHost, POST/headers; no dedicated “parameter name” fuzzing mode. |
| **Response analysis** | Plugins (Python) | Built-in modules (fingerprint, CORS, AI, urlextract, links). |
| **Output** | Various (e.g. JSON) | TXT, JSON, HTML, CSV, NDJSON, FFUF-JSON. |
| **Recursion / discovery** | Depends on usage | Recursion + link enqueue. |

**Summary:** PSFuzz offers **no Python dependency**, **built-in modules**, **audit log**, and **link-driven discovery**. wfuzz is more flexible via plugins and encoder chains; PSFuzz is simpler to deploy and run.

---

### vs. **vaf** (Web fuzzer, Nim) — [github.com/andreiverse/vaf](https://github.com/andreiverse/vaf)

| Aspect | vaf | PSFuzz |
|--------|-----|--------|
| **FUZZ** | Yes | Yes. |
| **Filters** | Status, grep, notgrep, reflexive (word reflected in page) | Status, size, words, lines, regex, block-words, interesting-strings, near-duplicate. |
| **Recursion** | — | **Yes:** depth, strategy (default/greedy). |
| **Response analysis** | — | **Yes:** fingerprint, CORS, urlextract, links, AI. |
| **Audit / replay** | — | Audit log + replay proxy. |
| **WAF / bypass** | — | 403 bypass, WAF-adaptive, auto-wildcard. |
| **Presets / login** | — | Presets, login session. |

**Summary:** PSFuzz has a **broader feature set** (recursion, modules, audit, WAF handling, presets, login). vaf is lighter and has “reflexive” (reflection) matching, which PSFuzz could add as a filter or module.

---

## 2. Rule- and API-Focused Tools (Different Niche)

### **qsfuzz** (Query string fuzz, Go) — [github.com/ameenmaali/qsfuzz](https://github.com/ameenmaali/qsfuzz)

- **Focus:** Query string fuzzing with **YAML rules**: inject value → define success (response content/code/header). Heuristics (e.g. `'` vs `''` for SQLi).
- **PSFuzz:** Generic FUZZ replacement; no “rule” concept (“inject X, success iff response matches Y”). You can emulate with `-mr`/`-fr` but not per-rule expectations.
- **Gap:** A **rule-based injection mode** (or a module that evaluates “injection + expected response”) could complement PSFuzz for parameter/vuln testing.

### **fuzzapi** (REST API pentesting, Rails + API_Fuzzer) — [github.com/Fuzzapi/fuzzapi](https://github.com/Fuzzapi/fuzzapi)

- **Focus:** REST API fuzzing with a **web UI**; uses API_Fuzzer gem.
- **PSFuzz:** CLI only; can fuzz APIs with `-X POST -d "key=FUZZ"` or `-request`, but no UI and no schema-driven API discovery.
- **Gap:** Out of scope for a CLI fuzzer; integration with API schemas (OpenAPI) could be a future idea (see IDEAS.md).

---

## 3. Payload and Wordlist Sources (Not Fuzzers)

These are **dictionaries and payload lists**. PSFuzz does not replace them; it **consumes** them via `-w <path>`.

| Resource | Description | Use with PSFuzz |
|----------|-------------|-----------------|
| **fuzzdb** | Attack patterns, discovery paths, regex for responses | `-w /path/to/fuzzdb/discovery/...` or custom list. |
| **IntruderPayloads** | Burp Intruder/BurpBounty payloads, fuzz lists, uploads | `-w /path/to/IntruderPayloads/...` |
| **fuzz.txt** (Bo0oM) | Potentially dangerous file names | `-w /path/to/fuzz.txt` (or merge into your wordlist). |

**Possible improvements:**

- Document recommended wordlist sources (fuzzdb, SecLists, fuzz.txt, IntruderPayloads) in README or CHEATSHEET.
- Optional “wordlist presets” that point to well-known repos (e.g. clone and use a specific fuzzdb path) if you want to avoid bundling.

---

## 4. Out-of-Scope (Different Domain)

| Tool | Purpose | Relation to PSFuzz |
|------|---------|--------------------|
| **fuzzilli** | JavaScript engine fuzzer (coverage-guided, FuzzIL, Swift) | Different domain (interpreter/browser fuzzing). No direct comparison. |

---

## 5. What PSFuzz Does Better (Summary)

- **Single binary, ffuf-compatible:** No Python/Ruby; same mental model as ffuf.
- **Response modules:** Fingerprint, CORS, urlextract, links, optional AI verdict—no second tool.
- **Audit log:** Full request/response NDJSON for post-run analysis.
- **Link-driven discovery:** Enqueue URLs from HTML/body; spider-like without a separate crawler.
- **Recursion:** Depth + strategy (default vs greedy).
- **WAF/403 handling:** Bypass variants, adaptive slowdown, auto-wildcard.
- **Session:** Login once, cookies applied to all requests.
- **Presets:** quick / stealth / thorough.
- **Output:** Multiple formats including FFUF-JSON for compatibility.

---

## 6. What Might Be Missing or Improved

| Feature | Priority | Notes |
|---------|----------|--------|
| **Payload encoders** | High | e.g. urlencode, base64 per keyword (like ffuf `-enc`). |
| **Autocalibration** | Medium | Auto-suggest or apply size/code filters from a few probes (like ffuf `-ac`). |
| **Reflexive filter** | Low | “Match only if FUZZ value is reflected in response” (like vaf). |
| **Rule-based injection** | Idea | YAML-style rules: inject X, success if response matches Y (qsfuzz-style). |
| **Interactive mode** | Idea | Pause, change filters, resume (ffuf-style). |
| **Input command / mutator** | Idea | `-input-cmd` to generate payloads externally (e.g. radamsa). |
| **Wordlist docs** | Low | Link fuzzdb, SecLists, fuzz.txt, IntruderPayloads in README/CHEATSHEET. |

---

## References

- [ffuf](https://github.com/ffuf/ffuf) – Fast web fuzzer written in Go  
- [wfuzz](https://github.com/xmendez/wfuzz) – Web application fuzzer (Python)  
- [fuzzdb](https://github.com/fuzzdb-project/fuzzdb) – Attack patterns and discovery dictionaries  
- [IntruderPayloads](https://github.com/1N3/IntruderPayloads) – Burpsuite Intruder/BurpBounty payloads and lists  
- [fuzz.txt](https://github.com/Bo0oM/fuzz.txt) – Potentially dangerous files  
- [fuzzilli](https://github.com/googleprojectzero/fuzzilli) – JavaScript engine fuzzer  
- [fuzzapi](https://github.com/Fuzzapi/fuzzapi) – REST API pentesting (Rails UI)  
- [qsfuzz](https://github.com/ameenmaali/qsfuzz) – Query string fuzz with YAML rules  
- [vaf](https://github.com/andreiverse/vaf) – Web fuzzer in Nim  
