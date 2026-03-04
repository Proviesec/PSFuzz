package modules

import (
	"context"
	"regexp"
	"strings"
)

// FingerprintAnalyzer detects technologies from response headers and body.
type FingerprintAnalyzer struct{}

func init() {
	Register("fingerprint", func(*Config) Analyzer { return FingerprintAnalyzer{} })
}

func (FingerprintAnalyzer) Name() string { return "fingerprint" }

func (FingerprintAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}
	var tech []string
	seen := map[string]bool{}

	// Case-insensitive header lookup (Go sends canonical keys like "Server")
	getHeader := func(name string) string {
		for k, v := range in.Headers {
			if strings.EqualFold(k, name) {
				return v
			}
		}
		return ""
	}

	// Header signatures: header name -> pattern (substring or regex) -> technology name
	headerSigs := map[string]map[string]string{
		"server": {
			"nginx": "nginx", "apache": "apache", "cloudflare": "cloudflare",
			"microsoft-iis": "iis", "openresty": "openresty", "litespeed": "litespeed",
			"php": "php", "tomcat": "tomcat", "jetty": "jetty", "gunicorn": "gunicorn",
		},
		"x-powered-by": {
			"php": "php", "asp.net": "aspnet", "express": "express", "next.js": "nextjs",
		},
		"x-generator": {
			"wordpress": "wordpress", "drupal": "drupal", "joomla": "joomla",
		},
		"x-aspnet-version": {"": "aspnet"},
	}
	for h, patterns := range headerSigs {
		v := getHeader(h)
		if v == "" {
			continue
		}
		lower := strings.ToLower(v)
		for pattern, name := range patterns {
			if pattern == "" || strings.Contains(lower, pattern) {
				if !seen[name] {
					tech = append(tech, name)
					seen[name] = true
				}
				break
			}
		}
	}

	// Body signatures: substring or regex -> technology
	bodySigs := []struct {
		re   *regexp.Regexp
		sub  string
		name string
	}{
		{sub: "wp-content", name: "wordpress"},
		{sub: "wp-includes", name: "wordpress"},
		{sub: "generator=\"wordpress\"", name: "wordpress"},
		{sub: "joomla", name: "joomla"},
		{sub: "drupal", name: "drupal"},
		{sub: "powered by php", name: "php"},
		{sub: "laravel", name: "laravel"},
		{sub: "react", name: "react"},
		{sub: "vue.js", name: "vue"},
		{sub: "angular", name: "angular"},
		{sub: "next/", name: "nextjs"},
		{sub: "nuxt", name: "nuxt"},
		{sub: "asp.net", name: "aspnet"},
		{sub: "__VIEWSTATE", name: "aspnet"},
		{sub: "jquery", name: "jquery"},
		{sub: "bootstrap", name: "bootstrap"},
	}
	bodyLower := strings.ToLower(in.Body)
	if len(bodyLower) > 100000 {
		bodyLower = bodyLower[:100000]
	}
	for _, sig := range bodySigs {
		if seen[sig.name] {
			continue
		}
		if sig.re != nil {
			if sig.re.MatchString(bodyLower) {
				tech = append(tech, sig.name)
				seen[sig.name] = true
			}
		} else if sig.sub != "" && strings.Contains(bodyLower, sig.sub) {
			tech = append(tech, sig.name)
			seen[sig.name] = true
		}
	}

	if len(tech) == 0 {
		return Output{Data: nil}, nil
	}
	return Output{Data: map[string]any{"technologies": tech}}, nil
}
