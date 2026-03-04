package engine

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
	"github.com/Proviesec/PSFuzz/internal/httpx"
	"github.com/Proviesec/PSFuzz/internal/llm"
	"github.com/Proviesec/PSFuzz/internal/modules"
)

const (
	exploreAIMaxBody  = 4000
	exploreAITimeout  = 20 * time.Second
	exploreAICacheTTL = 1 * time.Hour
	exploreAIDefaultMaxTokens = 500
)

// Headers we never send to the AI (sensitive).
var exploreAIRedactHeaders = map[string]bool{
	"cookie": true, "authorization": true, "proxy-authorization": true,
	"x-api-key": true, "x-auth-token": true, "api-key": true,
}

// buildHeaderSummaryForAI returns header lines for the prompt; sensitive headers are redacted.
func buildHeaderSummaryForAI(headerMap map[string]string) []string {
	out := make([]string, 0, len(headerMap))
	for k, v := range headerMap {
		keyLower := strings.ToLower(k)
		if exploreAIRedactHeaders[keyLower] || strings.Contains(keyLower, "token") || strings.Contains(keyLower, "secret") {
			v = "[redacted]"
		} else if len(v) > 60 {
			v = v[:60] + "..."
		}
		out = append(out, k+": "+v)
	}
	return out
}

// ExploreAIResult is the structured recommendation returned by the AI backend for explore-ai mode.
// The AI receives fingerprint, headers, and response; it returns JSON with wordlist_url or wordlist_urls so PSFuzz can use them as the payload source. Multiple URLs are merged and deduplicated.
type ExploreAIResult struct {
	WordlistType      string   `json:"wordlist_type"`      // e.g. wordpress, typo3, laravel, generic
	SuggestedWordlist string   `json:"suggested_wordlist"` // e.g. wordpress.txt
	WordlistURL       string   `json:"wordlist_url"`       // single URL; used when wordlist_urls is empty
	WordlistURLs      []string `json:"wordlist_urls"`      // multiple URLs (e.g. nginx + wordpress + fav); tool fetches each, merges and deduplicates, then uses as one payload
	Extensions        []string `json:"extensions"`         // e.g. ["php","html"]
	StatusCodes       []int    `json:"status_codes"`       // e.g. [200,301,302]
	Reasoning         string   `json:"reasoning"`
	SuggestedCommand  string   `json:"suggested_command"`  // optional one-liner
	FocusAreas        string   `json:"focus_areas"`       // optional: e.g. "wp-admin, plugin versions, readme.html"
	NextSteps         string   `json:"next_steps"`        // optional: e.g. "consider recursion on /wp-content"
}

// exploreAICacheEntry is stored on disk for Explore AI cache.
type exploreAICacheEntry struct {
	Result   *ExploreAIResult `json:"result"`
	CachedAt string           `json:"cached_at"` // RFC3339
}

func exploreAICacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "psfuzz", "explore-ai")
	return dir, os.MkdirAll(dir, 0750)
}

