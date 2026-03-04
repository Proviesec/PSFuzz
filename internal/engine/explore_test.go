package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveExploreWordlist(t *testing.T) {
	dir := t.TempDir()
	wp := filepath.Join(dir, "wordpress.txt")
	if err := os.WriteFile(wp, []byte("wp-admin\nwp-login\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := &ExploreAIResult{
		WordlistType:      "wordpress",
		SuggestedWordlist: "wordpress.txt",
	}

	path, ok := ResolveExploreWordlist(dir, result)
	if !ok {
		t.Fatal("expected wordlist to be found")
	}
	if path != wp {
		t.Errorf("got path %q, want %q", path, wp)
	}

	_, ok = ResolveExploreWordlist(dir, &ExploreAIResult{WordlistType: "nonexistent"})
	if ok {
		t.Error("expected no match for nonexistent.txt")
	}

	_, ok = ResolveExploreWordlist("", result)
	if ok {
		t.Error("expected false when dir is empty")
	}
	_, ok = ResolveExploreWordlist(dir, nil)
	if ok {
		t.Error("expected false when result is nil")
	}
}

func TestResolveExploreWordlistFromMap(t *testing.T) {
	m := map[string]string{
		"wordpress": "/custom/wordpress.txt",
		"typo3":     "https://example.com/typo3-paths.txt",
	}
	result := &ExploreAIResult{WordlistType: "wordpress", SuggestedWordlist: "wordpress.txt"}
	path, ok := ResolveExploreWordlistFromMap(m, result)
	if !ok || path != "/custom/wordpress.txt" {
		t.Errorf("expected wordpress map match, got ok=%v path=%q", ok, path)
	}
	result.WordlistType = "typo3"
	result.SuggestedWordlist = "typo3.txt"
	path, ok = ResolveExploreWordlistFromMap(m, result)
	if !ok || path != "https://example.com/typo3-paths.txt" {
		t.Errorf("expected typo3 URL, got ok=%v path=%q", ok, path)
	}
	_, ok = ResolveExploreWordlistFromMap(m, &ExploreAIResult{WordlistType: "unknown"})
	if ok {
		t.Error("expected no match for unknown")
	}
}

func TestResolveExploreWordlistDefault(t *testing.T) {
	for _, tt := range []struct {
		wordlistType string
		wantOK      bool
		wantURL     string
	}{
		{"wordpress", true, "wordpress.fuzz.txt"},
		{"generic", true, "common.txt"},
		{"typo3", true, "top-3.txt"},
		{"nginx", true, "nginx.txt"},
		{"unknown", false, ""},
		{"", false, ""},
	} {
		result := &ExploreAIResult{WordlistType: tt.wordlistType}
		path, ok := ResolveExploreWordlistDefault(result)
		if ok != tt.wantOK {
			t.Errorf("wordlist_type=%q: ok=%v, want %v", tt.wordlistType, ok, tt.wantOK)
		}
		if ok && tt.wantURL != "" && !strings.Contains(path, tt.wantURL) {
			t.Errorf("wordlist_type=%q: path=%q should contain %q", tt.wordlistType, path, tt.wantURL)
		}
	}
	_, ok := ResolveExploreWordlistDefault(nil)
	if ok {
		t.Error("expected false when result is nil")
	}
}

func TestMergeWordlistURLs(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("a\nb\nc\n"))
	}))
	defer s1.Close()
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("b\nc\nd\n")) // b,c duplicate
	}))
	defer s2.Close()

	ctx := context.Background()
	path, err := MergeWordlistURLs(ctx, []string{s1.URL, s2.URL})
	if err != nil {
		t.Fatalf("MergeWordlistURLs: %v", err)
	}
	defer os.Remove(path)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read merged file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// Order preserved, dedupe: a, b, c from s1, then d from s2 (b,c skipped)
	if len(lines) != 4 {
		t.Errorf("expected 4 unique lines, got %d: %v", len(lines), lines)
	}
	seen := make(map[string]bool)
	for _, l := range lines {
		if seen[l] {
			t.Errorf("duplicate in merged list: %q", l)
		}
		seen[l] = true
	}
	for _, want := range []string{"a", "b", "c", "d"} {
		if !seen[want] {
			t.Errorf("merged list missing %q", want)
		}
	}
}

// --- Explore AI cache tests (same package can use unexported cache helpers) ---

func TestNormalizeProbeURLForCache(t *testing.T) {
	tests := []struct {
		probeURL string
		want    string
	}{
		{"https://example.com/FUZZ", "https://example.com/"},
		{"https://example.com/", "https://example.com/"},
		{"https://example.com", "https://example.com/"},
		{"https://example.com/path/FUZZ", "https://example.com/path/"},
		{"https://example.com?a=1#x", "https://example.com/"},
	}
	for _, tt := range tests {
		got := normalizeProbeURLForCache(tt.probeURL)
		if got != tt.want {
			t.Errorf("normalizeProbeURLForCache(%q) = %q, want %q", tt.probeURL, got, tt.want)
		}
	}
}

