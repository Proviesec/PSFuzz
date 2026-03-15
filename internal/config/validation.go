package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// DefaultMaxResponseSizeCap is the response body size cap (10 MiB) used when MaxResponseSize is 0.
// When -max-size 0 (default), responses are limited to this size to avoid memory exhaustion from large or streaming bodies.
const DefaultMaxResponseSizeCap = 10 * 1024 * 1024

// EffectiveMaxResponseSize returns the limit to use for reading response bodies.
// If MaxResponseSize is 0, returns DefaultMaxResponseSizeCap; otherwise returns MaxResponseSize.
func EffectiveMaxResponseSize(cfg *Config) int {
	if cfg == nil {
		return DefaultMaxResponseSizeCap
	}
	if cfg.MaxResponseSize > 0 {
		return cfg.MaxResponseSize
	}
	return DefaultMaxResponseSizeCap
}

// validateConfig runs after config file and CLI merge. It normalizes fields, applies defaults where needed, and returns an error if the config is invalid.
func validateConfig(cfg *Config, visited map[string]bool) error {
	// -s (show all status) and -mc/-fsc (match status) are mutually exclusive
	if cfg.ShowStatus && isSet(visited, "mc", "fsc", "filterStatusCode") && len(cfg.FilterStatus) > 0 {
		return errors.New("cannot use -s (show all status) together with -mc/-fsc (match status); -s shows all responses and ignores status filter")
	}
	if cfg.ShowStatus {
		cfg.FilterStatus = nil
	}
	// Apply default match status when none was set (ensures 302 etc. are included).
	if (cfg.FilterStatus == nil || len(cfg.FilterStatus) == 0) && !cfg.ShowStatus {
		cfg.FilterStatus = defaultMatcher()
	}

	if cfg.URL == "" && len(cfg.URLs) == 0 && cfg.SaveConfigPath == "" {
		return errors.New("-u/-url or -list is required")
	}
	if cfg.URL != "" && !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
		cfg.URL = "https://" + cfg.URL
	}
	if cfg.URL != "" && len(cfg.URLs) > 0 {
		cfg.URLs = appendIfMissing(cfg.URLs, cfg.URL)
	}
	if len(cfg.URLs) > 0 {
		cfg.URLs = normalizeURLList(cfg.URLs)
	}
	if cfg.Concurrency <= 0 {
		return errors.New("concurrency must be > 0")
	}
	if cfg.Depth < 0 {
		return errors.New("depth must be >= 0")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if !validOutputFormats[cfg.OutputFormat] {
		return fmt.Errorf("unsupported output format %q", cfg.OutputFormat)
	}
	cfg.InputMode = normalizeInputMode(cfg.InputMode)
	if !isInputModeValid(cfg.InputMode) {
		return fmt.Errorf("unsupported input mode %q", cfg.InputMode)
	}
	if cfg.MaxResponseSize < 0 {
		return fmt.Errorf("max-size must be >= 0")
	}
	if cfg.MinResponseSize < 0 {
		return fmt.Errorf("min-size must be >= 0")
	}
	if cfg.MaxResponseSize > 0 && cfg.MinResponseSize > cfg.MaxResponseSize {
		return fmt.Errorf("min-size (%d) cannot exceed max-size (%d)", cfg.MinResponseSize, cfg.MaxResponseSize)
	}
	if cfg.RandomizeWordlistCase != "" && cfg.RandomizeWordlistCase != "lower" && cfg.RandomizeWordlistCase != "upper" {
		return fmt.Errorf("wordlist-case must be lower or upper")
	}
	if cfg.RecursionStrategy != "default" && cfg.RecursionStrategy != "greedy" {
		return fmt.Errorf("recursion-strategy must be default or greedy")
	}
	if cfg.MaxTime < 0 || cfg.MaxTimeJob < 0 {
		return errors.New("maxtime and maxtime-job must be >= 0")
	}
	if cfg.AutoVerbs && len(cfg.Verbs) == 0 {
		cfg.Verbs = defaultVerbs()
	}
	if len(cfg.Verbs) > 0 {
		cfg.Verbs = normalizeVerbs(cfg.Verbs)
	}
	if cfg.UseDefaultExtensions {
		cfg.Extensions = mergeExtensions(cfg.Extensions, defaultExtensions())
	}
	cfg.Wordlists = ParseWordlistSpecs(cfg.Wordlist)

	// enqueue-module-urls: every listed module must be in -modules
	if cfg.ModuleConfig.EnqueueModuleUrls != "" {
		enqueueNames := ParseCSV(cfg.ModuleConfig.EnqueueModuleUrls)
		moduleSet := make(map[string]bool)
		for _, m := range cfg.ModuleConfig.Modules {
			moduleSet[strings.ToLower(strings.TrimSpace(m))] = true
		}
		for _, name := range enqueueNames {
			n := strings.ToLower(strings.TrimSpace(name))
			if n == "" {
				continue
			}
			if !moduleSet[n] {
				return fmt.Errorf("enqueue-module-urls includes %q but that module is not in -modules (add it to -modules for enqueue to work)", name)
			}
		}
	}

	if cfg.ResumeFile != "" && cfg.ResumeEvery <= 0 {
		return errors.New("when using -resume set -resume-every to a positive number (e.g. 500) so progress is saved during the run")
	}

	return nil
}
