package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ResolvedWordlist struct {
	Keyword string
	Words   []string
	Source  string
}

// ResolveWordlists resolves wordlist specs (paths, URLs, or built-in names) using cfg.
// ctx is used for HTTP requests when fetching remote wordlists; cancellation or timeout applies.
func ResolveWordlists(ctx context.Context, cfg *Config) ([]ResolvedWordlist, error) {
	specs := cfg.Wordlists
	if len(specs) == 0 {
		specs = []WordlistSpec{{Keyword: "FUZZ", Path: cfg.Wordlist}}
	}
	out := make([]ResolvedWordlist, 0, len(specs))
	for _, spec := range specs {
		words, source, err := resolveOne(ctx, cfg, spec.Path)
		if err != nil {
			return nil, fmt.Errorf("wordlist %q: %w", spec.Path, err)
		}
		words = applyWordlistCase(words, cfg.RandomizeWordlistCase)
		words = applyExtensions(words, cfg.Extensions)
		out = append(out, ResolvedWordlist{Keyword: spec.Keyword, Words: words, Source: source})
	}
	return out, nil
}

func resolveOne(ctx context.Context, cfg *Config, path string) ([]string, string, error) {
	switch path {
	case "default":
		return readRemote(ctx, DefaultPayloadURL, cfg.Timeout, cfg.IgnoreWordlistComments)
	case "fav":
		return readRemote(ctx, FavPayloadURL, cfg.Timeout, cfg.IgnoreWordlistComments)
	case "subdomain":
		return readRemote(ctx, SubdomainPayloadURL, cfg.Timeout, cfg.IgnoreWordlistComments)
	}

	if cfg.GeneratePayload {
		entries := make([]string, 0, cfg.GeneratePayloadLength)
		last := ""
		for i := 0; i < cfg.GeneratePayloadLength; i++ {
			last = nextAlias(last)
			entries = append(entries, last)
		}
		return entries, fmt.Sprintf("generated:%d", cfg.GeneratePayloadLength), nil
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return readRemote(ctx, path, cfg.Timeout, cfg.IgnoreWordlistComments)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	words, err := scanWords(f, cfg.IgnoreWordlistComments)
	return words, path, err
}

// readRemote fetches a wordlist from url with the given timeout and context. Cancellation or timeout aborts the request.
func readRemote(ctx context.Context, url string, timeout time.Duration, ignoreComments bool) ([]string, string, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("wordlist status %d", resp.StatusCode)
	}
	words, err := scanWords(resp.Body, ignoreComments)
	return words, url, err
}

func scanWords(r io.Reader, ignoreComments bool) ([]string, error) {
	s := bufio.NewScanner(r)
	words := make([]string, 0, 1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if ignoreComments && isCommentLine(line) {
			continue
		}
		if line != "" {
			words = append(words, line)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return words, nil
}

func applyExtensions(words []string, extensions []string) []string {
	if len(words) == 0 || len(extensions) == 0 {
		return words
	}
	out := make([]string, 0, len(words)*(len(extensions)+1))
	for _, w := range words {
		out = append(out, w)
		for _, ext := range extensions {
			out = append(out, w+ext)
		}
	}
	return out
}

func applyWordlistCase(words []string, mode string) []string {
	if len(words) == 0 || mode == "" {
		return words
	}
	out := make([]string, 0, len(words))
	switch mode {
	case "lower":
		for _, w := range words {
			out = append(out, strings.ToLower(w))
		}
	case "upper":
		for _, w := range words {
			out = append(out, strings.ToUpper(w))
		}
	default:
		return words
	}
	return out
}

func isCommentLine(line string) bool {
	if line == "" {
		return false
	}
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";")
}

func nextAlias(last string) string {
	if last == "" {
		return "a"
	}
	// Increment like base-26 with carry
	runes := []rune(last)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < 'z' {
			runes[i]++
			return string(runes)
		}
		runes[i] = 'a'
	}
	return "a" + string(runes)
}

// ParseWordlistSpecs parses a comma-separated list of wordlist specs. Each spec may be "path" or "path:keyword".
// Uses the last colon to split path and keyword so Windows paths like C:\wordlists\file.txt:FUZZ work (path=C:\wordlists\file.txt, keyword=FUZZ).
// If the part after the last colon contains \ or /, it is treated as part of the path (e.g. C:\file.txt).
func ParseWordlistSpecs(raw string) []WordlistSpec {
	parts := ParseCSV(raw)
	if len(parts) == 0 {
		return nil
	}
	out := make([]WordlistSpec, 0, len(parts))
	for _, p := range parts {
		key := "FUZZ"
		path := p
		if idx := strings.LastIndex(p, ":"); idx >= 0 && idx < len(p)-1 {
			suffix := p[idx+1:]
			// Only split if suffix does not look like a path (Windows or Unix)
			if !strings.Contains(suffix, "\\") && !strings.Contains(suffix, "/") {
				path = p[:idx]
				if k := strings.TrimSpace(suffix); k != "" {
					key = k
				}
			}
		} else if idx == len(p)-1 {
			path = p[:idx]
		}
		out = append(out, WordlistSpec{Keyword: key, Path: path})
	}
	return out
}