func TestExploreAICacheKey(t *testing.T) {
	// Same logical target + same provider => same key
	k1 := exploreAICacheKey("https://example.com/FUZZ", "openai")
	k2 := exploreAICacheKey("https://example.com/", "openai")
	if k1 != k2 {
		t.Errorf("same target and provider should yield same key: %q vs %q", k1, k2)
	}
	// Different target => different key
	k3 := exploreAICacheKey("https://other.com/", "openai")
	if k1 == k3 {
		t.Error("different target should yield different key")
	}
	// Same URL, different provider => different key (cache is per provider)
	k4 := exploreAICacheKey("https://example.com/", "ollama")
	if k1 == k4 {
		t.Error("different provider should yield different key")
	}
	// Key is hex string (SHA256)
	if len(k1) != 64 {
		t.Errorf("key length want 64, got %d", len(k1))
	}
	for _, c := range k1 {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		t.Errorf("key should be hex, got %q", k1)
		break
	}
}

func TestExploreAICacheSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	result := &ExploreAIResult{
		WordlistType: "wordpress",
		WordlistURL:  "https://example.com/wp.txt",
		Extensions:   []string{"php", "html"},
	}
	key := exploreAICacheKey("https://target.example.com/", "openai")

	if err := saveExploreAICache(dir, key, result); err != nil {
		t.Fatalf("saveExploreAICache: %v", err)
	}
	loaded, ok := loadExploreAICache(dir, key, 1*time.Hour)
	if !ok {
		t.Fatal("loadExploreAICache: expected hit")
	}
	if loaded.WordlistType != result.WordlistType || loaded.WordlistURL != result.WordlistURL {
		t.Errorf("loaded result mismatch: got %+v", loaded)
	}
	if len(loaded.Extensions) != 2 || loaded.Extensions[0] != "php" || loaded.Extensions[1] != "html" {
		t.Errorf("loaded extensions: got %v", loaded.Extensions)
	}
}

func TestExploreAICacheExpired(t *testing.T) {
	dir := t.TempDir()
	key := exploreAICacheKey("https://expired.example.com/", "openai")
	path := filepath.Join(dir, "explore-"+key+".json")
	oldTime := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	entry := exploreAICacheEntry{
		Result:   &ExploreAIResult{WordlistType: "generic"},
		CachedAt: oldTime,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	loaded, ok := loadExploreAICache(dir, key, 1*time.Hour)
	if ok || loaded != nil {
		t.Errorf("expired entry should miss: ok=%v, loaded=%v", ok, loaded)
	}
}

func TestExploreAICacheLoadMissing(t *testing.T) {
	dir := t.TempDir()
	loaded, ok := loadExploreAICache(dir, "nonexistent-key-not-on-disk", 1*time.Hour)
	if ok || loaded != nil {
		t.Errorf("missing file should miss: ok=%v, loaded=%v", ok, loaded)
	}
}

func TestExploreAICacheLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	key := exploreAICacheKey("https://invalid.example.com/", "gemini")
	path := filepath.Join(dir, "explore-"+key+".json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, ok := loadExploreAICache(dir, key, 1*time.Hour)
	if ok || loaded != nil {
		t.Errorf("invalid JSON should miss: ok=%v, loaded=%v", ok, loaded)
	}
}

func TestBuildHeaderSummaryForAI(t *testing.T) {
	headers := map[string]string{
		"Content-Type": "text/html",
		"Cookie":       "session=secret123",
		"Authorization": "Bearer tok",
		"X-Custom":      "ok",
	}
	lines := buildHeaderSummaryForAI(headers)
	if len(lines) != 4 {
		t.Fatalf("expected 4 header lines, got %d", len(lines))
	}
	lineMap := make(map[string]string)
	for _, l := range lines {
		idx := strings.Index(l, ": ")
		if idx == -1 {
			continue
		}
		lineMap[l[:idx]] = l[idx+2:]
	}
	if lineMap["Cookie"] != "[redacted]" {
		t.Errorf("Cookie should be redacted, got %q", lineMap["Cookie"])
	}
	if lineMap["Authorization"] != "[redacted]" {
		t.Errorf("Authorization should be redacted, got %q", lineMap["Authorization"])
	}
	if lineMap["Content-Type"] != "text/html" {
		t.Errorf("Content-Type should be passed through, got %q", lineMap["Content-Type"])
	}
}

func TestBuildExplorePrompt(t *testing.T) {
	prompt := buildExplorePrompt("https://example.com/", 200, []string{"wordpress"}, []string{"Content-Type: text/html"}, "body snippet", "balanced")
	if prompt == "" {
		t.Fatal("prompt should be non-empty")
	}
	if !strings.Contains(prompt, "https://example.com/") {
		t.Error("prompt should contain target URL")
	}
	if !strings.Contains(prompt, "200") {
		t.Error("prompt should contain status code")
	}
	if !strings.Contains(prompt, "wordpress") {
		t.Error("prompt should contain technologies")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("prompt should ask for JSON")
	}
	// Profile affects strategy hint
	quick := buildExplorePrompt("https://x.com/", 200, nil, nil, "", "quick")
	if !strings.Contains(quick, "quick") {
		t.Error("quick profile should mention quick")
	}
}
