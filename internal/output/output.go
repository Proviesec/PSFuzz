package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
	"github.com/Proviesec/PSFuzz/internal/engine"
)

func Write(cfg *config.Config, report *engine.Report) error {
	if cfg == nil {
		return fmt.Errorf("config must not be nil")
	}
	if report == nil {
		return fmt.Errorf("report must not be nil")
	}
	txtPath := cfg.OutputBase + ".txt"
	if err := writeTXT(cfg, txtPath, report); err != nil {
		return fmt.Errorf("write txt: %w", err)
	}
	switch cfg.OutputFormat {
	case "txt":
		return nil
	case "json":
		if err := writeJSON(cfg.OutputBase+".json", report); err != nil {
			return fmt.Errorf("write json: %w", err)
		}
		return nil
	case "html":
		if err := writeHTML(cfg.OutputBase+".html", report); err != nil {
			return fmt.Errorf("write html: %w", err)
		}
		return nil
	case "csv":
		if err := writeCSV(cfg.OutputBase+".csv", report); err != nil {
			return fmt.Errorf("write csv: %w", err)
		}
		return nil
	case "ndjson":
		if err := writeNDJSON(cfg.OutputBase+".ndjson", report); err != nil {
			return fmt.Errorf("write ndjson: %w", err)
		}
		return nil
	case "compatjson":
		if err := writeCompatJSON(cfg.OutputBase+".json", report, cfg); err != nil {
			return fmt.Errorf("write compat json: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %s", cfg.OutputFormat)
	}
}

// displayURL returns the host when onlyDomains is true and parsing succeeds, otherwise the original URL.
func displayURL(rawURL string, onlyDomains bool) string {
	if !onlyDomains {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Host
}

func writeTXT(cfg *config.Config, path string, report *engine.Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "PSFuzz Scan Report\n")
	fmt.Fprintf(f, "Target: %s\n", report.TargetURL)
	fmt.Fprintf(f, "Wordlist: %s (%d entries)\n", report.WordlistSource, report.WordlistCount)
	if !report.StartedAt.IsZero() {
		fmt.Fprintf(f, "Started: %s\n", report.StartedAt.Format(time.RFC3339))
	}
	if !report.EndedAt.IsZero() {
		fmt.Fprintf(f, "Ended: %s\n", report.EndedAt.Format(time.RFC3339))
	}
	fmt.Fprintf(f, "Duration: %s\n", report.Duration)
	fmt.Fprintf(f, "Total Requests: %d\n\n", report.TotalRequests)

	for _, r := range report.Results {
		target := displayURL(r.URL, cfg.OnlyDomains)
		trunc := ""
		if r.Truncated {
			trunc = " trunc=true"
		}
		ct := r.ContentType
		if ct == "" {
			ct = "-"
		}
		redir := r.RedirectURL
		if redir == "" {
			redir = "-"
		}
		interesting := "-"
		if len(r.Interesting) > 0 {
			interesting = strings.Join(r.Interesting, "|")
		}
		modSum := formatModuleDataSummary(r.ModuleData)
		fmt.Fprintf(f, "%s | %s | depth=%d | len=%d | words=%d | time=%dms | ct=%s | redir=%s | conf=%.2f | interesting=%s | modules=%s%s\n", target, r.Status, r.Depth, r.Length, r.Words, r.TimeMS, ct, redir, r.Confidence, interesting, modSum, trunc)
	}

	fmt.Fprintf(f, "\nStatus Distribution:\n")
	keys := make([]string, 0, len(report.StatusCount))
	for k := range report.StatusCount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pct := 0.0
		if report.TotalRequests > 0 {
			pct = float64(report.StatusCount[k]) / float64(report.TotalRequests) * 100
		}
		fmt.Fprintf(f, "- %s: %d (%.1f%%)\n", k, report.StatusCount[k], pct)
	}
	if len(report.DiscoveredDirs) > 0 {
		fmt.Fprintf(f, "\nDiscovered Directories:\n")
		for _, d := range report.DiscoveredDirs {
			fmt.Fprintf(f, "- %s\n", d)
		}
	}

	fmt.Fprintf(f, "\n---\nModules used: %s\n", modulesUsed(report.Modules))
	if report.Commandline != "" {
		fmt.Fprintf(f, "Commandline: %s\n", report.Commandline)
	}
	return nil
}

func modulesUsed(m []string) string {
	if len(m) == 0 {
		return "none"
	}
	return strings.Join(m, ", ")
}

