# Roadmap

Planned items for future releases. For a broader list of **ideas** (reporting, UX, integrations, polish), see **[IDEAS.md](IDEAS.md)**—north-star and “extra class” features that could be picked up over time.

## Response modules (planned)

| Module    | Description |
|-----------|-------------|
| **headers** | *Implemented.* Evaluate security-related response headers (CSP, X-Frame-Options, HSTS, X-Content-Type-Options, Set-Cookie Secure/HttpOnly). Flags missing or weak values. Use `-modules headers`. |
| **secrets** | *Implemented.* Scan body and response headers for common secret/key patterns (AWS keys, JWTs, GitHub/Slack tokens, password in response, api_key/secret). Reports finding types only. Use `-modules secrets`. |
| **auth**    | *Implemented.* Detect auth-related responses: login form, logout link, 401, 302 to login, “session expired”, Set-Cookie with session/auth name. Use `-modules auth`. |

*Implemented: **links** (extract `href`/`action`/`src`, output `urls`; use with `-enqueue-module-urls links` for link-driven discovery).*

Implementation: same pattern as existing modules (see [MODULES.md](MODULES.md#adding-a-new-module-for-developers)).

## Features (planned)

| Feature | Description |
|---------|-------------|
| **Automatic login / session** | *Implemented.* `-login-url`, `-login-user`, `-login-pass` (or `-login-body` for custom form/JSON). One login request; Set-Cookie applied to all fuzz requests. |
| **TLS skip verify (`-insecure`)** | *Implemented.* `-insecure` or `-k` disables TLS certificate verification (labs, Burp MITM, self-signed). |
| **File-based body payload (`@file`)** | *Implemented.* `-d @path` loads the request body from a file; keywords (e.g. FUZZ) in the file are replaced per request. See [CHEATSHEET.md](CHEATSHEET.md#body-from-file). |
| **Payload encoders** | *Implemented.* `-enc KEYWORD:enc1,enc2` (e.g. `FUZZ:urlencode`, `FUZZ:urlencode,base64encode`). Encoders: urlencode, doubleurlencode, base64encode, base64decode, htmlencode, htmldoubleencode. See [CHEATSHEET.md](CHEATSHEET.md#encoders). |
| **Multipart file upload fuzzing** | Fuzz file upload endpoints: multipart/form-data with file from wordlist (e.g. paths or extensions). Enables testing upload restrictions, extension bypass, LFI via filename. See [FUZZING_GUIDES.md](FUZZING_GUIDES.md). |

*Implemented: **Audit log** (`-audit-log`, NDJSON). **Spider / link-driven discovery** via `-enqueue-module-urls urlextract,links` (URLs from those modules are queued for scanning). **Explore AI** (`-explore-ai`): probe base URL (fingerprint+headers), OpenAI recommends wordlist/extensions for the detected stack (e.g. WordPress, TYPO3); requires `OPENAI_API_KEY`. See [CHEATSHEET.md](CHEATSHEET.md#explore-ai).*

*Inspired by ffuf/feroxbuster/gobuster GitHub issues, Reddit, and security tool discussions.*
