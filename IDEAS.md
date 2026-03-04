# Ideas – what would make PSFuzz extra class

A living list of **north-star** and **nice-to-have** ideas. Not all will be built; this doc keeps the vision sharp and gives contributors direction. See [ROADMAP.md](ROADMAP.md) for committed plans.

---

## Reporting & triage

| Idea | Why it’s cool |
|------|----------------|
| **Run diff** | Compare two scan runs (e.g. before/after patch): highlight new, removed, or changed URLs. “What appeared after the deploy?” in one view. |
| **Discovery timeline** | Order results by discovery time; optional “narrative” export (we found /admin at 0:42, then /admin/users at 1:01). Great for reports and storytelling. |
| **Priority / risk score** | Per-URL score from status, length, modules (e.g. CORS permissive + 200 + tech stack). Sort by “most interesting first” for triage. |
| **Export to Burp** | Export found URLs as Burp project or sitemap import so testers continue in Burp without copy-paste. |
| **Summary stats block** | In JSON/report: `top_content_types`, `findings_by_depth`, `unique_hosts`, `status_distribution`. One glance to understand the app. |

---

## UX & CLI

| Idea | Why it’s cool |
|------|----------------|
| **Stdout output** | `-o -` writes the chosen format to stdout (e.g. `-of json -o - \| jq .results`). Piping into jq/grep without temp files. |
| **Presets** | `-preset quick` (low concurrency, small list), `-preset thorough` (high concurrency, extensions, modules), `-preset stealth` (jitter, random UA, rate limit). Good defaults for beginners, one flag for experts. |
| **Live progress** | Optional TUI or periodic line: “Hits: 12 | Queue: 340 | Rate: 45/s | Current: /api/FUZZ”. No need to tail a file. |
| **ETA** | Estimate “~2 min left” from queue size and current rate (like download managers). |
| **Why included/excluded** | In JSON/HTML: short reason per result: “included: -mc 200”, “excluded: -fr not found”. Makes filter behavior obvious. |
| **URL list from stdin** | `-list -` to read target URLs from stdin (e.g. `cat urls.txt \| psfuzz -list - -w wordlist.txt`). Fits Amass/httpx-style pipelines; see [FUZZING_GUIDES.md](FUZZING_GUIDES.md). |
| **Autocalibration** | Auto baseline from a few probe requests and suggest or apply filters (status/size). Like ffuf `-ac`; reduces manual filter tuning. |

---

---

## Security & modules

| Idea | Why it’s cool |
|------|----------------|
| **Parameter wordlist** | From links/forms, collect query and body param names; output a “parameter wordlist” for later fuzzing. Attack surface map. |
| **Open redirect / SSRF probe** | Optional: inject URL in common params, check response or out-of-band. Could be a small module or a dedicated mode. |
| **Reflexive / reflection filter** | Only match if the fuzzed value (FUZZ) is reflected in the response body. Useful for XSS/reflection checks; vaf has this. Could be a filter (e.g. `-reflective`) or a small module. |

---

## Integrations

| Idea | Why it’s cool |
|------|----------------|
| **Post-scan hook** | After run, call a script or binary with report path (e.g. `-post-hook ./notify.sh`). CI, Slack, custom pipeline. |
| **Webhook on finish** | POST report summary or URL list to a URL. Fits into dashboards and automation. |
| **Nuclei / template run** | Run Nuclei (or similar) on found URLs, or export a list for Nuclei. “Fuzz first, then template-scan.” |
| **Streaming NDJSON** | Option to stream one NDJSON result per line to stdout as they’re found. Real-time piping. |

---

## Data quality & scale

| Idea | Why it’s cool |
|------|----------------|
| **Similarity groups** | Expose “these N URLs have almost the same body” (build on near-duplicate logic). Reduces noise in reports. |
| **Content-type & size distribution** | In report: “50% HTML, 30% JSON, 20% other” and size histogram. Quick picture of the app. |
| **Resume from audit log** | Re-filter or re-analyze from an existing audit log without re-requesting. “Replay from log” mode. |
| **Distributed mode** | Split wordlist or URLs across multiple workers/machines. Enterprise / very large scans. |

---

## Polish & branding

| Idea | Why it’s cool |
|------|----------------|
| **Example configs** | `config.example.quick.json`, `config.example.stealth.json` next to `config.example.json`. Copy-paste starting points. |
| **Tested against** | Document “PSFuzz tested with OWASP WebGoat, DVWA, bWAPP” (or similar). Builds trust. |
| **Version in banner** | (Already have banner.) Keep it; optional `-v` / `--version` that only prints version for scripts. |
| **Mascot / visual** | Small, professional mascot or icon for README and releases. Memorable without being silly. |
| **Recommended wordlist sources** | Short section in README or CHEATSHEET: SecLists, FuzzDB, fuzz.txt, PayloadsAllTheThings, Assetnote, etc. Aligns with bug-bounty fuzzing guides; see [FUZZING_GUIDES.md](FUZZING_GUIDES.md). |

---

*New ideas welcome as PRs or issues.*
