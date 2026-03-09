// Package config loads and applies PSFuzz configuration from CLI flags, optional JSON file (-cf), and presets.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Load parses command-line args and optional config file (-cf) into a Config. Preset and validation are applied.
// Returns an error if required flags are missing or values are invalid.
func Load(args []string) (*Config, error) {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("psfuzz", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	cli := &cliConfig{}
	registerFlags(fs, cli)

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	cfg := defaultConfig()
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	configPath := cli.ConfigFilePath
	// Config is only loaded when explicitly given via -cf / -configfile (no auto-load of config.json).
	if configPath != "" {
		if err := applyFileConfig(cfg, configPath); err != nil {
			return nil, fmt.Errorf("config file %s: %w", configPath, err)
		}
		cfg.ConfigFilePath = configPath
	}

	// Preset applies a set of defaults; apply before CLI so flags can override.
	if err := applyPreset(cfg, cli.Preset); err != nil {
		return nil, err
	}

	if err := applyCLIConfig(cfg, cli, visited); err != nil {
		return nil, fmt.Errorf("apply CLI: %w", err)
	}

	if err := validateConfig(cfg, visited); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyPreset applies a named preset (quick, stealth, thorough) to cfg. CLI flags applied later override preset values.
func applyPreset(cfg *Config, name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil
	}
	switch name {
	case "quick":
		cfg.Concurrency = 5
		cfg.Wordlist = "fav"
		cfg.MaxTime = 300
		cfg.ModuleConfig.Modules = []string{"fingerprint"}
		cfg.OutputFormat = "txt"
	case "stealth":
		cfg.Concurrency = 5
		cfg.DelayMin = 500 * time.Millisecond
		cfg.DelayMax = 1 * time.Second
		cfg.RandomUserAgent = true
		cfg.ThrottleRPS = 10
		cfg.JitterProfile = true
	case "thorough":
		cfg.Concurrency = 30
		cfg.ModuleConfig.Modules = []string{"fingerprint", "cors", "urlextract", "links"}
		cfg.UseDefaultExtensions = true
		cfg.Depth = 2
		cfg.OutputFormat = "json"
	default:
		return fmt.Errorf("unknown preset %q (use quick, stealth, or thorough)", name)
	}
	return nil
}

func ParseStatusRanges(raw string) ([]StatusRange, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	r, err := ParseRanges(raw)
	if err != nil {
		return nil, fmt.Errorf("parse status range: %w", err)
	}
	return r, nil
}

// applyRanges parses src as a range list and sets *dst. Used by applyFileConfig and applyCLIConfig.
func applyRanges(dst *[]Range, src string) error {
	r, err := ParseRanges(src)
	if err != nil {
		return err
	}
	*dst = r
	return nil
}

// applyStatusRanges parses src as a status range list and sets *dst.
func applyStatusRanges(dst *[]StatusRange, src string) error {
	r, err := ParseStatusRanges(src)
	if err != nil {
		return err
	}
	*dst = r
	return nil
}

// applyRangesFromPtr is like applyRanges but only runs when src is non-nil (for file config).
func applyRangesFromPtr(dst *[]Range, src *string) error {
	if src == nil {
		return nil
	}
	return applyRanges(dst, *src)
}

// applyStatusRangesFromPtr is like applyStatusRanges but only runs when src is non-nil (for file config).
func applyStatusRangesFromPtr(dst *[]StatusRange, src *string) error {
	if src == nil {
		return nil
	}
	return applyStatusRanges(dst, *src)
}

func ParseRanges(raw string) ([]Range, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := ParseCSV(raw)
	out := make([]Range, 0, len(parts))
	for _, p := range parts {
		if strings.Contains(p, "-") {
			s := strings.SplitN(p, "-", 2)
			min, err := strconv.Atoi(strings.TrimSpace(s[0]))
			if err != nil {
				return nil, err
			}
			max, err := strconv.Atoi(strings.TrimSpace(s[1]))
			if err != nil {
				return nil, err
			}
			if max < min {
				return nil, fmt.Errorf("invalid range %q", p)
			}
			out = append(out, Range{Min: min, Max: max})
			continue
		}
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		out = append(out, Range{Min: v, Max: v})
	}
	return out, nil
}

func ParseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadURLList(raw string) []string {
	return loadList(raw, true)
}

func loadWordList(raw string) []string {
	words := loadList(raw, false)
	out := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.TrimSpace(strings.ToLower(w))
		if w == "" {
			continue
		}
		out = append(out, w)
	}
	return out
}

func loadPathList(raw string) []string {
	return loadList(raw, false)
}

func loadList(raw string, ignoreComments bool) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if isFilePath(raw) {
		lines, err := readLines(raw, ignoreComments)
		if err == nil {
			return lines
		}
	}
	return ParseCSV(raw)
}

