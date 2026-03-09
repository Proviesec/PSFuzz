package engine

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Proviesec/PSFuzz/internal/config"
)

func extractTitle(body, lowerBody string) string {
	start := strings.Index(lowerBody, "<title>")
	if start == -1 {
		return ""
	}
	start += len("<title>")
	end := strings.Index(lowerBody[start:], "</title>")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(body[start : start+end])
}

func countWords(s string) int {
	n := 0
	wasSpace := true
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			wasSpace = true
		default:
			if wasSpace {
				n++
				wasSpace = false
			}
		}
	}
	return n
}

func countLines(body string) int {
	if body == "" {
		return 0
	}
	return strings.Count(body, "\n") + 1
}

func addExactRange(list *[]config.Range, v int) {
	for _, r := range *list {
		if r.Min == v && r.Max == v {
			return
		}
	}
	*list = append(*list, config.Range{Min: v, Max: v})
}

func requestKey(url, method, body string, headers map[string]string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, method)
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, url)
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, body)
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
	for _, k := range keys {
		_, _ = io.WriteString(h, "\n")
		_, _ = io.WriteString(h, strings.ToLower(k))
		_, _ = io.WriteString(h, ":")
		_, _ = io.WriteString(h, headers[k])
	}
	return url + "|sig:" + hex.EncodeToString(h.Sum(nil))
}

func computeConfidence(status int, title string, length int, words int, lines int, contentType string, truncated bool, suspect bool) float64 {
	score := 0.5
	switch {
	case status >= 200 && status < 300:
		score += 0.25
	case status >= 300 && status < 400:
		score += 0.1
	case status >= 400 && status < 500:
		score -= 0.2
	case status >= 500:
		score -= 0.4
	}
	lowerTitle := strings.ToLower(title)
	if strings.Contains(lowerTitle, "not found") || strings.Contains(lowerTitle, "error") || strings.Contains(lowerTitle, "forbidden") {
		score -= 0.2
	}
	if length == 0 || words == 0 || lines == 0 {
		score -= 0.1
	}
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/json") {
		score += 0.05
	}
	if truncated {
		score -= 0.05
	}
	if suspect {
		score -= 0.2
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func findInteresting(lowerBody string, needles []string) []string {
	if len(needles) == 0 || lowerBody == "" {
		return nil
	}
	out := make([]string, 0, len(needles))
	for _, n := range needles {
		n = strings.TrimSpace(strings.ToLower(n))
		if n == "" {
			continue
		}
		if strings.Contains(lowerBody, n) {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func readBodyWithLimit(r io.Reader, max int) ([]byte, bool, error) {
	if max <= 0 {
		data, err := io.ReadAll(r)
		return data, false, err
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(max)+1))
	if err != nil {
		return nil, false, err
	}
	if len(data) > max {
		return data[:max], true, nil
	}
	return data, false, nil
}

func dumpResponse(dir, url, method string, status int, contentType string, body []byte) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	sum := sha1.Sum([]byte(method + "\n" + url))
	base := hex.EncodeToString(sum[:8])
	bodyPath := filepath.Join(dir, base+".txt")
	metaPath := filepath.Join(dir, base+".meta.json")
	if err := os.WriteFile(bodyPath, body, 0644); err != nil {
		return err
	}
	meta := map[string]any{
		"url":         url,
		"method":      method,
		"status":      status,
		"contentType": contentType,
		"length":      len(body),
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, b, 0644)
}

func isWAFSuspect(headers http.Header, lowerBody string) bool {
	suspectBodies := []string{
		"access denied", "request blocked", "forbidden", "captcha", "attention required",
		"security check", "ddos protection", "temporarily blocked",
	}
	for _, s := range suspectBodies {
		if strings.Contains(lowerBody, s) {
			return true
		}
	}
	suspectHeaders := []string{
		"cf-ray", "cf-cache-status", "x-sucuri-id", "x-sucuri-block", "x-akamai-session-info",
		"x-distil-cs", "x-waf", "x-cdn", "x-cdn-request-id", "x-ae", "x-amz-cf-id",
	}
	for _, h := range suspectHeaders {
		if headers.Get(h) != "" {
			return true
		}
	}
	server := strings.ToLower(headers.Get("Server"))
	if strings.Contains(server, "cloudflare") || strings.Contains(server, "sucuri") || strings.Contains(server, "akamai") || strings.Contains(server, "fastly") {
		return true
	}
	return false
}

func isWrongStatus200(title string, length int) bool {
	errorTitles := []string{
		"Technical subdomain", "Page Not Available", "Access Gateway",
		"Not Found", "ERROR", "Error", "Forbidden", "Bad Request", "Internal Server Error", "Bad Gateway",
	}
	for _, t := range errorTitles {
		if strings.Contains(title, t) {
			return true
		}
	}
	return length <= 1
}

// isWrongSubdomain detects common CDN/DNS error pages that return 200 but indicate a dead or misconfigured subdomain.
// Body lengths: 21 = Cloudflare "error" stub, 180 = generic CDN "host not configured" page.
func isWrongSubdomain(title string, length int, statusCode int) bool {
	const (
		cloudflareErrorStubLen = 21
		cdnHostNotConfiguredLen = 180
	)
	return length == cloudflareErrorStubLen || length == cdnHostNotConfiguredLen || strings.Contains(title, "Origin DNS error") || statusCode == http.StatusPermanentRedirect
}

// headerMapFromHTTPHeader copies non-empty headers from h into a map[string]string.
func headerMapFromHTTPHeader(h http.Header) map[string]string {
	out := make(map[string]string)
	for k := range h {
		if v := h.Get(k); v != "" {
			out[k] = v
		}
	}
	return out
}
