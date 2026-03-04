package modules

import (
	"context"
	"regexp"
	"strings"
)

// AuthAnalyzer detects auth-related responses: login/logout forms, 401/302 to login,
// "session expired" / "please log in" text, and cookie-based auth hints (Set-Cookie).
// Helps prioritize auth flows for testing.
type AuthAnalyzer struct{}

func init() {
	Register("auth", func(*Config) Analyzer { return AuthAnalyzer{} })
}

func (AuthAnalyzer) Name() string { return "auth" }

var (
	// Login form: <form ...> with login/signin in action or body, and password input.
	loginFormRE   = regexp.MustCompile(`(?is)<form[^>]*>.*?(?:login|signin|sign[- ]?in|auth).*?<input[^>]*(?:type\s*=\s*["']?password["']?|name\s*=\s*["']?password["']?|id\s*=\s*["']?password["']?)`)
	logoutMatchRE = regexp.MustCompile(`(?i)(logout|log\s*out|sign\s*out|signout)`)
)

func (AuthAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}

	var hints []string
	body := in.Body
	bodyLower := strings.ToLower(body)

	// Login form: form with password field and login/signin context
	if loginFormRE.MatchString(body) {
		hints = append(hints, "login_form")
	} else if strings.Contains(bodyLower, "<form") &&
		(strings.Contains(bodyLower, "password") || strings.Contains(bodyLower, "type=\"password\"")) &&
		(strings.Contains(bodyLower, "login") || strings.Contains(bodyLower, "signin") || strings.Contains(bodyLower, "sign-in") || strings.Contains(bodyLower, "auth")) {
		hints = append(hints, "login_form")
	}

	// Logout link/button
	if logoutMatchRE.MatchString(body) {
		hints = append(hints, "logout_present")
	}

	// 401 Unauthorized
	if in.StatusCode == 401 {
		hints = append(hints, "status_401")
	}

	// 302/301 redirect to login
	if (in.StatusCode == 302 || in.StatusCode == 301) && locationContainsLogin(in.Headers) {
		hints = append(hints, "redirect_to_login")
	}

	// Session expired / please log in (body text)
	sessionPhrases := []string{"session expired", "please log in", "please sign in", "login required", "authentication required", "sign in to continue", "you must be logged in", "unauthorized access"}
	for _, p := range sessionPhrases {
		if strings.Contains(bodyLower, p) {
			hints = append(hints, "session_or_login_required")
			break
		}
	}

	// Set-Cookie with auth-like name (session, auth, token, sid, jwt)
	if setCookieHasAuthName(in.Headers) {
		hints = append(hints, "auth_cookie_set")
	}

	if len(hints) == 0 {
		return Output{Data: nil}, nil
	}

	seen := make(map[string]bool)
	var unique []string
	for _, h := range hints {
		if !seen[h] {
			seen[h] = true
			unique = append(unique, h)
		}
	}

	return Output{
		Data: map[string]any{
			"hints": unique,
			"count": len(unique),
		},
	}, nil
}

func locationContainsLogin(headers map[string]string) bool {
	for k, v := range headers {
		if !strings.EqualFold(k, "Location") {
			continue
		}
		vLower := strings.ToLower(v)
		return strings.Contains(vLower, "login") ||
			strings.Contains(vLower, "signin") ||
			strings.Contains(vLower, "auth") ||
			strings.Contains(vLower, "sso")
	}
	return false
}

func setCookieHasAuthName(headers map[string]string) bool {
	authNames := []string{"session", "auth", "token", "sid", "jwt", "sess", "csrf", "xsrf"}
	for k, v := range headers {
		if !strings.EqualFold(k, "Set-Cookie") {
			continue
		}
		vLower := strings.ToLower(v)
		// First cookie name is before = or ;
		namePart := vLower
		if idx := strings.IndexAny(namePart, "=;"); idx > 0 {
			namePart = strings.TrimSpace(namePart[:idx])
		} else if idx := strings.Index(namePart, "="); idx > 0 {
			namePart = strings.TrimSpace(namePart[:idx])
		}
		for _, name := range authNames {
			if strings.Contains(namePart, name) {
				return true
			}
		}
	}
	return false
}
