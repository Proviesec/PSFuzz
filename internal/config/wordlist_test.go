package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWordlistSpecs(t *testing.T) {
	tests := []struct {
		raw  string
		want int
		key  string // first spec keyword
		path string // first spec path
	}{
		{"", 0, "", ""},
		{"list.txt", 1, "FUZZ", "list.txt"},
		{"list.txt:KEY", 1, "KEY", "list.txt"},
		{"list.txt:", 1, "FUZZ", "list.txt"},
		{"a.txt,b.txt:OTHER", 2, "FUZZ", "a.txt"},
		{"C:\\wordlist.txt:FUZZ", 1, "FUZZ", "C:\\wordlist.txt"}, // Windows path with keyword
	}
	for _, tt := range tests {
		got := ParseWordlistSpecs(tt.raw)
		if len(got) != tt.want {
			t.Errorf("ParseWordlistSpecs(%q) len=%d, want %d", tt.raw, len(got), tt.want)
		}
		if tt.want > 0 {
			if got[0].Keyword != tt.key {
				t.Errorf("ParseWordlistSpecs(%q) first Keyword=%q, want %q", tt.raw, got[0].Keyword, tt.key)
			}
			if got[0].Path != tt.path {
				t.Errorf("ParseWordlistSpecs(%q) first Path=%q, want %q", tt.raw, got[0].Path, tt.path)
			}
		}
	}
}

func TestResolveWordlists_LocalFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "words.txt")
	if err := os.WriteFile(f, []byte("a\nb\nc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Wordlist: f, IgnoreWordlistComments: false}
	out, err := ResolveWordlists(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ResolveWordlists: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 wordlist, got %d", len(out))
	}
	if len(out[0].Words) != 3 {
		t.Errorf("expected 3 words, got %d", len(out[0].Words))
	}
	if out[0].Source != f {
		t.Errorf("Source=%q, want %q", out[0].Source, f)
	}
}

func TestResolveWordlists_InvalidPath(t *testing.T) {
	cfg := &Config{Wordlist: "/nonexistent/path/xyz"}
	_, err := ResolveWordlists(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if len(err.Error()) == 0 {
		t.Error("error message should contain path context")
	}
}
