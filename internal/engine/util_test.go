package engine

import (
	"net/http"
	"strings"
	"testing"
)

func TestCountWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one\ttwo\nthree", 3},
		{"  \t\n  ", 0},
		{"a", 1},
		{"the quick brown fox jumps over the lazy dog", 9},
		{"line1\nline2\nline3", 3},
		{"\r\n\r\n", 0},
		{"word1\r\nword2\r\nword3", 3},
		{"<html><body>hello world</body></html>", 2},
	}
	for _, tt := range tests {
		got := countWords(tt.input)
		if got != tt.want {
			t.Errorf("countWords(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestCountWords_MatchesFieldsLen(t *testing.T) {
	samples := []string{
		"",
		"word",
		"hello world",
		"  spaced   out  ",
		"tabs\there",
		"new\nlines\nhere",
		"mixed \t\n whitespace",
		"<div class=\"test\">content here</div>",
	}
	for _, s := range samples {
		got := countWords(s)
		want := len(strings.Fields(s))
		if got != want {
			t.Errorf("countWords(%q) = %d, want %d (strings.Fields)", s, got, want)
		}
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 1},
		{"line1\nline2", 2},
		{"line1\nline2\nline3", 3},
		{"\n", 2},
		{"\n\n\n", 4},
		{"no newline at end", 1},
	}
	for _, tt := range tests {
		got := countLines(tt.input)
		if got != tt.want {
			t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"simple title", "<html><head><title>Hello World</title></head></html>", "Hello World"},
		{"mixed case tags", "<html><head><TITLE>Mixed Case</TITLE></head></html>", "Mixed Case"},
		{"no title", "<html><head></head></html>", ""},
		{"empty title", "<html><title></title></html>", ""},
		{"title with whitespace", "<title>  Spaced  </title>", "Spaced"},
		{"unclosed title", "<title>No end tag", ""},
		{"no head", "just plain text", ""},
		{"preserves original case", "<title>MyApp Dashboard</title>", "MyApp Dashboard"},
		{"title with entities", "<title>A &amp; B</title>", "A &amp; B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lower := strings.ToLower(tt.body)
			got := extractTitle(tt.body, lower)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindInteresting(t *testing.T) {
	tests := []struct {
		name      string
		lowerBody string
		needles   []string
		want      int
	}{
		{"no needles", "hello world", nil, 0},
		{"empty body", "", []string{"test"}, 0},
		{"match one", "this page has admin access", []string{"admin"}, 1},
		{"match multiple", "admin panel with debug info", []string{"admin", "debug", "missing"}, 2},
		{"case insensitive needles", "admin panel", []string{"ADMIN", "Panel"}, 2},
		{"no match", "nothing here", []string{"admin", "secret"}, 0},
		{"empty needle skipped", "hello", []string{"", "  ", "hello"}, 1},
		{"whitespace in needle trimmed", "hello", []string{"  hello  "}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findInteresting(tt.lowerBody, tt.needles)
			if len(got) != tt.want {
				t.Errorf("findInteresting() returned %d matches, want %d (got: %v)", len(got), tt.want, got)
			}
		})
	}
}

func TestIsWAFSuspect(t *testing.T) {
	tests := []struct {
		name      string
		headers   http.Header
		lowerBody string
		want      bool
	}{
		{
			"clean response",
			http.Header{"Content-Type": {"text/html"}},
			"hello world",
			false,
		},
		{
			"body contains access denied",
			http.Header{},
			"access denied - your request has been blocked",
			true,
		},
		{
			"body contains captcha",
			http.Header{},
			"please solve the captcha to continue",
			true,
		},
		{
			"cloudflare cf-ray header",
			http.Header{"Cf-Ray": {"abc123"}},
			"normal content",
			true,
		},
		{
			"x-waf header",
			http.Header{"X-Waf": {"blocked"}},
			"normal content",
			true,
		},
		{
			"cloudflare server",
			http.Header{"Server": {"cloudflare"}},
			"normal content",
			true,
		},
		{
			"akamai server",
			http.Header{"Server": {"Akamai Ghost"}},
			"normal content",
			true,
		},
		{
			"ddos protection in body",
			http.Header{},
			"ddos protection by service",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWAFSuspect(tt.headers, tt.lowerBody)
			if got != tt.want {
				t.Errorf("isWAFSuspect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeConfidence(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		title       string
		length      int
		words       int
		lines       int
		contentType string
		truncated   bool
		suspect     bool
		wantMin     float64
		wantMax     float64
	}{
		{
			"200 OK HTML",
			200, "Dashboard", 500, 50, 10, "text/html", false, false,
			0.7, 0.85,
		},
		{
			"404 not found title",
			404, "Not Found", 100, 10, 3, "text/html", false, false,
			0.0, 0.2,
		},
		{
			"200 suspect WAF with error title",
			200, "Forbidden", 200, 20, 5, "text/html", false, true,
			0.3, 0.45,
		},
		{
			"500 server error",
			500, "Internal Server Error", 50, 5, 1, "text/html", false, false,
			0.0, 0.1,
		},
		{
			"301 redirect",
			301, "", 0, 0, 0, "", false, false,
			0.0, 0.55,
		},
		{
			"200 empty body",
			200, "", 0, 0, 0, "text/html", false, false,
			0.55, 0.75,
		},
		{
			"clamps to 0",
			500, "Error", 0, 0, 0, "", true, true,
			0.0, 0.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeConfidence(tt.status, tt.title, tt.length, tt.words, tt.lines, tt.contentType, tt.truncated, tt.suspect)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("computeConfidence() = %.2f, want [%.2f, %.2f]", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestIsWrongStatus200(t *testing.T) {
	tests := []struct {
		title  string
		length int
		want   bool
	}{
		{"Normal Page", 5000, false},
		{"Not Found", 200, true},
		{"Error", 100, true},
		{"ERROR", 100, true},
		{"Page Not Available", 300, true},
		{"Forbidden", 150, true},
		{"Internal Server Error", 500, true},
		{"Bad Gateway", 100, true},
		{"", 0, true},
		{"", 1, true},
		{"", 2, false},
		{"Some Title", 50, false},
	}
	for _, tt := range tests {
		got := isWrongStatus200(tt.title, tt.length)
		if got != tt.want {
			t.Errorf("isWrongStatus200(%q, %d) = %v, want %v", tt.title, tt.length, got, tt.want)
		}
	}
}

func TestMatchesBaseline(t *testing.T) {
	valid := baselineFingerprint{Status: 200, Length: 1000, Words: 100, Lines: 20, Valid: true}
	invalid := baselineFingerprint{Status: 200, Length: 1000, Words: 100, Lines: 20, Valid: false}

	tests := []struct {
		name    string
		b       baselineFingerprint
		status  int
		length  int
		words   int
		lines   int
		want    bool
	}{
		{"exact match", valid, 200, 1000, 100, 20, true},
		{"invalid baseline never matches", invalid, 200, 1000, 100, 20, false},
		{"different status", valid, 404, 1000, 100, 20, false},
		{"different length", valid, 200, 999, 100, 20, false},
		{"different words", valid, 200, 1000, 99, 20, false},
		{"different lines", valid, 200, 1000, 100, 19, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesBaseline(tt.b, tt.status, tt.length, tt.words, tt.lines)
			if got != tt.want {
				t.Errorf("matchesBaseline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadBodyWithLimit(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		max       int
		wantLen   int
		wantTrunc bool
	}{
		{"no limit", "hello world", 0, 11, false},
		{"under limit", "hello", 100, 5, false},
		{"at limit", "hello", 5, 5, false},
		{"over limit", "hello world", 5, 5, true},
		{"empty body", "", 100, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.body)
			data, trunc, err := readBodyWithLimit(r, tt.max)
			if err != nil {
				t.Fatalf("readBodyWithLimit() error = %v", err)
			}
			if len(data) != tt.wantLen {
				t.Errorf("len(data) = %d, want %d", len(data), tt.wantLen)
			}
			if trunc != tt.wantTrunc {
				t.Errorf("truncated = %v, want %v", trunc, tt.wantTrunc)
			}
		})
	}
}

func TestIsWrongSubdomain(t *testing.T) {
	tests := []struct {
		title      string
		length     int
		statusCode int
		want       bool
	}{
		{"Normal Page", 5000, 200, false},
		{"Origin DNS error", 100, 200, true},
		{"", 21, 200, true},
		{"", 180, 200, true},
		{"", 500, 308, true},
		{"", 500, 200, false},
	}
	for _, tt := range tests {
		got := isWrongSubdomain(tt.title, tt.length, tt.statusCode)
		if got != tt.want {
			t.Errorf("isWrongSubdomain(%q, %d, %d) = %v, want %v", tt.title, tt.length, tt.statusCode, got, tt.want)
		}
	}
}

func TestRequestKey_Deterministic(t *testing.T) {
	headers := map[string]string{"X-Custom": "val", "Auth": "token"}
	k1 := requestKey("https://x.com/a", "GET", "", headers)
	k2 := requestKey("https://x.com/a", "GET", "", headers)
	if k1 != k2 {
		t.Error("requestKey should be deterministic")
	}
}

func TestRequestKey_DifferentInputs(t *testing.T) {
	h := map[string]string{}
	k1 := requestKey("https://x.com/a", "GET", "", h)
	k2 := requestKey("https://x.com/b", "GET", "", h)
	k3 := requestKey("https://x.com/a", "POST", "", h)
	k4 := requestKey("https://x.com/a", "GET", "data", h)

	if k1 == k2 {
		t.Error("different URLs should produce different keys")
	}
	if k1 == k3 {
		t.Error("different methods should produce different keys")
	}
	if k1 == k4 {
		t.Error("different bodies should produce different keys")
	}
}
