package modules

import (
	"context"
	"testing"
)

func TestURLExtractAnalyzer_ExtractsAndDedupes(t *testing.T) {
	in := Input{
		StatusCode: 200,
		Headers:    map[string]string{"Location": "https://example.com/redirect"},
		Body:       `See <a href="https://example.com/page">link</a> and https://other.com/path and https://example.com/page again.`,
	}
	out, err := URLExtractAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected non-nil data")
	}
	urls, ok := out.Data["urls"].([]string)
	if !ok || len(urls) < 2 {
		t.Errorf("expected at least 2 unique URLs, got %v", out.Data["urls"])
	}
	found := make(map[string]bool)
	for _, u := range urls {
		found[u] = true
	}
	if !found["https://example.com/redirect"] {
		t.Error("expected Location header URL in results")
	}
	if !found["https://example.com/page"] || !found["https://other.com/path"] {
		t.Error("expected body URLs in results")
	}
}
