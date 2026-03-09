package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Proviesec/PSFuzz/internal/config"
	"github.com/Proviesec/PSFuzz/internal/engine"
)

func TestWriteJSONWithModuleData(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "scan")
	cfg := &config.Config{OutputBase: base, OutputFormat: "json"}
	report := &engine.Report{
		TargetURL:     "https://example.com",
		TotalRequests: 1,
		StatusCount:   map[string]int{"200": 1},
		Results: []engine.Result{
			{
				URL: "https://example.com/admin", StatusCode: 200, Status: "200",
				ModuleData: map[string]map[string]any{
					"fingerprint": {"technologies": []any{"nginx", "php"}},
					"cors":        {"access_control_allow_origin": "*"},
				},
			},
		},
	}
	if err := Write(cfg, report); err != nil {
		t.Fatal(err)
	}
	path := base + ".json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Results []struct {
			ModuleData map[string]map[string]any `json:"module_data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(decoded.Results))
	}
	if decoded.Results[0].ModuleData["fingerprint"]["technologies"] == nil {
		t.Error("expected fingerprint.technologies in module_data")
	}
	if decoded.Results[0].ModuleData["cors"]["access_control_allow_origin"] != "*" {
		t.Error("expected cors.access_control_allow_origin in module_data")
	}
}

func TestWriteTXTZeroRequests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	report := &engine.Report{TargetURL: "https://x", StatusCount: map[string]int{"Error": 0}, TotalRequests: 0}
	cfg := &config.Config{}
	if err := writeTXT(cfg, path, report); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestWriteNDJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.ndjson")
	report := &engine.Report{
		Results: []engine.Result{
			{URL: "https://x", StatusCode: 200, Status: "200 OK"},
		},
	}
	if err := writeNDJSON(path, report); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestWriteHTML(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "scan")
	cfg := &config.Config{OutputBase: base, OutputFormat: "html"}
	report := &engine.Report{
		TargetURL:     "https://example.com",
		TotalRequests: 1,
		Results: []engine.Result{
			{URL: "https://example.com/admin", StatusCode: 200, Status: "200"},
		},
	}
	if err := Write(cfg, report); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(base + ".html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "PSFuzz Report") || !strings.Contains(content, "<table>") {
		t.Errorf("HTML should contain title and table, got: %s...", content[:min(200, len(content))])
	}
}

func TestWriteCSV(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "scan")
	cfg := &config.Config{OutputBase: base, OutputFormat: "csv"}
	report := &engine.Report{
		TargetURL:     "https://example.com",
		TotalRequests: 1,
		Results: []engine.Result{
			{URL: "https://example.com/foo", StatusCode: 200, Status: "200", ContentType: "text/html"},
		},
	}
	if err := Write(cfg, report); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(base + ".csv")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "url,status_code,status,content_type") {
		t.Errorf("CSV should have header with url,status_code,..., got: %s", content[:min(150, len(content))])
	}
	if !strings.Contains(content, "https://example.com/foo") {
		t.Error("CSV should contain result URL")
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "scan")
	cfg := &config.Config{OutputBase: base, OutputFormat: "compatjson", RequestMethod: "GET"}
	report := &engine.Report{
		TargetURL:     "https://example.com",
		TotalRequests: 1,
		Results: []engine.Result{
			{URL: "https://example.com/admin", StatusCode: 200, Status: "200"},
		},
	}
	if err := Write(cfg, report); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Results []struct {
			URL    string `json:"url"`
			Status int    `json:"status"`
		} `json:"results"`
		Config struct {
			Method string `json:"method"`
		} `json:"config"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid compat json: %v", err)
	}
	if len(decoded.Results) != 1 || decoded.Results[0].URL != "https://example.com/admin" || decoded.Results[0].Status != 200 {
		t.Errorf("unexpected results: %+v", decoded.Results)
	}
	if decoded.Config.Method != "GET" {
		t.Errorf("config.method: got %q", decoded.Config.Method)
	}
}

func TestWriteCSV_SpecialCharacters(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "scan")
	cfg := &config.Config{OutputBase: base, OutputFormat: "csv"}
	report := &engine.Report{
		TargetURL:     "https://example.com",
		TotalRequests: 2,
		Results: []engine.Result{
			{
				URL:         `https://example.com/path?a=1&b=2`,
				StatusCode:  200,
				Status:      "200 OK",
				ContentType: `text/html; charset="utf-8"`,
				RedirectURL: "",
			},
			{
				URL:         "https://example.com/with,comma",
				StatusCode:  301,
				Status:      "301 Moved",
				ContentType: "text/html",
				RedirectURL: "https://example.com/new,location",
				Interesting: []string{"admin|panel"},
			},
		},
	}
	if err := Write(cfg, report); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(base + ".csv")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 CSV lines (header + 2 rows), got %d", len(lines))
	}
	if !strings.Contains(content, `"https://example.com/with,comma"`) {
		t.Error("URL with comma should be quoted in CSV")
	}
	if !strings.Contains(content, `"text/html; charset=""utf-8"""`) {
		t.Error("content-type with quotes should be properly escaped in CSV")
	}
}

func TestWriteUnsupportedFormat(t *testing.T) {
	cfg := &config.Config{OutputFormat: "invalid"}
	report := &engine.Report{}
	err := Write(cfg, report)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention unsupported: %v", err)
	}
}
