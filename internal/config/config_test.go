package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadUsesConfigFilePathAndCLIOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.json")
	if err := os.WriteFile(cfgPath, []byte(`{"url":"https://from-file.example","dirlist":"/tmp/filelist","concurrency":3}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load([]string{"-cf", cfgPath, "-u", "https://from-cli.example", "-c", "9"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.URL != "https://from-cli.example" {
		t.Fatalf("expected cli URL override, got %s", cfg.URL)
	}
	if cfg.Wordlist != "/tmp/filelist" {
		t.Fatalf("expected dirlist from file, got %s", cfg.Wordlist)
	}
	if cfg.Concurrency != 9 {
		t.Fatalf("expected concurrency 9, got %d", cfg.Concurrency)
	}
}

func TestParseRanges(t *testing.T) {
	ranges, err := ParseRanges("200-204,301,404")
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}
	if !MatchAnyRange(ranges, 203) || !MatchAnyRange(ranges, 301) || MatchAnyRange(ranges, 500) {
		t.Fatal("range matching failed")
	}
}

func TestLegacyFlagsAreAccepted(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-s",
		"-od",
		"-cb",
		"-b",
		"-btr",
		"-f", "text/html",
		"-t",
		"-fws",
		"-fwd",
		"-p404",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.ShowStatus || !cfg.OnlyDomains || !cfg.CheckBackslash || !cfg.Bypass || !cfg.BypassTooManyRequests {
		t.Fatal("legacy flags not mapped correctly")
	}
	if !cfg.FilterTestLength || !cfg.FilterWrongStatus200 || !cfg.FilterWrongSubdomain || !cfg.FilterPossible404 {
		t.Fatal("legacy filter flags not mapped correctly")
	}
	if len(cfg.FilterContentTypes) != 1 || cfg.FilterContentTypes[0] != "text/html" {
		t.Fatal("legacy content-type flag not parsed")
	}
}

func TestFfufAliases(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-mc", "200,301",
		"-fc", "404",
		"-ms", "100",
		"-fs", "50",
		"-mw", "10",
		"-fw", "3",
		"-ml", "5",
		"-fls", "2",
		"-mt", "10-1000",
		"-ft", "2000",
		"-fr", "error",
		"-ac",
		"-acn", "3",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.FilterStatus) == 0 || len(cfg.FilterStatusNot) == 0 {
		t.Fatal("status match/filter aliases not applied")
	}
	if len(cfg.FilterLength) == 0 || len(cfg.FilterLengthNot) == 0 {
		t.Fatal("size match/filter aliases not applied")
	}
	if len(cfg.FilterWords) == 0 || len(cfg.FilterWordsNot) == 0 {
		t.Fatal("word match/filter aliases not applied")
	}
	if len(cfg.FilterLines) == 0 || len(cfg.FilterLinesNot) == 0 {
		t.Fatal("line match/filter aliases not applied")
	}
	if len(cfg.FilterTime) == 0 || len(cfg.FilterTimeNot) == 0 {
		t.Fatal("time match/filter aliases not applied")
	}
	if cfg.FilterMatchRegexNot == nil {
		t.Fatal("regex filter alias not applied")
	}
	if !cfg.AutoCalibrate || cfg.AutoCalibrateN != 3 {
		t.Fatal("auto calibration flags not applied")
	}
}

func TestHttpMethodAndDataFlags(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-X", "PUT",
		"-d", "name=test",
		"-H", "Content-Type:application/x-www-form-urlencoded",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.RequestMethod != "PUT" {
		t.Fatalf("expected method PUT, got %q", cfg.RequestMethod)
	}
	if cfg.RequestData != "name=test" {
		t.Fatalf("expected data set")
	}
	if cfg.RequestHeaders["Content-Type"] == "" {
		t.Fatalf("expected Content-Type header")
	}
}

func TestDashDWordlistDisambiguation(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-d", "default",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Wordlist != "default" {
		t.Fatalf("expected wordlist default, got %q", cfg.Wordlist)
	}

	cfg2, err := Load([]string{
		"-u", "example.com",
		"-d", "name=test",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg2.RequestData != "name=test" {
		t.Fatalf("expected data name=test, got %q", cfg2.RequestData)
	}
}

func TestStopFlags(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-sf", "403,401",
		"-se",
		"-sa", "10",
		"-resume", "/tmp/resume.txt",
		"-resume-every", "500",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.StopOnStatus) == 0 || !cfg.StopOnErrors || cfg.StopOnMatches != 10 || cfg.ResumeFile == "" || cfg.ResumeEvery != 500 {
		t.Fatal("stop/resume flags not applied")
	}
}

func TestResumeRequiresResumeEvery(t *testing.T) {
	_, err := Load([]string{"-u", "https://example.com", "-w", "default", "-resume", "/tmp/r.json"})
	if err == nil {
		t.Fatal("expected error when -resume is set without -resume-every")
	}
	if !strings.Contains(err.Error(), "resume-every") {
		t.Errorf("error should mention resume-every, got: %v", err)
	}
}

func TestEnqueueModuleUrlsRequiresModule(t *testing.T) {
	_, err := Load([]string{"-u", "https://example.com", "-w", "default", "-modules", "fingerprint", "-enqueue-module-urls", "links"})
	if err == nil {
		t.Fatal("expected error when enqueue-module-urls lists a module not in -modules")
	}
	if !strings.Contains(err.Error(), "links") || !strings.Contains(err.Error(), "modules") {
		t.Errorf("error should mention links and modules, got: %v", err)
	}
}

func TestShowStatusExcludesMatchStatus(t *testing.T) {
	_, err := Load([]string{"-u", "https://example.com", "-w", "default", "-s", "-mc", "200"})
	if err == nil {
		t.Fatal("expected error when -s and -mc are used together")
	}
	if !strings.Contains(err.Error(), "show all status") || !strings.Contains(err.Error(), "-mc") {
		t.Errorf("error should mention -s and -mc, got: %v", err)
	}
}

func TestDelayParse(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-p", "0.1-0.2",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.DelayMin == 0 || cfg.DelayMax == 0 {
		t.Fatal("delay not parsed")
	}
}

func TestReplayOnMatchFlag(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-replay-proxy", "http://127.0.0.1:8080",
		"-replay-on-match=false",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ReplayProxy == "" || cfg.ReplayOnMatch {
		t.Fatal("replay-on-match flag not applied")
	}
}

func TestBypassAndWAFFlags(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-bypass-budget", "5",
		"-bypass-ratio", "0.2",
		"-waf-adaptive",
		"-waf-threshold", "10",
		"-waf-factor", "3.0",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.BypassBudget != 5 || cfg.BypassRatioLimit != 0.2 {
		t.Fatal("bypass settings not applied")
	}
	if !cfg.WAFAdaptive || cfg.WAFSlowdownThreshold != 10 || cfg.WAFSlowdownFactor != 3.0 {
		t.Fatal("waf settings not applied")
	}
}

func TestResumeEveryFlag(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-resume", "/tmp/resume.json",
		"-resume-every", "100",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ResumeEvery != 100 {
		t.Fatal("resume-every not applied")
	}
}

func TestWordlistSpecs(t *testing.T) {
	specs := ParseWordlistSpecs("list.txt:USER,pass.txt:PASS,default")
	if len(specs) != 3 {
		t.Fatalf("expected 3 specs, got %d", len(specs))
	}
	if specs[0].Keyword != "USER" || specs[1].Keyword != "PASS" || specs[2].Keyword != "FUZZ" {
		t.Fatal("keyword parsing failed")
	}
}

func TestParseEncoders(t *testing.T) {
	m := ParseEncoders("FUZZ:urlencode")
	if len(m) != 1 || len(m["FUZZ"]) != 1 || m["FUZZ"][0] != "urlencode" {
		t.Fatalf("ParseEncoders(FUZZ:urlencode) = %v", m)
	}
	m = ParseEncoders("FUZZ:urlencode,base64encode;PARAM:b64")
	if len(m) != 2 {
		t.Fatalf("expected 2 keywords, got %d", len(m))
	}
	if len(m["FUZZ"]) != 2 || m["FUZZ"][0] != "urlencode" || m["FUZZ"][1] != "base64encode" {
		t.Fatalf("FUZZ chain = %v", m["FUZZ"])
	}
	if len(m["PARAM"]) != 1 || m["PARAM"][0] != "b64" {
		t.Fatalf("PARAM = %v", m["PARAM"])
	}
	m = ParseEncoders("")
	if len(m) != 0 {
		t.Fatalf("empty spec should give empty map: %v", m)
	}
}

func TestLoadWithEncoders(t *testing.T) {
	cfg, err := Load([]string{"-u", "https://example.com/FUZZ", "-w", "default", "-enc", "FUZZ:urlencode"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Encoders == nil || len(cfg.Encoders["FUZZ"]) != 1 || cfg.Encoders["FUZZ"][0] != "urlencode" {
		t.Fatalf("Encoders not set: %v", cfg.Encoders)
	}
}

func TestDataFromFile(t *testing.T) {
	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "body.json")
	content := `{"user":"FUZZ","role":"admin"}`
	if err := os.WriteFile(bodyFile, []byte(content), 0644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	cfg, err := Load([]string{"-u", "https://example.com/", "-w", "default", "-d", "@" + bodyFile})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.RequestData != content {
		t.Fatalf("RequestData = %q, want file content", cfg.RequestData)
	}
	if cfg.RequestDataPath != bodyFile {
		t.Fatalf("RequestDataPath = %q, want %q", cfg.RequestDataPath, bodyFile)
	}
}

func TestDataFromFileNotFound(t *testing.T) {
	_, err := Load([]string{"-u", "https://example.com/", "-w", "default", "-d", "@/nonexistent/file.json"})
	if err == nil {
		t.Fatal("Load expected to fail for missing file")
	}
	if !strings.Contains(err.Error(), "nonexistent") && !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("error should mention missing file: %v", err)
	}
}

func TestDataFromFileSaveConfig(t *testing.T) {
	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "body.txt")
	content := `{"key":"FUZZ"}`
	if err := os.WriteFile(bodyFile, []byte(content), 0644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	cfgPath := filepath.Join(dir, "config.json")
	cfg, err := Load([]string{"-u", "https://example.com/", "-w", "default", "-d", "@" + bodyFile, "-save-config", cfgPath})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := Save(cfg, cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	// Reload from saved config; data should be "@path" so it reads the file again
	cfg2, err := Load([]string{"-cf", cfgPath})
	if err != nil {
		t.Fatalf("Load from saved config: %v", err)
	}
	if cfg2.RequestData != content {
		t.Fatalf("after reload RequestData = %q, want file content", cfg2.RequestData)
	}
}

func TestExtensionsAndInputModeFlags(t *testing.T) {
	cfg, err := Load([]string{
		"-u", "example.com",
		"-e", "php,txt",
		"-mode", "pitchfork",
		"-ic",
		"-awc",
		"-max-size", "2048",
		"-min-size", "10",
		"-ext-defaults",
		"-wordlist-case", "lower",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Extensions) < 2 {
		t.Fatalf("extensions not normalized, got %#v", cfg.Extensions)
	}
	foundPHP := false
	foundTXT := false
	for _, ext := range cfg.Extensions {
		if ext == ".php" {
			foundPHP = true
		}
		if ext == ".txt" {
			foundTXT = true
		}
	}
	if !foundPHP || !foundTXT {
		t.Fatalf("extensions missing, got %#v", cfg.Extensions)
	}
	if cfg.InputMode != "pitchfork" {
		t.Fatalf("input mode not set, got %q", cfg.InputMode)
	}
	if !cfg.IgnoreWordlistComments {
		t.Fatal("ignore comments flag not applied")
	}
	if !cfg.AutoWildcard {
		t.Fatal("auto wildcard flag not applied")
	}
	if cfg.MaxResponseSize != 2048 {
		t.Fatalf("max-size not applied, got %d", cfg.MaxResponseSize)
	}
	if cfg.MinResponseSize != 10 {
		t.Fatalf("min-size not applied, got %d", cfg.MinResponseSize)
	}
	if !cfg.UseDefaultExtensions {
		t.Fatal("default extensions flag not applied")
	}
	if cfg.RandomizeWordlistCase != "lower" {
		t.Fatalf("wordlist-case not applied, got %q", cfg.RandomizeWordlistCase)
	}
}

func TestListAndAdvancedFlags(t *testing.T) {
	dir := t.TempDir()
	listPath := filepath.Join(dir, "urls.txt")
	if err := os.WriteFile(listPath, []byte("example.com\nhttps://host.tld/path\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{
		"-list", listPath,
		"-bw", "blocked,ignore",
		"-is", "secret,token",
		"-nd",
		"-nd-len", "30",
		"-nd-words", "6",
		"-nd-lines", "4",
		"-random-ua",
		"-jitter",
		"-jitter-threshold", "900",
		"-jitter-factor", "1.5",
		"-exclude-paths", "js,img",
		"-q",
		"-verbs", "get,post",
		"-auto-verb",
		"-dump",
		"-dump-dir", "dumps",
		"-save-config", filepath.Join(dir, "cfg.json"),
		"-u", "base.example",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.URLs) != 3 {
		t.Fatalf("expected 3 urls, got %d", len(cfg.URLs))
	}
	if !cfg.NearDuplicates || cfg.NearDuplicateLenBucket != 30 || cfg.NearDuplicateWordBucket != 6 || cfg.NearDuplicateLineBucket != 4 {
		t.Fatal("near duplicate flags not applied")
	}
	if !cfg.RandomUserAgent {
		t.Fatal("random-ua flag not applied")
	}
	if !cfg.JitterProfile || cfg.JitterThresholdMS != 900 || cfg.JitterFactor != 1.5 {
		t.Fatal("jitter flags not applied")
	}
	if len(cfg.BlockWords) != 2 {
		t.Fatal("block words not applied")
	}
	if len(cfg.ExcludePaths) != 2 || !cfg.Quiet {
		t.Fatal("exclude paths or quiet not applied")
	}
	if len(cfg.InterestingStrings) != 2 {
		t.Fatal("interesting strings not applied")
	}
	if len(cfg.Verbs) == 0 || !cfg.AutoVerbs {
		t.Fatal("verbs not applied")
	}
	if !cfg.DumpResponses || cfg.DumpDir == "" || cfg.SaveConfigPath == "" {
		t.Fatal("dump/save-config not applied")
	}
}

func TestPresetQuick(t *testing.T) {
	cfg, err := Load([]string{"-u", "https://example.com/FUZZ", "-preset", "quick"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Concurrency != 5 {
		t.Errorf("preset quick: expected Concurrency 5, got %d", cfg.Concurrency)
	}
	if cfg.Wordlist != "fav" {
		t.Errorf("preset quick: expected Wordlist fav, got %s", cfg.Wordlist)
	}
	if cfg.MaxTime != 300 {
		t.Errorf("preset quick: expected MaxTime 300, got %d", cfg.MaxTime)
	}
	if len(cfg.ModuleConfig.Modules) != 1 || cfg.ModuleConfig.Modules[0] != "fingerprint" {
		t.Errorf("preset quick: expected modules [fingerprint], got %v", cfg.ModuleConfig.Modules)
	}
	if cfg.OutputFormat != "txt" {
		t.Errorf("preset quick: expected OutputFormat txt, got %s", cfg.OutputFormat)
	}
}

func TestPresetUnknown(t *testing.T) {
	_, err := Load([]string{"-u", "https://example.com", "-preset", "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
	if !strings.Contains(err.Error(), "unknown preset") || !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected error message about unknown preset, got: %v", err)
	}
}

func TestPresetStealth(t *testing.T) {
	cfg, err := Load([]string{"-u", "https://example.com/FUZZ", "-preset", "stealth"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Concurrency != 5 {
		t.Errorf("preset stealth: expected Concurrency 5, got %d", cfg.Concurrency)
	}
	if cfg.DelayMin != 500*time.Millisecond || cfg.DelayMax != 1*time.Second {
		t.Errorf("preset stealth: expected Delay 500ms–1s, got %v–%v", cfg.DelayMin, cfg.DelayMax)
	}
	if !cfg.RandomUserAgent {
		t.Errorf("preset stealth: expected RandomUserAgent true, got %v", cfg.RandomUserAgent)
	}
	if cfg.ThrottleRPS != 10 {
		t.Errorf("preset stealth: expected ThrottleRPS 10, got %d", cfg.ThrottleRPS)
	}
	if !cfg.JitterProfile {
		t.Errorf("preset stealth: expected JitterProfile true, got %v", cfg.JitterProfile)
	}
}

func TestPresetThorough(t *testing.T) {
	cfg, err := Load([]string{"-u", "https://example.com/FUZZ", "-preset", "thorough"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Concurrency != 30 {
		t.Errorf("preset thorough: expected Concurrency 30, got %d", cfg.Concurrency)
	}
	wantMods := []string{"fingerprint", "cors", "urlextract", "links"}
	if len(cfg.ModuleConfig.Modules) != len(wantMods) {
		t.Errorf("preset thorough: expected %d modules, got %v", len(wantMods), cfg.ModuleConfig.Modules)
	} else {
		for i, m := range wantMods {
			if cfg.ModuleConfig.Modules[i] != m {
				t.Errorf("preset thorough: modules[%d] want %q, got %q", i, m, cfg.ModuleConfig.Modules[i])
			}
		}
	}
	if !cfg.UseDefaultExtensions {
		t.Errorf("preset thorough: expected UseDefaultExtensions true, got %v", cfg.UseDefaultExtensions)
	}
	if cfg.Depth != 2 {
		t.Errorf("preset thorough: expected Depth 2, got %d", cfg.Depth)
	}
	if cfg.OutputFormat != "json" {
		t.Errorf("preset thorough: expected OutputFormat json, got %s", cfg.OutputFormat)
	}
}

func TestPresetCLIOverridesPreset(t *testing.T) {
	// CLI is applied after preset, so explicit flags win
	cfg, err := Load([]string{"-u", "https://example.com/FUZZ", "-preset", "quick", "-c", "99", "-of", "json"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Concurrency != 99 {
		t.Errorf("CLI override: expected Concurrency 99 (from -c), got %d", cfg.Concurrency)
	}
	if cfg.OutputFormat != "json" {
		t.Errorf("CLI override: expected OutputFormat json (from -of), got %s", cfg.OutputFormat)
	}
	// Preset values that were not overridden stay
	if cfg.Wordlist != "fav" || cfg.MaxTime != 300 {
		t.Errorf("CLI override: preset values should remain where not overridden: wordlist=%s maxtime=%d", cfg.Wordlist, cfg.MaxTime)
	}
}

func TestPresetCaseInsensitive(t *testing.T) {
	for _, name := range []string{"QUICK", "Quick", "stealth", "THOROUGH"} {
		cfg, err := Load([]string{"-u", "https://example.com", "-preset", name})
		if err != nil {
			t.Fatalf("Load with -preset %q failed: %v", name, err)
		}
		if cfg == nil {
			t.Fatalf("Load with -preset %q returned nil config", name)
		}
	}
}
