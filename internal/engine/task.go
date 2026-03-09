package engine

import (
	"net/url"
	"strings"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
)

func shouldRecurse(taskURL string, depth int, statusCode int, cfg *config.Config) (string, bool) {
	if depth >= cfg.Depth {
		return "", false
	}
	if cfg.RecursionStrategy != "greedy" {
		if cfg.RecursionSmart && !config.MatchAnyRange(cfg.RecursionStatus, statusCode) {
			return "", false
		}
	}
	u, err := url.Parse(taskURL)
	if err != nil {
		return "", false
	}
	// Check if the last path segment looks like a file (has extension)
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) > 0 {
		lastSegment := segments[len(segments)-1]
		if strings.Contains(lastSegment, ".") {
			return "", false
		}
	}
	base := strings.TrimSuffix(taskURL, "/") + "/"
	return base, true
}

func buildBaseURL(base, word string, checkBackslash bool) string {
	if strings.Contains(base, "#PSFUZZ#") {
		return strings.Replace(base, "#PSFUZZ#", word, 1)
	}
	if checkBackslash {
		return base + word
	}
	return strings.TrimSuffix(base, "/") + "/" + word
}

func buildRecursiveURL(base, word string) string {
	u := strings.TrimSuffix(base, "/") + "/" + word
	if !strings.Contains(word, ".") {
		u += "/"
	}
	return u
}

func isExcludedURL(rawURL string, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}
	path := rawURL
	if u, err := url.Parse(rawURL); err == nil {
		path = u.Path
	}
	path = "/" + strings.Trim(path, "/") + "/"
	for _, ex := range excludes {
		ex = strings.ToLower(strings.TrimSpace(ex))
		if ex == "" {
			continue
		}
		ex = "/" + strings.Trim(ex, "/") + "/"
		if strings.Contains(strings.ToLower(path), ex) {
			return true
		}
	}
	return false
}

func bypassVariants(rawURL string) []Task {
	pathVariants := []string{
		"/*", "//.", "/%2e/", "/%2f/", "/./", "/", "/*/", "/..;/", "/..%3B/", "////",
		"/%20", "/%09", "/%2e%2e/", "/%2e%2f", "/%2e%2e%2f", "/%2f%2e%2e%2f",
		"/;%2f", "/..%00/", "/%00", "%00", "#test",
	}
	extVariants := []string{".yml", ".php", ".html", ".zip", ".txt", ".yaml", ".wadl", ".htm", ".asp", ".aspx", ".bak", ".old"}
	methods := []string{"HEAD", "OPTIONS", "TRACE"}
	out := make([]Task, 0, len(pathVariants)+len(extVariants)+len(methods)+30)
	for _, v := range pathVariants {
		out = append(out, Task{URL: rawURL + v})
	}
	for _, v := range extVariants {
		out = append(out, Task{URL: rawURL + v})
	}
	for _, m := range methods {
		out = append(out, Task{URL: rawURL, Method: m})
	}
	if u, err := url.Parse(rawURL); err == nil {
		path := u.Path
		if u.RawQuery != "" {
			path += "?" + u.RawQuery
		}
		if path == "" {
			path = "/"
		}
		queryVariants := []string{"?debug=true", "?admin=true", "?bypass=1", "?_=%00", "?_=" + time.Now().Format("150405")}
		for _, q := range queryVariants {
			appendURL := rawURL
			if u.RawQuery != "" {
				appendURL = rawURL + "&" + strings.TrimPrefix(q, "?")
			} else {
				appendURL = rawURL + q
			}
			out = append(out, Task{URL: appendURL})
		}
		headerVariants := []map[string]string{
			{"X-Original-URL": path},
			{"X-Rewrite-URL": path},
			{"X-Custom-IP-Authorization": "127.0.0.1"},
			{"X-Forwarded-For": "127.0.0.1"},
			{"X-Forwarded-For": "127.0.0.1, 127.0.0.1"},
			{"X-Forwarded-For": "localhost"},
			{"X-Real-IP": "127.0.0.1"},
			{"X-Client-IP": "127.0.0.1"},
			{"Forwarded": "for=127.0.0.1;proto=https"},
			{"X-Forwarded-Proto": u.Scheme},
			{"X-HTTP-Method-Override": "GET"},
			{"X-HTTP-Method": "GET"},
			{"X-Method-Override": "GET"},
		}
		if u.Host != "" {
			headerVariants = append(headerVariants,
				map[string]string{"X-Forwarded-Host": u.Host},
				map[string]string{"X-Host": u.Host},
				map[string]string{"Forwarded": "for=127.0.0.1;proto=https;host=" + u.Host},
			)
		}
		for _, hv := range headerVariants {
			out = append(out, Task{URL: rawURL, Headers: hv})
		}
	}
	return out
}
