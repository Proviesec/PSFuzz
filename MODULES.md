# Response Modules

Response modules analyze **every match** that passes the filters (status, length, content-type, etc.). For each match, all enabled modules run; their results are attached to that result and appear in the report.

## Enabling modules

- **CLI:** `-modules fingerprint,cors,ai` (comma-separated; names are case-insensitive)
- **Config file:** `"modules": "fingerprint,cors,ai"`

Unknown names are ignored; duplicates are only run once.

## Available modules

| Module       | Description | Output (example) |
|-------------|-------------|------------------|
| **fingerprint** | Detects technologies from response headers and body (e.g. nginx, PHP, WordPress, React). | `technologies`: list of detected technologies |
| **cors**     | Reads CORS headers from the response (no separate request with `Origin`). | `access_control_allow_origin`, `access_control_allow_credentials`, `potentially_permissive`, etc. |
| **ai**       | Sends status, URL, and (truncated) body to an AI backend; returns a short security assessment. Provider: **openai** (default) \| **ollama** \| **gemini**. Keys: `OPENAI_API_KEY`, or `GEMINI_API_KEY`/`GOOGLE_API_KEY`, or none for Ollama. | `verdict`: response text |
| **urlextract** | Parses URLs from the response body and Location header; deduplicates and normalizes. Stored per result in the report. | `urls`: list of URLs (strings) |
| **links** | Extracts links from HTML (`href`, `action`, `src`), resolves to absolute URLs, deduplicates. Use with `-enqueue-module-urls links` to enqueue discovered URLs. | `urls`: list of URLs (strings) |
| **headers** | Evaluates security-related response headers: Content-Security-Policy, X-Frame-Options, Strict-Transport-Security, X-Content-Type-Options; Set-Cookie (Secure, HttpOnly). Flags missing or weak values. | `missing_headers`, `weak_headers`, `issues`, `cookie_has_secure`, `cookie_has_httponly`, etc. |
| **secrets** | Scans response body (and Authorization header) for common secret patterns: AWS keys, JWTs, GitHub/Slack tokens, `password=` in response, generic api_key/secret. Reports finding types only; no secret values stored. | `findings`: list of types (e.g. aws_access_key, jwt, password_in_response), `count` |
| **auth** | Detects auth-related responses: login form (form + password field + login/signin), logout link, 401, 302 to login, "session expired" / "please log in" text, Set-Cookie with session/auth-like name. Helps prioritize auth flows. | `hints`: list (e.g. login_form, logout_present, status_401, redirect_to_login, session_or_login_required, auth_cookie_set), `count` |

### AI module in detail

- **Providers:** `-ai-provider openai | ollama | gemini`. Default: **openai**. Ollama uses local server (default `http://localhost:11434`); no API key. OpenAI needs `OPENAI_API_KEY`; Gemini needs `GEMINI_API_KEY` or `GOOGLE_API_KEY`. Override URL with `-ai-endpoint`, model with `-ai-model`, token limit with `-ai-max-tokens` (0 = default 150).
- **Error handling:** If the API key is missing or the API call fails, the module does not crash: it returns structured data in `module_data.ai`: `"skipped": "reason"` (e.g. OPENAI_API_KEY not set) or `"error": "api", "message": "..."`. So you can see in the JSON/HTML report why a verdict is missing.
- **Prompt:** Default prompt (security-focused) or custom text via `-ai-prompt` or config `aiPrompt`.
- **Placeholders:** `{{status}}`, `{{method}}`, `{{url}}`, `{{body}}` are replaced.
- **Body:** Truncated to 3000 characters to limit API cost and latency.

Example custom prompt:

```bash
-ai-prompt "Security expert: status {{status}}, {{url}}. Anything unusual? One sentence.\n\n{{body}}"
```

## Where module data appears

- **TXT, CSV, HTML:** Each row shows a **short summary** of module data (e.g. `fingerprint: technologies=nginx,php | cors: potentially_permissive`).
- **JSON, NDJSON, FFUF-JSON:** Each result includes the **full** `module_data` (object per module with all fields). Ideal for evaluation or filtering by technologies, CORS, or AI verdict.

**Recommendation:** Use `-of json` (or `ndjson`) when you need full module data; then all module output is available per URL in a structured form.

### Enqueue URLs from modules

With **`-enqueue-module-urls`** (or config `enqueueModuleUrls`) you can feed URLs discovered by certain modules back into the scan queue. Only modules that output a `urls` field (list of strings) are supported, e.g. **urlextract** and **links**.

- **urlextract:** URLs from body (regex) and `Location` header.
- **links:** URLs from HTML attributes `href`, `action`, `src` (resolved against the request URL).

Example: discover directories by following links from each page:

