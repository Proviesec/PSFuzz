package modules

import (
	"context"
	"testing"
)

func TestRun_EmptyAnalyzers(t *testing.T) {
	out := Run(context.Background(), nil, Input{})
	if out != nil {
		t.Errorf("expected nil for nil analyzers, got %v", out)
	}
	out = Run(context.Background(), []Analyzer{}, Input{})
	if out != nil {
		t.Errorf("expected nil for empty analyzers, got %v", out)
	}
}

func TestRun_ContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := Run(ctx, []Analyzer{FingerprintAnalyzer{}}, Input{StatusCode: 200, Body: "x"})
	// may be nil or partial (fingerprint might not run if ctx is checked first)
	if len(out) > 1 {
		t.Errorf("expected no or single result when context done, got %d", len(out))
	}
}

func TestRun_FingerprintAndCORS(t *testing.T) {
	in := Input{
		StatusCode:  200,
		Headers:     map[string]string{"Server": "nginx/1.18", "X-Powered-By": "PHP/7.4"},
		Body:        "<html>wp-content theme</html>",
		ContentType: "text/html",
	}
	out := Run(context.Background(), []Analyzer{FingerprintAnalyzer{}, CORSAnalyzer{}}, in)

	if out == nil {
		t.Fatal("expected non-nil output")
	}
	if fp, ok := out["fingerprint"]; !ok || fp == nil {
		t.Error("expected fingerprint result")
	} else if tech, ok := fp["technologies"].([]string); !ok || len(tech) < 2 {
		t.Errorf("expected at least nginx and php from headers/body, got %v", fp)
	}
}

func TestEnabled_NilConfig(t *testing.T) {
	out := Enabled(nil)
	if out != nil {
		t.Errorf("expected nil for nil config, got %d analyzers", len(out))
	}
}

func TestEnabled_EmptyModules(t *testing.T) {
	out := Enabled((*Config)(nil))
	if out != nil {
		t.Errorf("expected nil for nil config, got %d analyzers", len(out))
	}
	out = Enabled(&Config{Modules: nil})
	if out != nil {
		t.Errorf("expected nil for nil modules, got %d", len(out))
	}
	out = Enabled(&Config{Modules: []string{}})
	if out != nil {
		t.Errorf("expected nil for empty modules, got %d", len(out))
	}
}

func TestEnabled_Deduplication(t *testing.T) {
	mc := &Config{
		Modules:  []string{"fingerprint", "cors", "fingerprint", "ai", "cors"},
		AIPrompt: "",
	}
	out := Enabled(mc)
	if len(out) != 3 {
		t.Errorf("expected 3 unique analyzers (fingerprint,cors,ai), got %d", len(out))
	}
	names := make(map[string]int)
	for _, a := range out {
		names[a.Name()]++
	}
	for n, c := range names {
		if c != 1 {
			t.Errorf("analyzer %s should appear once, got %d", n, c)
		}
	}
}

func TestEnabled_UnknownSkipped(t *testing.T) {
	mc := &Config{Modules: []string{"unknown", "fingerprint"}}
	out := Enabled(mc)
	if len(out) != 1 {
		t.Errorf("expected 1 analyzer (fingerprint), got %d", len(out))
	}
	if len(out) > 0 && out[0].Name() != "fingerprint" {
		t.Errorf("expected fingerprint, got %s", out[0].Name())
	}
}

func TestEnabled_Links(t *testing.T) {
	mc := &Config{Modules: []string{"links", "fingerprint"}}
	out := Enabled(mc)
	if len(out) != 2 {
		t.Errorf("expected 2 analyzers, got %d", len(out))
	}
	names := make(map[string]bool)
	for _, a := range out {
		names[a.Name()] = true
	}
	if !names["links"] || !names["fingerprint"] {
		t.Errorf("expected links and fingerprint, got %v", names)
	}
}

func TestEnabled_Headers(t *testing.T) {
	mc := &Config{Modules: []string{"headers", "fingerprint"}}
	out := Enabled(mc)
	if len(out) != 2 {
		t.Errorf("expected 2 analyzers, got %d", len(out))
	}
	names := make(map[string]bool)
	for _, a := range out {
		names[a.Name()] = true
	}
	if !names["headers"] || !names["fingerprint"] {
		t.Errorf("expected headers and fingerprint, got %v", names)
	}
}
