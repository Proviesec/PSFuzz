package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
)

func TestApplyTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		values   map[string]string
		want     string
	}{
		{"FUZZ replaced", "https://example.com/FUZZ", map[string]string{"FUZZ": "admin"}, "https://example.com/admin"},
		{"#PSFUZZ# replaced", "https://x/#PSFUZZ#", map[string]string{"FUZZ": "p"}, "https://x/p"},
		{"multiple keywords", "https://x/USER/PASS", map[string]string{"USER": "u", "PASS": "p"}, "https://x/u/p"},
		{"empty template", "", map[string]string{"FUZZ": "a"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyTemplate(tt.template, tt.values)
			if got != tt.want {
				t.Errorf("applyTemplate(%q, %v) = %q, want %q", tt.template, tt.values, got, tt.want)
			}
		})
	}
}

func TestApplyHeaderTemplate(t *testing.T) {
	got := applyHeaderTemplate(
		map[string]string{"X-Custom": "prefix-FUZZ-suffix", "Static": "no-placeholder"},
		map[string]string{"FUZZ": "value"},
	)
	if got["X-Custom"] != "prefix-value-suffix" {
		t.Errorf("X-Custom: got %q", got["X-Custom"])
	}
	if got["Static"] != "no-placeholder" {
		t.Errorf("Static: got %q", got["Static"])
	}
}

func TestURLTemplateCoversAllKeywords(t *testing.T) {
	lists := []config.ResolvedWordlist{
		{Keyword: "FUZZ", Words: []string{"a"}},
	}
	if !urlTemplateCoversAllKeywords("https://x/FUZZ", lists) {
		t.Error("expected true for URL with FUZZ")
	}
	if urlTemplateCoversAllKeywords("https://x/", lists) {
		t.Error("expected false for URL without FUZZ")
	}
	lists2 := []config.ResolvedWordlist{
		{Keyword: "USER", Words: []string{"u"}},
		{Keyword: "PASS", Words: []string{"p"}},
	}
	if !urlTemplateCoversAllKeywords("https://x/USER/PASS", lists2) {
		t.Error("expected true for USER and PASS")
	}
	if urlTemplateCoversAllKeywords("https://x/USER", lists2) {
		t.Error("expected false when PASS missing")
	}
}

func TestDepthIsTaskLocalAndBounded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin", "/admin/", "/admin/admin/":
			fmt.Fprint(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	dir := t.TempDir()
	wordlist := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		URL:             ts.URL,
		Wordlist:        wordlist,
		Concurrency:     8,
		Depth:           2,
		RecursionSmart:  true,
		RecursionStatus: []config.StatusRange{{Min: 200, Max: 200}},
		FollowRedirects: true,
		OutputBase:      "output",
		OutputFormat:    "txt",
		Timeout:         5 * time.Second,
		SafeMode:        false,
		RetryCount:      0,
		RetryBackoff:    1 * time.Millisecond,
		RequestHeaders:  map[string]string{},
		RequestCookies:  map[string]string{},
	}

	report, err := func() (*Report, error) {
		e, err := New(cfg)
		if err != nil {
			return nil, err
		}
		return e.Run(context.Background())
	}()
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(report.Results) == 0 {
		t.Fatal("expected results")
	}
	for _, r := range report.Results {
		if r.Depth < 0 || r.Depth > 2 {
			t.Fatalf("invalid depth %d for %s", r.Depth, r.URL)
		}
	}
	foundDepth2 := false
	for _, r := range report.Results {
		if strings.Contains(r.URL, "/admin/admin/") && r.Depth == 1 {
			foundDepth2 = true
		}
	}
	if !foundDepth2 {
		t.Fatal("expected recursive hit at depth 1")
	}
}

func TestSafeModeBlocksLoopback(t *testing.T) {
	dir := t.TempDir()
	wordlist := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		URL:             "http://127.0.0.1",
		Wordlist:        wordlist,
		Concurrency:     1,
		Depth:           0,
		RecursionStatus: []config.StatusRange{{Min: 200, Max: 200}},
		FollowRedirects: true,
		OutputBase:      "output",
		OutputFormat:    "txt",
		Timeout:         1 * time.Second,
		SafeMode:        true,
		RetryCount:      0,
		RetryBackoff:    1 * time.Millisecond,
		RequestHeaders:  map[string]string{},
		RequestCookies:  map[string]string{},
	}

	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	report, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if report.StatusCount["Error"] == 0 {
		t.Fatal("expected safe mode error count")
	}
}