// normalizeProbeURLForCache returns a stable string for the same target (scheme + host + path with FUZZ removed).
func normalizeProbeURLForCache(probeURL string) string {
	u, err := url.Parse(probeURL)
	if err != nil {
		return probeURL
	}
	path := strings.TrimSuffix(u.Path, "/")
	path = strings.ReplaceAll(path, "FUZZ", "")
	if path == "" {
		path = "/"
	}
	u.Path = path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func exploreAICacheKey(probeURL, provider string) string {
	norm := normalizeProbeURLForCache(probeURL)
	p := strings.TrimSpace(strings.ToLower(provider))
	if p == "" {
		p = "openai"
	}
	h := sha256.Sum256([]byte(norm + "|" + p))
	return hex.EncodeToString(h[:])
}

func loadExploreAICache(cacheDir, key string, ttl time.Duration) (*ExploreAIResult, bool) {
	path := filepath.Join(cacheDir, "explore-"+key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry exploreAICacheEntry
	if err := json.Unmarshal(data, &entry); err != nil || entry.Result == nil {
		return nil, false
	}
	t, err := time.Parse(time.RFC3339, entry.CachedAt)
	if err != nil || time.Since(t) > ttl {
		return nil, false
	}
	return entry.Result, true
}

func saveExploreAICache(cacheDir, key string, result *ExploreAIResult) error {
	path := filepath.Join(cacheDir, "explore-"+key+".json")
	entry := exploreAICacheEntry{Result: result, CachedAt: time.Now().UTC().Format(time.RFC3339)}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// RunExploreAI probes the base URL, runs fingerprint and headers modules, calls the configured AI backend (openai/ollama/gemini) for a wordlist/payload recommendation, prints the result, and returns the recommendation so the caller can optionally run a scan with the selected wordlist.
func RunExploreAI(ctx context.Context, cfg *config.Config) (*ExploreAIResult, error) {
	probeURL := cfg.URL
	if probeURL == "" && len(cfg.URLs) > 0 {
		probeURL = cfg.URLs[0]
	}
	if probeURL == "" {
		return nil, fmt.Errorf("explore-ai requires -u or -list with at least one URL")
	}

	if !cfg.ExploreAINoCache {
		cacheDir, err := exploreAICacheDir()
		if err == nil {
			ck := exploreAICacheKey(probeURL, cfg.ExploreAIProvider)
			if cached, ok := loadExploreAICache(cacheDir, ck, exploreAICacheTTL); ok {
				printExploreResult(probeURL, cached)
				fmt.Println("(from cache)")
				return cached, nil
			}
		}
	}

	apiKey, err := llm.GetAPIKey(llm.NormalizeProviderFromString(cfg.ExploreAIProvider))
	if err != nil {
		return nil, err
	}

	client, err := httpx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("http client: %w", err)
	}
	defer client.Close()

	if cfg.LoginURL != "" {
		if err := doLogin(ctx, client, cfg); err != nil {
			return nil, fmt.Errorf("login before explore: %w", err)
		}
	}

	spec := httpx.RequestSpec{URL: probeURL, Method: "GET", Headers: cfg.RequestHeaders}
	resp, err := client.Do(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("probe request: %w", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	headerMap := headerMapFromHTTPHeader(resp.Header)
	bodyStr := string(respBody)
	if len(bodyStr) > exploreAIMaxBody {
		bodyStr = bodyStr[:exploreAIMaxBody] + "...[truncated]"
	}
	words := len(strings.Fields(bodyStr))
	lines := strings.Count(bodyStr, "\n") + 1
	contentType := resp.Header.Get("Content-Type")

	in := modules.Input{
		URL:         probeURL,
		Method:      "GET",
		StatusCode:  resp.StatusCode,
		Headers:     headerMap,
		Body:        bodyStr,
		ContentType: contentType,
		Length:      len(respBody),
		Words:       words,
		Lines:       lines,
	}
	analyzers := []modules.Analyzer{modules.FingerprintAnalyzer{}, modules.HeadersAnalyzer{}}
	moduleOut := modules.Run(ctx, analyzers, in)

	techList := []string{}
	if fp, ok := moduleOut["fingerprint"]; ok && fp != nil {
		if t, ok := fp["technologies"].([]string); ok {
			techList = t
		}
	}
	headerSummary := buildHeaderSummaryForAI(headerMap)

	profile := strings.ToLower(strings.TrimSpace(cfg.ExploreAIProfile))
	if profile != "quick" && profile != "balanced" && profile != "thorough" {
		profile = "balanced"
	}
	prompt := buildExplorePrompt(probeURL, resp.StatusCode, techList, headerSummary, bodyStr, profile)
	result, err := callExploreAI(ctx, cfg, apiKey, prompt, nil)
	if err != nil {
		return nil, err
	}

	printExploreResult(probeURL, result)
	if !cfg.ExploreAINoCache {
		if cacheDir, err := exploreAICacheDir(); err == nil {
			_ = saveExploreAICache(cacheDir, exploreAICacheKey(probeURL, cfg.ExploreAIProvider), result)
		}
	}
	return result, nil
}

// Default Explore AI wordlist URLs (SecLists). Used when the user does not provide -explore-ai-wordlist or -explore-ai-wordlists-dir.
var defaultExploreWordlistURLs = map[string]string{
	"wordpress":   "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/CMS/wordpress.fuzz.txt",
	"drupal":      "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/CMS/drupal.txt",
	"joomla":      "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/CMS/joomla-themes.fuzz.txt",
	"typo3":       "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/CMS/top-3.txt",
	"nginx":       "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/CMS/nginx.txt",
	"laravel":     "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/common.txt",
	"generic":     "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/common.txt",
	"directories": "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/directory-list-2.3-small.txt",
	"raft":        "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/raft-small-directories.txt",
}

// ResolveExploreWordlistDefault returns the default wordlist URL for the AI-recommended wordlist_type. Used when the user has not set a map or dir.
func ResolveExploreWordlistDefault(result *ExploreAIResult) (pathOrURL string, ok bool) {
	if result == nil {
		return "", false
	}
	key := strings.ToLower(strings.TrimSpace(result.WordlistType))
	if key == "" {
		return "", false
	}
	if u, ok := defaultExploreWordlistURLs[key]; ok && u != "" {
		return u, true
	}
	return "", false
}

// ResolveExploreWordlistFromMap returns the path or URL from the user map for the recommended wordlist type or name. Keys are matched case-insensitively (wordlist_type and basename of suggested_wordlist).
func ResolveExploreWordlistFromMap(m map[string]string, result *ExploreAIResult) (pathOrURL string, ok bool) {
	if len(m) == 0 || result == nil {
		return "", false
	}
	keys := []string{}
	if result.WordlistType != "" {
		keys = append(keys, strings.ToLower(strings.TrimSpace(result.WordlistType)))
	}
	if result.SuggestedWordlist != "" {
		base := result.SuggestedWordlist
		if idx := strings.LastIndex(base, "."); idx > 0 {
			base = base[:idx]
		}
		base = strings.TrimSpace(strings.ToLower(base))
		if base != "" {
			keys = append(keys, base)
		}
	}
	for _, k := range keys {
		if v, ok := m[k]; ok && v != "" {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

// ResolveExploreWordlist returns the path to a wordlist file in dir that matches the Explore AI recommendation (suggested_wordlist or wordlist_type.txt). Returns ("", false) if no file exists.
func ResolveExploreWordlist(dir string, result *ExploreAIResult) (wordlistPath string, ok bool) {
	if dir == "" || result == nil {
		return "", false
	}
	candidates := []string{}
	if result.SuggestedWordlist != "" {
		candidates = append(candidates, result.SuggestedWordlist)
	}
	if result.WordlistType != "" {
		candidates = append(candidates, result.WordlistType+".txt")
	}
	dirClean := filepath.Clean(dir)
	for _, name := range candidates {
		name = filepath.Clean(name)
		if name == "" || name == "." || strings.HasPrefix(name, "..") {
			continue
		}
		p := filepath.Join(dirClean, name)
		rel, err := filepath.Rel(dirClean, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// fetchWordlistFromURL fetches a wordlist from an http(s) URL and returns lines (trimmed, non-empty, no comment filtering).
func fetchWordlistFromURL(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wordlist %s status %d", url, resp.StatusCode)
	}
	var words []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, ";") {
			words = append(words, line)
		}
	}
	return words, sc.Err()
}

// MergeWordlistURLs fetches each URL, merges all words, removes duplicates (order preserved), writes to a temp file and returns its path.
func MergeWordlistURLs(ctx context.Context, urls []string) (tmpPath string, err error) {
	seen := make(map[string]struct{})
	var merged []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || (!strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://")) {
			continue
		}
		words, err := fetchWordlistFromURL(ctx, u)
		if err != nil {
			return "", fmt.Errorf("fetch %s: %w", u, err)
		}
		for _, w := range words {
			if _, ok := seen[w]; !ok {
				seen[w] = struct{}{}
				merged = append(merged, w)
			}
		}
	}
	if len(merged) == 0 {
		return "", fmt.Errorf("no words after merging %d URL(s)", len(urls))
	}
	f, err := os.CreateTemp("", "psfuzz-merged-*.txt")
	if err != nil {
		return "", err
	}
	for _, w := range merged {
		if _, err := fmt.Fprintln(f, w); err != nil {
			f.Close()
			os.Remove(f.Name())
			return "", err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// SecLists base URL (raw) for AI reference. Discovery/Web-Content/CMS has many lists: wordpress.fuzz.txt, drupal.txt, joomla-themes.fuzz.txt, joomla-plugins.fuzz.txt, top-3.txt, nginx.txt, coldfusion.txt, django.txt, sharepoint.txt, etc.
const exploreAISecListsRawBase = "https://raw.githubusercontent.com/danielmiessler/SecLists/master"

func buildExplorePrompt(url string, status int, technologies []string, headerLines []string, bodySnippet string, profile string) string {
	techStr := "none detected"
	if len(technologies) > 0 {
		techStr = strings.Join(technologies, ", ")
	}
	headerStr := strings.Join(headerLines, "\n")
	if len(headerStr) > 800 {
		headerStr = headerStr[:800] + "..."
	}
	if len(bodySnippet) > 1500 {
		bodySnippet = bodySnippet[:1500] + "...[truncated]"
	}
	favURL := config.FavPayloadURL
	defaultURL := config.DefaultPayloadURL
	var strategyHint string
	switch profile {
	case "quick":
		strategyHint = "quick: prefer small/fast wordlists (e.g. fav, common.txt, or small CMS list); fewer extensions."
	case "thorough":
		strategyHint = "thorough: prefer larger wordlists (e.g. raft-large-directories, full CMS list); more extensions; consider wordlist_urls combining multiple lists."
	default:
		strategyHint = "balanced: medium wordlist and extensions."
	}
	return fmt.Sprintf(`You are a security fuzzing expert. A single HTTP response from a target URL was probed (fingerprint, headers, response). Suggest the best fuzzing setup and return a JSON object. Strategy: %s

Target URL: %s
HTTP status: %d
Detected technologies: %s

Response headers (sample):
%s

Response body (sample):
%s

Wordlist sources (use FULL raw URLs):
- SecLists raw base: %s
  - CMS: %s/Discovery/Web-Content/CMS/ (wordpress.fuzz.txt, drupal.txt, joomla-themes.fuzz.txt, top-3.txt, nginx.txt, coldfusion.txt, django.txt, sharepoint.txt, etc.)
  - Web-Content: %s/Discovery/Web-Content/common.txt, directory-list-2.3-small.txt, raft-small-directories.txt, raft-large-directories.txt, quickhits.txt
- Fav (short): %s
- Default (large): %s

Return either wordlist_url (single) or wordlist_urls (array to merge and deduplicate). Optionally add focus_areas (what to check first, e.g. "wp-admin, plugin versions") and next_steps (e.g. "recursion on /wp-content").

Reply with ONLY a valid JSON object, no markdown. Structure:
{"wordlist_type":"wordpress|typo3|drupal|joomla|nginx|generic","suggested_wordlist":"wordpress.txt","wordlist_url":"https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/CMS/wordpress.fuzz.txt","wordlist_urls":[],"extensions":["php","html"],"status_codes":[200,301,302],"reasoning":"one short sentence","suggested_command":"psfuzz -u URL -w wordlist.txt -e php,html -mc 200,301,302","focus_areas":"","next_steps":""}

If combining technologies use wordlist_urls. Keep reasoning, focus_areas and next_steps brief.`, strategyHint, url, status, techStr, headerStr, bodySnippet, exploreAISecListsRawBase, exploreAISecListsRawBase, exploreAISecListsRawBase, favURL, defaultURL)
}

// exploreLLMConfig builds llm.Config from main config for Explore AI.
func exploreLLMConfig(cfg *config.Config) llm.Config {
	maxTokens := cfg.ExploreAIMaxTokens
	if maxTokens <= 0 {
		maxTokens = exploreAIDefaultMaxTokens
	}
	return llm.Config{
		Provider:  llm.NormalizeProviderFromString(cfg.ExploreAIProvider),
		Endpoint:  cfg.ExploreAIEndpoint,
		Model:     cfg.ExploreAIModel,
		MaxTokens: maxTokens,
		Timeout:   exploreAITimeout,
	}
}

// exploreAIRetry holds the assistant reply and the follow-up user message for a JSON-fix retry.
type exploreAIRetry struct {
	AssistantContent string
	UserFollowUp     string
}

// callExploreAI calls the LLM layer and returns the parsed result. On JSON parse failure, retries once with a fix instruction.
func callExploreAI(ctx context.Context, cfg *config.Config, apiKey, prompt string, retry *exploreAIRetry) (*ExploreAIResult, error) {
	var messages []llm.Message
	if retry != nil {
		messages = []llm.Message{
			{Role: "user", Content: prompt},
			{Role: "assistant", Content: retry.AssistantContent},
			{Role: "user", Content: retry.UserFollowUp},
		}
	} else {
		messages = []llm.Message{{Role: "user", Content: prompt}}
	}
	content, err := llm.Call(ctx, exploreLLMConfig(cfg), apiKey, messages)
	if err != nil {
		return nil, err
	}
	content = normalizeExploreContent(content)
	var result ExploreAIResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		if retry != nil {
			return nil, fmt.Errorf("parse API JSON: %w", err)
		}
		repaired, retryErr := callExploreAI(ctx, cfg, apiKey, prompt, &exploreAIRetry{
			AssistantContent: content,
			UserFollowUp:     "Your previous response was not valid JSON. Reply with ONLY the JSON object, no other text or markdown.",
		})
		if retryErr != nil {
			return nil, fmt.Errorf("parse API JSON: %w", err)
		}
		return repaired, nil
	}
	return &result, nil
}

func normalizeExploreContent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func printExploreResult(probeURL string, r *ExploreAIResult) {
	fmt.Println("--- Explore AI recommendation ---")
	fmt.Printf("Wordlist type: %s\n", r.WordlistType)
	if len(r.WordlistURLs) > 0 {
		fmt.Printf("Wordlist URLs (merged, deduplicated): %s\n", strings.Join(r.WordlistURLs, ", "))
	}
	if r.WordlistURL != "" {
		fmt.Printf("Wordlist URL (payload source): %s\n", r.WordlistURL)
	}
	if r.SuggestedWordlist != "" {
		fmt.Printf("Suggested wordlist: %s\n", r.SuggestedWordlist)
	}
	if len(r.Extensions) > 0 {
		fmt.Printf("Extensions: %s\n", strings.Join(r.Extensions, ","))
	}
	if len(r.StatusCodes) > 0 {
		codes := make([]string, len(r.StatusCodes))
		for i, c := range r.StatusCodes {
			codes[i] = fmt.Sprintf("%d", c)
		}
		fmt.Printf("Match status codes: %s\n", strings.Join(codes, ","))
	}
	if r.Reasoning != "" {
		fmt.Printf("Reasoning: %s\n", r.Reasoning)
	}
	if r.FocusAreas != "" {
		fmt.Printf("Focus areas: %s\n", r.FocusAreas)
	}
	if r.NextSteps != "" {
		fmt.Printf("Next steps: %s\n", r.NextSteps)
	}
	if r.SuggestedCommand != "" {
		cmd := strings.ReplaceAll(r.SuggestedCommand, "URL", probeURL)
		fmt.Printf("Suggested command: %s\n", cmd)
	}
	fmt.Println("---------------------------------")
}

// StatusRangesFromCodes converts a list of status codes to config.StatusRange slice (each code as Min=Max).
// Used when applying Explore AI result status codes to the scan filter.
func StatusRangesFromCodes(codes []int) []config.StatusRange {
	if len(codes) == 0 {
		return nil
	}
	r := make([]config.StatusRange, 0, len(codes))
	for _, c := range codes {
		r = append(r, config.StatusRange{Min: c, Max: c})
	}
	return r
}

// ResolveExploreWordlistPath resolves the wordlist path or URL from an Explore AI result, trying merged URLs,
// single URL, user map, wordlists dir, and default in order. When cfg.Quiet is false, logs merge failures to stdout.
func ResolveExploreWordlistPath(ctx context.Context, cfg *config.Config, result *ExploreAIResult) (pathOrURL string, ok bool) {
	if len(result.WordlistURLs) > 0 {
		merged, err := MergeWordlistURLs(ctx, result.WordlistURLs)
		if err == nil {
			return merged, true
		}
		if cfg != nil && !cfg.Quiet {
			fmt.Fprintf(os.Stdout, "Merge wordlist URLs failed: %v — trying other sources.\n", err)
		}
	}
	if u := strings.TrimSpace(result.WordlistURL); u != "" && (strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
		return u, true
	}
	if cfg != nil {
		if path, ok := ResolveExploreWordlistFromMap(cfg.ExploreAIWordlistMap, result); ok {
			return path, true
		}
		if cfg.ExploreAIWordlistsDir != "" {
			if path, ok := ResolveExploreWordlist(cfg.ExploreAIWordlistsDir, result); ok {
				return path, true
			}
		}
	}
	return ResolveExploreWordlistDefault(result)
}
