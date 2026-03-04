package modules

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

// urlRegex finds http/https URLs in text (simple, avoids complex backtracking).
var urlRegex = regexp.MustCompile(`https?://[^\s"'<>)\]]+`)

// URLExtractAnalyzer parses URLs from the response body and Location header,
// deduplicates and normalizes them, and returns them in module output.
// Output is stored per result in report.ModuleData["urlextract"]["urls"] ([]string).
type URLExtractAnalyzer struct{}

func init() {
	Register("urlextract", func(*Config) Analyzer { return URLExtractAnalyzer{} })
}

func (URLExtractAnalyzer) Name() string { return "urlextract" }

func (URLExtractAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}
	seen := make(map[string]struct{})
	var urls []string

	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		// Trim trailing punctuation that regex might have included
		raw = strings.TrimRight(raw, ".,;:!?)")
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return
		}
		norm := u.String()
		if _, ok := seen[norm]; ok {
			return
		}
		seen[norm] = struct{}{}
		urls = append(urls, norm)
	}

	// Location header (redirects)
	for k, v := range in.Headers {
		if strings.EqualFold(k, "Location") && v != "" {
			add(v)
			break
		}
	}

	// Body: regex find all
	body := in.Body
	if len(body) > 50000 {
		body = body[:50000]
	}
	for _, raw := range urlRegex.FindAllString(body, -1) {
		add(raw)
	}

	if len(urls) == 0 {
		return Output{Data: nil}, nil
	}
	return Output{Data: map[string]any{"urls": urls}}, nil
}
