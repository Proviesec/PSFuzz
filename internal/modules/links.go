package modules

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

// Regexes for HTML link attributes (capture group 1 = value; unquoted and single/double quoted).
var (
	hrefRe   = regexp.MustCompile(`(?i)\bhref\s*=\s*["']?([^"'\s>]+)`)
	actionRe = regexp.MustCompile(`(?i)\baction\s*=\s*["']?([^"'\s>]+)`)
	srcRe    = regexp.MustCompile(`(?i)\bsrc\s*=\s*["']?([^"'\s>]+)`)
)

// LinksAnalyzer extracts links from HTML (href, action, src), resolves them against the request URL,
// deduplicates and returns them in module output. Output is stored in report.ModuleData["links"]["urls"] ([]string).
// Use with -enqueue-module-urls links to enqueue discovered URLs into the scan queue.
type LinksAnalyzer struct{}

func init() {
	Register("links", func(*Config) Analyzer { return LinksAnalyzer{} })
}

func (LinksAnalyzer) Name() string { return "links" }

func (LinksAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}
	base, err := url.Parse(in.URL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return Output{Data: nil}, nil
	}
	body := in.Body
	if len(body) > 500000 {
		body = body[:500000]
	}
	seen := make(map[string]struct{})
	var urls []string

	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(strings.TrimLeft(raw, " \t"), "javascript:") {
			return
		}
		u, err := base.Parse(raw)
		if err != nil {
			return
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return
		}
		norm := u.String()
		if _, ok := seen[norm]; ok {
			return
		}
		seen[norm] = struct{}{}
		urls = append(urls, norm)
	}

	for _, m := range hrefRe.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range actionRe.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range srcRe.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}

	if len(urls) == 0 {
		return Output{Data: nil}, nil
	}
	return Output{Data: map[string]any{"urls": urls}}, nil
}