```bash
./psfuzz -u https://example.com/ -w wordlist.txt -modules links -enqueue-module-urls links -D 2 -of json -o scan
```

Enqueued URLs get the same scope/visited handling as recursion (depth +1, same method/headers).

## Using modules effectively

1. **Analyze only certain status codes**  
   With `-mc 200` (or other status codes), modules run only for those matches—less work and, for AI, fewer API calls.

2. **Combine with filters**  
   Filters (status, length, content-type, block-words) reduce the number of matches; fewer matches mean fewer module runs.

3. **Use AI sparingly**  
   AI per request consumes API calls. Consider running it only for 200 responses (`-mc 200 -modules ai`) or, after an initial run with `fingerprint,cors`, a second run with `ai` only on interesting paths.

4. **Full module data**  
   For scripts or post-processing: `-of json -o scan` → in `scan.json`, each entry has full data under `module_data` (e.g. `module_data.fingerprint.technologies`, `module_data.ai.verdict`).

## Examples

```bash
# Technology detection + CORS for all matches, output with full module_data
./psfuzz -u https://example.com/FUZZ -w wordlist.txt -modules fingerprint,cors -of json -o scan

# Only 200 responses with fingerprint and AI (fewer API costs)
./psfuzz -u https://example.com/FUZZ -w wordlist.txt -mc 200 -modules fingerprint,ai -of json -o scan

# Custom AI prompt from config
# In config.json: "modules": "ai", "aiPrompt": "Brief check: {{url}} Status {{status}}. Anything notable?\n\n{{body}}"
./psfuzz -u https://example.com/FUZZ -w wordlist.txt -cf config.json -of json -o scan
```

## Adding a new module (for developers)

To add a new response-analysis module:

1. **Implement the `Analyzer` interface** in `internal/modules/`:
   - `Name() string` – lowercase, short name (e.g. `"urlextract"`).
   - `Analyze(ctx context.Context, in Input) (Output, error)` – use `in.URL`, `in.Body`, `in.Headers`, etc.; return `Output{Data: map[string]any{...}}`. Return `Output{Data: nil}` when there is nothing to report (no error). Respect `ctx.Done()` for cancellation.

2. **Register the module** in your new file with `init()` – no need to edit `registry.go`:
   - Call `Register("modulename", func(c *Config) Analyzer { return YourAnalyzer{} })` in `func init()`. If your module needs config (e.g. a prompt), use `func(c *Config) Analyzer { return YourAnalyzer{Option: c.SomeField} }`.
   - **If your module needs a CLI flag or config file option** (e.g. custom prompt, API key): add the field to `internal/config` (in `cliConfig` and in `applyCLIConfig` so it is applied to `cfg.ModuleConfig` or the main config), and add it to `modules.Config` in `internal/modules/config.go`. Then pass it in the factory, e.g. `return YourAnalyzer{Option: c.YourOption}`. Example: the `ai` module uses `AIPrompt` from config.

   - **If your module needs to call an LLM (OpenAI, Ollama, Gemini):** use the shared **`internal/llm`** package. It provides `llm.Provider`, `llm.Config`, `llm.Message`, `llm.GetAPIKey(provider)`, and `llm.Call(ctx, cfg, apiKey, messages)`. Add your config (provider, endpoint, model, max tokens) to `modules.Config` and wire flags in `internal/config`. See `internal/modules/ai.go` for a full example. New modules can reuse the same layer without duplicating API logic.

3. **Output format:** Put only JSON-serializable values in `Data` (e.g. `string`, `[]string`, `bool`, `float64`). This data appears in the report per result and in all output formats (TXT summary, JSON/NDJSON/FFUF-JSON in full).

4. **Best practices:**
   - Keep the module stateless or use only config passed at creation (e.g. `AIAnalyzer{Prompt: cfg.AIPrompt}`).
   - Limit work per response (e.g. cap body size, timeouts for external calls).
   - On error, return the error; the runner skips that module for that result but continues others.
   - Add a test in `modules_test.go` (e.g. `TestYourAnalyzer_...`).

5. **Document:** Add the module to the "Available modules" table above and, if needed, to README and CHEATSHEET.

**Minimal example:** See `internal/modules/urlextract.go` (URL parsing from body/headers) or `internal/modules/cors.go` (header-only).

## Technical notes (for developers)

- Modules run **after** the filter, once per match.
- Input: URL, Method, StatusCode, Headers, Body, ContentType, Length, Words, Lines.
- Output: per module a `map[string]any`; one module failing does not block the others.
- Data is stored in the engine report per `Result` in `ModuleData` and used by all output formats (TXT/CSV/HTML as summary, JSON/NDJSON/FFUF-JSON in full).