func writeJSON(path string, report *engine.Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func writeNDJSON(path string, report *engine.Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range report.Results {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV(path string, report *engine.Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"url", "status_code", "status", "content_type", "redirect_url", "length", "words", "time_ms", "depth", "timestamp", "truncated", "confidence", "interesting", "modules"})
	for _, r := range report.Results {
		_ = w.Write([]string{
			r.URL,
			fmt.Sprintf("%d", r.StatusCode),
			r.Status,
			r.ContentType,
			r.RedirectURL,
			fmt.Sprintf("%d", r.Length),
			fmt.Sprintf("%d", r.Words),
			fmt.Sprintf("%d", r.TimeMS),
			fmt.Sprintf("%d", r.Depth),
			r.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%t", r.Truncated),
			fmt.Sprintf("%.2f", r.Confidence),
			strings.Join(r.Interesting, "|"),
			formatModuleDataSummary(r.ModuleData),
		})
	}
	w.Flush()
	return w.Error()
}

func writeCompatJSON(path string, report *engine.Report, cfg *config.Config) error {
	type compatSummary struct {
		Commandline string `json:"commandline"`
		Time        string `json:"time"`
		StartedAt   string `json:"started_at,omitempty"`
		EndedAt     string `json:"ended_at,omitempty"`
		Results     int    `json:"results"`
		Duration    string `json:"duration"`
	}
	type compatConfig struct {
		Commandline string            `json:"commandline"`
		Time        string            `json:"time"`
		URL         string            `json:"url"`
		Wordlist    string            `json:"wordlist"`
		Method      string            `json:"method"`
		Headers     map[string]string `json:"headers,omitempty"`
		InputMode   string            `json:"input_mode"`
		Delay       string            `json:"delay"`
		Proxy       string            `json:"proxy"`
	}
	type compatResult struct {
		URL         string                    `json:"url"`
		Status      int                       `json:"status"`
		ContentType string                    `json:"content_type,omitempty"`
		RedirectURL string                    `json:"redirect_url,omitempty"`
		Length      int                       `json:"length"`
		Words       int                       `json:"words"`
		Lines       int                       `json:"lines"`
		TimeMS      int                       `json:"time_ms,omitempty"`
		Confidence  float64                   `json:"confidence,omitempty"`
		Interesting []string                  `json:"interesting,omitempty"`
		Input       map[string]string         `json:"input,omitempty"`
		Position    int                       `json:"position,omitempty"`
		ModuleData  map[string]map[string]any `json:"module_data,omitempty"`
	}
	type compatOutput struct {
		Commandline string         `json:"commandline"`
		Time        string         `json:"time"`
		Results     []compatResult `json:"results"`
		Config      compatConfig   `json:"config"`
		Duration    string         `json:"duration"`
		Summary     compatSummary  `json:"summary"`
	}
	results := make([]compatResult, 0, len(report.Results))
	for _, r := range report.Results {
		results = append(results, compatResult{
			URL:         r.URL,
			Status:      r.StatusCode,
			ContentType: r.ContentType,
			RedirectURL: r.RedirectURL,
			Length:      r.Length,
			Words:       r.Words,
			Lines:       r.Lines,
			TimeMS:      r.TimeMS,
			Confidence:  r.Confidence,
			Interesting: r.Interesting,
			Input:       r.Inputs,
			Position:    r.Position,
			ModuleData:  r.ModuleData,
		})
	}
	nowStr := time.Now().Format(time.RFC3339)
	out := compatOutput{
		Commandline: report.Commandline,
		Time:        nowStr,
		Results:     results,
		Config: compatConfig{
			Commandline: report.Commandline,
			Time:        nowStr,
			URL:         report.TargetURL,
			Wordlist:    report.WordlistSource,
			Method:      cfg.RequestMethod,
			Headers:     cfg.RequestHeaders,
			InputMode:   cfg.InputMode,
			Delay:       formatDelay(cfg),
			Proxy:       cfg.Proxy,
		},
		Duration: report.Duration.String(),
		Summary: compatSummary{
			Commandline: report.Commandline,
			Time:        nowStr,
			StartedAt:   formatTimeOrEmpty(report.StartedAt),
			EndedAt:     formatTimeOrEmpty(report.EndedAt),
			Results:     len(report.Results),
			Duration:    report.Duration.String(),
		},
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func formatDelay(cfg *config.Config) string {
	if cfg.DelayMin == 0 && cfg.DelayMax == 0 {
		return ""
	}
	if cfg.DelayMin == cfg.DelayMax {
		return cfg.DelayMin.String()
	}
	return fmt.Sprintf("%s-%s", cfg.DelayMin, cfg.DelayMax)
}

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func writeHTML(path string, report *engine.Report) error {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>PSFuzz Report</title>")
	b.WriteString("<style>body{font-family:ui-sans-serif;padding:16px;background:#f4f7fb}table{width:100%;border-collapse:collapse;background:#fff}th,td{padding:8px;border:1px solid #ddd}th{background:#102a43;color:#fff}.ok{color:#0a7f3f}.redir{color:#c17d00}.client{color:#b42318}.server{color:#6941c6}</style></head><body>")
	if !report.StartedAt.IsZero() && !report.EndedAt.IsZero() {
		fmt.Fprintf(&b, "<h1>PSFuzz Report</h1><p><strong>Target:</strong> %s<br><strong>Started:</strong> %s<br><strong>Ended:</strong> %s<br><strong>Duration:</strong> %s<br><strong>Total requests:</strong> %d</p>", report.TargetURL, report.StartedAt.Format(time.RFC3339), report.EndedAt.Format(time.RFC3339), report.Duration, report.TotalRequests)
	} else {
		fmt.Fprintf(&b, "<h1>PSFuzz Report</h1><p><strong>Target:</strong> %s<br><strong>Duration:</strong> %s<br><strong>Total requests:</strong> %d</p>", report.TargetURL, report.Duration, report.TotalRequests)
	}
	b.WriteString("<h2>Results</h2><table><tr><th>URL</th><th>Status</th><th>Depth</th><th>Length</th><th>Words</th><th>Time(ms)</th><th>Content-Type</th><th>Redirect</th><th>Conf</th><th>Interesting</th><th>Modules</th><th>Trunc</th></tr>")
	for _, r := range report.Results {
		cls := statusClass(r.StatusCode)
		modCell := htmlEscape(formatModuleDataSummary(r.ModuleData))
		fmt.Fprintf(&b, "<tr><td>%s</td><td class=%q>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%s</td><td>%s</td><td>%.2f</td><td>%s</td><td>%s</td><td>%t</td></tr>", htmlEscape(r.URL), cls, htmlEscape(r.Status), r.Depth, r.Length, r.Words, r.TimeMS, htmlEscape(r.ContentType), htmlEscape(r.RedirectURL), r.Confidence, htmlEscape(strings.Join(r.Interesting, "|")), modCell, r.Truncated)
	}
	b.WriteString("</table>")
	fmt.Fprintf(&b, "<p><strong>Modules used:</strong> %s</p>", htmlEscape(modulesUsed(report.Modules)))
	if report.Commandline != "" {
		fmt.Fprintf(&b, "<p><strong>Commandline:</strong> <code>%s</code></p>", htmlEscape(report.Commandline))
	}
	b.WriteString("</body></html>")
	return os.WriteFile(path, []byte(b.String()), 0644) //nolint:gosec
}

// statusClass returns a CSS class name for the status code (ok, redir, client, server).
func statusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "ok"
	case code >= 300 && code < 400:
		return "redir"
	case code >= 400 && code < 500:
		return "client"
	default:
		return "server"
	}
}

var htmlReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")

func htmlEscape(s string) string {
	return htmlReplacer.Replace(s)
}

// formatModuleDataSummary returns a short one-line summary of module data for display in tables.
func formatModuleDataSummary(m map[string]map[string]any) string {
	if len(m) == 0 {
		return "-"
	}
	var parts []string
	for name, data := range m {
		if len(data) == 0 {
			parts = append(parts, name+": -")
			continue
		}
		var vals []string
		for k, v := range data {
			switch x := v.(type) {
			case string:
				vals = append(vals, k+"="+x)
			case []string:
				vals = append(vals, k+"="+strings.Join(x, ","))
			case []any:
				var ss []string
				for _, item := range x {
					if s, ok := item.(string); ok {
						ss = append(ss, s)
					}
				}
				vals = append(vals, k+"="+strings.Join(ss, ","))
			case bool:
				if x {
					vals = append(vals, k)
				}
			default:
				vals = append(vals, fmt.Sprintf("%v", v))
			}
		}
		sort.Strings(vals)
		parts = append(parts, name+": "+strings.Join(vals, "; "))
	}
	sort.Strings(parts)
	return strings.Join(parts, " | ")
}
