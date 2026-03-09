package modules

import (
	"context"
	"testing"
)

func TestLinksAnalyzer_ExtractsHrefActionSrc(t *testing.T) {
	in := Input{
		URL:        "https://example.com/dir/page.html",
		StatusCode: 200,
		Body:       `<html><a href="/about">About</a><form action="submit">x</form><img src="img/logo.png" /><a href="https://other.com/external">Ext</a></html>`,
	}
	out, err := LinksAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected non-nil data")
	}
	urls, ok := out.Data["urls"].([]string)
	if !ok || len(urls) < 4 {
		t.Errorf("expected at least 4 URLs, got %v", out.Data["urls"])
	}
	found := make(map[string]bool)
	for _, u := range urls {
		found[u] = true
	}
	// Resolved relative links
	if !found["https://example.com/about"] {
		t.Error("expected resolved /about")
	}
	if !found["https://example.com/dir/submit"] {
		t.Error("expected resolved action submit")
	}
	if !found["https://example.com/dir/img/logo.png"] {
		t.Error("expected resolved src img/logo.png")
	}
	if !found["https://other.com/external"] {
		t.Error("expected external link")
	}
}
