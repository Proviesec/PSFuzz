package config

func Help() string {
	return `PSFuzz - Web Fuzzer

USAGE:
  psfuzz -u <URL> [options]

CORE:
  -u, -url <url>                Target URL
  -list, -url-list <file|csv>   URL list file or csv
  -d, -dirlist <path|url|name>  Wordlist (default|fav|subdomain|path|url)
  -w <path|url|name>            Wordlist (ffuf). Supports list:KEY
  -e, -ext <exts>               Extensions (ffuf, comma-separated)
  -ext-defaults                 Add default extensions
  -mode, -input-mode <m>        Input mode: clusterbomb|pitchfork|sniper
  -wordlist-case <m>            Wordlist case: lower|upper
  -enc <spec>                   Encoders (ffuf): KEYWORD:enc1,enc2 (e.g. FUZZ:urlencode)
  -c, -concurrency <n>          Concurrency (default 10)
  -D, -depth <n>                Max recursion depth (default 0)
  -r, -redirect                 Follow redirects (default true)
  -o, -output <base>            Output base name (default output)
  -of, -outputFormat <fmt>      txt|json|html|csv|ndjson|ffufjson

HTTP:
  -X, -method <verb>            HTTP method
  -data <body|@file>            HTTP data/body (POST if set); use @path to load body from file
  -H <h>                        Add header (ffuf)
  -x, -proxy <url>              Proxy
  -request <file>               Raw HTTP request file (ffuf)
  -replay-proxy <url>           Replay proxy (ffuf)
  -replay-on-match <bool>       Replay only matches (default true)
  -request-proto <scheme>       Override request scheme (ffuf)
  -proxy-user <user>            Proxy username
  -proxy-pass <pass>            Proxy password
  -verbs <csv>                  HTTP verbs
  -auto-verb                    Enable common verbs

STOP:
  -sf <codes>                   Stop on status codes
  -se                           Stop on errors
  -sa <n>                       Stop after matches

RESUME:
  -resume <file>                Resume file (requires -resume-every to save progress)
  -resume-every <n>             Write resume every N requests
  -save-config <file>           Save config to file

BYPASS:
  -bypass-budget <n>            Max bypass variants per URL
  -bypass-ratio <f>             Max ratio of bypass requests

WAF:
  -waf-adaptive                 Adaptive WAF slowdown
  -waf-threshold <n>            Trigger after N suspicious responses
  -waf-factor <f>               Slowdown factor

FILTERS:
  -fsc, -filterStatusCode <r>   Include status codes (e.g. 200,301-302)
  -fscn, -filterStatusCodeNot   Exclude status codes
  -mc <r>                       Match status codes (ffuf)
  -fc <r>                       Filter status codes (ffuf)
  -fl, -filterLength <r>        Include lengths
  -fln, -filterLengthNot <r>    Exclude lengths
  -ms <r>                       Match size (ffuf)
  -fs <r>                       Filter size (ffuf)
  -mw <r>                       Match words (ffuf)
  -fw <r>                       Filter words (ffuf)
  -ml <r>                       Match lines (ffuf)
  -fls <r>                      Filter lines (ffuf)
  -mt <r>                       Match time in ms (ffuf-like)
  -ft <r>                       Filter time in ms (ffuf-like)
  -nd                           Near-duplicate filter
  -nd-len <n>                   Near-duplicate length bucket
  -nd-words <n>                 Near-duplicate word bucket
  -nd-lines <n>                 Near-duplicate line bucket
  -bw <csv|file>                Block words (exclude if present)
  -is <csv|file>                Interesting strings (only keep matches)
  -t, -filterTestLength         Filter out baseline (random) length
  -fws, -filterWrongStatus200   Filter false 200 responses
  -fwd, -filterWrongSubdomain   Filter wrong subdomain responses
  -p404, -filterPossible404     Filter possible 404 titles
  -fm, -filterMatchWord <word>  Require word in body
  -mr, -filterMatchRegex <re>   Require regex match
  -mrn, -filterMatchRegexNot    Exclude regex match
  -fr <re>                      Filter regex (ffuf)
  -mrt, -filterMatchRegexTextOnly  Regex on visible text only
  -f, -filterContentType <ct>   Filter content-type (matches ct or ct; charset=...)
  -fd, -filterDuplicates        Dedupe by body fingerprint
  -dt, -duplicateThreshold <n>  Max duplicates per fingerprint
  -ic                           Ignore wordlist comments (ffuf)

AUTO-CALIBRATION:
  -ac                          Auto calibration (ffuf)
  -acn <n>                     Auto calibration requests

AUTH:
  -C, -cookie <k=v,...>         Add cookies
  -rah, -requestAddHeader <h>   Add headers (H:V,H2:V2)
  -raa, -requestAddAgent <ua>   User-Agent
  -random-ua                    Randomize User-Agent
  -bau, -basicAuthUser <u>      Basic Auth user
  -bap, -basicAuthPass <p>      Basic Auth pass

SECURITY:
  -safe <bool>                  Safe mode (default true)
  -insecure, -k                 Skip TLS certificate verification (e.g. self-signed, Burp MITM)
  -allow-hosts <h1,h2>          Restrict scan to hosts
  -exclude-paths <csv|file>     Exclude path segments
  -q, -quiet                    Quiet mode
  -dump                         Dump response bodies
  -dump-dir <dir>               Dump directory

LOGIN (session for all requests):
  -login-url <url>              Perform login once; Set-Cookie used for all fuzz requests
  -login-method <GET|POST>      HTTP method (default POST)
  -login-user <user>            Username (form field 'username')
  -login-pass <pass>            Password (form field 'password')
  -login-body <body>            Raw body (overrides user/pass; use for other field names or JSON)
  -login-content-type <ct>      Content-Type (default application/x-www-form-urlencoded)
  (Without login-user/login-pass and without login-body, a request with empty body is sent.)

PERF:
  -tr, -throttleRate <n>        Max requests per second
  -rate <n>                     Max requests per second (ffuf)
  -p <sec|min-max>              Delay between requests (ffuf)
  -timeout <sec>                Request timeout in seconds (default 30)
  -retry <n>                    Retry count on 5xx (and 429 when enabled)
  -retry-backoff-ms <ms>        Base backoff in ms
  -max-size <bytes>             Max response size (bytes)
  -min-size <bytes>             Min response size (bytes)
  -jitter                        Adaptive jitter profiling
  -jitter-threshold <ms>         Jitter threshold ms
  -jitter-factor <f>             Jitter factor

WILDCARD:
  -awc, -auto-wildcard          Per-host baseline wildcard filter

LEGACY FLAGS (still supported):
  -s, -showStatus            Show all status codes (cannot be combined with -mc/-fsc)
  -od, -onlydomains
  -cb, -checkBackslash
  -b, -bypass
  -btr, -bypassTooManyRequests
  -cf, -configfile

MODULES (response analysis):
  -modules <list>               Comma-separated modules (e.g. fingerprint,cors,headers,secrets,auth,ai,urlextract,links)
  -ai-prompt <text>             Custom AI prompt; placeholders: {{status}}, {{method}}, {{url}}, {{body}}
  -ai-provider <p>              AI module backend: openai (default) | ollama | gemini. Keys: OPENAI_API_KEY, or GEMINI_API_KEY/GOOGLE_API_KEY, or none for Ollama.
  -ai-endpoint <url>            AI module: override API URL (e.g. http://localhost:11434 for Ollama)
  -ai-model <name>              AI module: model name (default: gpt-4o-mini, llama3.1, gemini-1.5-flash per provider)
  -ai-max-tokens <n>            AI module: max tokens in response (0 = default 150). Lower = cheaper/faster.
  -enqueue-module-urls <list>   Enqueue URLs from these modules into scan queue (e.g. urlextract,links)
  -extracted-urls-file <path>   Write all extracted URLs to file, one per line (no enqueue)

EXPLORE AI (probe then recommend wordlist):
  -explore-ai                   Probe first URL (fingerprint, headers, response), send to AI for wordlist recommendation (JSON with wordlist_url, extensions, etc.). Sensitive headers redacted. Provider: openai | ollama | gemini.
  -explore-ai-provider <p>       With -explore-ai: openai (default) | ollama | gemini. OpenAI needs OPENAI_API_KEY; Gemini needs GEMINI_API_KEY or GOOGLE_API_KEY; Ollama usually no key.
  -explore-ai-endpoint <url>     With -explore-ai: override API URL (e.g. http://localhost:11434 for Ollama, or proxy for OpenAI).
  -explore-ai-model <name>       With -explore-ai: model name (default per provider: gpt-4o-mini, llama3.1, gemini-1.5-flash).
  -explore-ai-max-tokens <n>     With -explore-ai: max tokens in response (0 = default 500). Lower = cheaper.
  -explore-ai-profile <p>        With -explore-ai: quick (small/fast wordlist) | balanced | thorough (larger wordlist, more extensions). Default: balanced.
  -explore-ai-wordlists-dir <d>  With -explore-ai: directory containing wordlist files (e.g. wordpress.txt). Overrides built-in defaults when file exists.
  -explore-ai-wordlist <spec>     With -explore-ai: name:path_or_url (e.g. wordpress:./wp.txt,generic:https://...). Checked first; then wordlists-dir; then built-in defaults. See CHEATSHEET.
  -explore-ai-no-cache            With -explore-ai: do not use cache; always call API. Default: use cache (1h TTL, ~/.cache/psfuzz/explore-ai).

AUDIT:
  -audit-log <path>             Append request/response audit log (NDJSON)
  -audit-max-body <bytes>       Max body size per audit entry (0=unlimited)

TIMING & RECURSION:
  -maxtime <sec>                Max scan duration in seconds (0=disabled)
  -maxtime-job <sec>            Max duration per task in seconds (0=disabled)
  -recursion-strategy <s>       default (by status) or greedy (all matches)
  -rsc, -recursionStatusCodes   Status codes for recursion (default 200,301-302,403)

HTTP & VHOST:
  -http2                        Use HTTP/2
  -vhost                        Fuzz Host header (first wordlist value as Host)

PRESETS (apply a set of defaults; CLI flags override):
  -preset <name>                quick | stealth | thorough
    quick   – fast smoke: low concurrency, fav wordlist, 5 min maxtime, fingerprint only
    stealth – low profile: rate limit, delay, random UA, jitter
    thorough – full discovery: high concurrency, all modules, default extensions, depth 2, JSON output

CONFIG:
  -cf, -configfile <file>       Load JSON config

  -v, -version                  Print version and exit
`
}