func readLines(path string, ignoreComments bool) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if ignoreComments && isCommentLine(line) {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

func isFilePath(path string) bool {
	if strings.Contains(path, ",") {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func normalizeExtensions(exts []string) []string {
	if len(exts) == 0 {
		return nil
	}
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out = append(out, e)
	}
	return out
}

func mergeExtensions(base []string, extra []string) []string {
	if len(extra) == 0 {
		return normalizeExtensions(base)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(extra))
	for _, e := range normalizeExtensions(base) {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	for _, e := range normalizeExtensions(extra) {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func normalizeVerbs(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, v := range raw {
		v = strings.TrimSpace(strings.ToUpper(v))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func normalizeInputMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return "clusterbomb"
	}
	return mode
}

func isInputModeValid(mode string) bool {
	switch mode {
	case "clusterbomb", "pitchfork", "sniper":
		return true
	default:
		return false
	}
}

func normalizeURLList(urls []string) []string {
	out := make([]string, 0, len(urls))
	seen := map[string]struct{}{}
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			raw = "https://" + raw
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}
	return out
}

func appendIfMissing(list []string, value string) []string {
	for _, v := range list {
		if v == value {
			return list
		}
	}
	return append(list, value)
}

func ParseKV(raw string, sep string) map[string]string {
	result := map[string]string{}
	for _, token := range ParseCSV(raw) {
		pair := strings.SplitN(token, sep, 2)
		if len(pair) != 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		value := strings.TrimSpace(pair[1])
		if key != "" {
			result[key] = value
		}
	}
	return result
}

// setRequestDataFromSpec sets RequestData (and optionally RequestDataPath) from -d / data spec.
// If spec starts with "@", the rest is a file path; the file content is read and used as body template.
func setRequestDataFromSpec(cfg *Config, spec string) error {
	if spec == "" {
		cfg.RequestData = ""
		cfg.RequestDataPath = ""
		return nil
	}
	if strings.HasPrefix(spec, "@") {
		path := strings.TrimSpace(spec[1:])
		if path == "" {
			return errors.New("data file path after @ must not be empty")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read data file %q: %w", path, err)
		}
		cfg.RequestData = string(b)
		cfg.RequestDataPath = path
		return nil
	}
	cfg.RequestData = spec
	cfg.RequestDataPath = ""
	return nil
}

// ParseEncoders parses encoder spec string into keyword -> encoder chain.
// Format: "KEYWORD:enc1,enc2;OTHER:enc3" (semicolon separates keywords, comma separates encoder chain).
func ParseEncoders(raw string) map[string][]string {
	out := make(map[string][]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, ":")
		if idx <= 0 {
			continue
		}
		keyword := strings.TrimSpace(part[:idx])
		encList := strings.TrimSpace(part[idx+1:])
		if keyword == "" {
			continue
		}
		var chain []string
		for _, e := range strings.Split(encList, ",") {
			e = strings.TrimSpace(strings.ToLower(e))
			if e != "" {
				chain = append(chain, e)
			}
		}
		if len(chain) > 0 {
			out[keyword] = chain
		}
	}
	return out
}

// ParseExploreAIWordlistMap parses "name:path_or_url,name2:path2" into a map. Keys are lowercased for lookup.
func ParseExploreAIWordlistMap(raw string) map[string]string {
	out := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, ":")
		if idx <= 0 || idx == len(part)-1 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(part[:idx]))
		val := strings.TrimSpace(part[idx+1:])
		if key != "" && val != "" {
			out[key] = val
		}
	}
	return out
}

func parseDelay(raw string) (time.Duration, time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, nil
	}
	if strings.Contains(raw, "-") {
		parts := strings.SplitN(raw, "-", 2)
		min, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse delay min: %w", err)
		}
		max, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse delay max: %w", err)
		}
		if max < min {
			return 0, 0, fmt.Errorf("invalid delay range")
		}
		return time.Duration(min * float64(time.Second)), time.Duration(max * float64(time.Second)), nil
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse delay: %w", err)
	}
	d := time.Duration(val * float64(time.Second))
	return d, d, nil
}

func normalizeArgs(args []string) []string {
	out := make([]string, 0, len(args))
	var headers []string
	var encs []string
	var wordlists []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-H" && i+1 < len(args) {
			headers = append(headers, args[i+1])
			i++
			continue
		}
		if (arg == "-enc" || arg == "--enc") && i+1 < len(args) {
			encs = append(encs, args[i+1])
			i++
			continue
		}
		if arg == "-w" && i+1 < len(args) {
			wordlists = append(wordlists, args[i+1])
			i++
			continue
		}
		if arg == "-d" && i+1 < len(args) {
			next := args[i+1]
			if isWordlistArg(next) {
				wordlists = append(wordlists, next)
			} else {
				out = append(out, "-data", next)
			}
			i++
			continue
		}
		out = append(out, arg)
	}
	if len(wordlists) > 0 {
		out = append(out, "-w", strings.Join(wordlists, ","))
	}
	if len(headers) > 0 {
		out = append(out, "-H", strings.Join(headers, ","))
	}
	if len(encs) > 0 {
		out = append(out, "-enc", strings.Join(encs, ";"))
	}
	return out
}

func isWordlistArg(v string) bool {
	if v == "" {
		return false
	}
	if strings.Contains(v, ":") {
		split := strings.SplitN(v, ":", 2)
		v = split[0]
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return true
	}
	switch v {
	case "default", "fav", "subdomain":
		return true
	}
	if strings.Contains(v, "/") || strings.Contains(v, ".") {
		if _, err := os.Stat(v); err == nil {
			return true
		}
	}
	return false
}

func MatchAnyRange(ranges []Range, v int) bool {
	if len(ranges) == 0 {
		return false
	}
	for _, r := range ranges {
		if v >= r.Min && v <= r.Max {
			return true
		}
	}
	return false
}
