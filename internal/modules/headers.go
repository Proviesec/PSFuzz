package modules

import (
	"context"
	"strings"
)

// HeadersAnalyzer evaluates security-related response headers and flags missing or weak values.
// Covers: Content-Security-Policy, X-Frame-Options, Strict-Transport-Security,
// X-Content-Type-Options, and Set-Cookie (Secure, HttpOnly).
type HeadersAnalyzer struct{}

func init() {
	Register("headers", func(*Config) Analyzer { return HeadersAnalyzer{} })
}

func (HeadersAnalyzer) Name() string { return "headers" }

func (HeadersAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}
	get := func(name string) string {
		for k, v := range in.Headers {
			if strings.EqualFold(k, name) {
				return v
			}
		}
		return ""
	}

	csp := get("Content-Security-Policy")
	xfo := get("X-Frame-Options")
	hsts := get("Strict-Transport-Security")
	xcto := get("X-Content-Type-Options")
	setCookie := get("Set-Cookie")

	var missing []string
	var weak []string

	if csp == "" {
		missing = append(missing, "Content-Security-Policy")
	} else if isWeakCSP(csp) {
		weak = append(weak, "Content-Security-Policy")
	}
	if xfo == "" {
		missing = append(missing, "X-Frame-Options")
	} else if !strings.EqualFold(strings.TrimSpace(xfo), "DENY") && !strings.EqualFold(strings.TrimSpace(xfo), "SAMEORIGIN") {
		weak = append(weak, "X-Frame-Options")
	}
	if hsts == "" {
		missing = append(missing, "Strict-Transport-Security")
	} else if !strings.Contains(strings.ToLower(hsts), "max-age") {
		weak = append(weak, "Strict-Transport-Security")
	}
	if xcto == "" {
		missing = append(missing, "X-Content-Type-Options")
	} else if !strings.EqualFold(strings.TrimSpace(xcto), "nosniff") {
		weak = append(weak, "X-Content-Type-Options")
	}

	cookieSecure := true
	cookieHTTPOnly := true
	if setCookie != "" {
		cookieSecure = hasCookieFlag(setCookie, "Secure")
		cookieHTTPOnly = hasCookieFlag(setCookie, "HttpOnly")
	}

	data := map[string]any{
		"content_security_policy":     csp,
		"x_frame_options":             xfo,
		"strict_transport_security":   hsts,
		"x_content_type_options":     xcto,
		"missing_headers":             missing,
		"weak_headers":                weak,
		"set_cookie_present":          setCookie != "",
		"cookie_has_secure":           cookieSecure,
		"cookie_has_httponly":         cookieHTTPOnly,
	}
	// Short summary for TXT/CSV: "missing=X,Y; weak=Z" or "ok"
	if len(missing) > 0 || len(weak) > 0 || !cookieSecure || !cookieHTTPOnly {
		data["issues"] = formatHeaderIssues(missing, weak, cookieSecure, cookieHTTPOnly)
	}
	return Output{Data: data}, nil
}

func isWeakCSP(csp string) bool {
	cspLower := strings.ToLower(strings.TrimSpace(csp))
	return strings.Contains(cspLower, "unsafe-inline") ||
		strings.Contains(cspLower, "unsafe-eval") ||
		strings.Contains(cspLower, "*")
}

func hasCookieFlag(cookieValue, flag string) bool {
	parts := strings.Split(strings.TrimSpace(cookieValue), ";")
	for _, p := range parts[1:] {
		if strings.TrimSpace(strings.ToLower(p)) == strings.ToLower(flag) {
			return true
		}
	}
	return false
}

func formatHeaderIssues(missing, weak []string, cookieSecure, cookieHTTPOnly bool) string {
	var parts []string
	if len(missing) > 0 {
		parts = append(parts, "missing="+strings.Join(missing, ","))
	}
	if len(weak) > 0 {
		parts = append(parts, "weak="+strings.Join(weak, ","))
	}
	if !cookieSecure {
		parts = append(parts, "cookie_no_secure")
	}
	if !cookieHTTPOnly {
		parts = append(parts, "cookie_no_httponly")
	}
	return strings.Join(parts, "; ")
}
