package modules

import (
	"context"
	"regexp"
	"strings"
)

// SecretsAnalyzer scans response body and headers for common secret/key patterns.
// Reports potential leaks: AWS keys, JWTs, GitHub/Slack tokens, password= in response, etc.
// Findings are type labels only; no secret values are stored.
type SecretsAnalyzer struct{}

func init() {
	Register("secrets", func(*Config) Analyzer { return SecretsAnalyzer{} })
}

func (SecretsAnalyzer) Name() string { return "secrets" }

// Well-known patterns (high signal, lower false positives).
var (
	awsAccessKeyRE   = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	jwtRE             = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	githubTokenRE     = regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`)
	githubOAuthRE     = regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`)
	slackTokenRE      = regexp.MustCompile(`xox[baprs]-[a-zA-Z0-9-]{10,}`)
	passwordInBodyRE  = regexp.MustCompile(`(?i)(["']?password["']?\s*[:=]\s*["']?)[^"'\s&]+`)
	genericSecretRE  = regexp.MustCompile(`(?i)(["']?(?:api[_-]?key|secret|apikey)["']?\s*[:=]\s*["']?)[a-zA-Z0-9_\-]{20,}`)
)

func (SecretsAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}

	var findings []string
	body := in.Body

	// AWS Access Key ID
	if awsAccessKeyRE.MatchString(body) {
		findings = append(findings, "aws_access_key")
	}
	// JWT (header.payload.signature)
	if jwtRE.MatchString(body) {
		findings = append(findings, "jwt")
	}
	// GitHub tokens
	if githubTokenRE.MatchString(body) || githubOAuthRE.MatchString(body) {
		findings = append(findings, "github_token")
	}
	// Slack
	if slackTokenRE.MatchString(body) {
		findings = append(findings, "slack_token")
	}
	// password= or "password":" in response (potential credential leak)
	if passwordInBodyRE.MatchString(body) {
		findings = append(findings, "password_in_response")
	}
	// Generic api_key/secret with long value
	if genericSecretRE.MatchString(body) {
		findings = append(findings, "generic_secret")
	}

	// Headers: Bearer token (we only report presence, not the value)
	for k, v := range in.Headers {
		if strings.EqualFold(k, "Authorization") && strings.HasPrefix(strings.TrimSpace(v), "Bearer ") {
			findings = append(findings, "bearer_in_response")
			break
		}
	}

	if len(findings) == 0 {
		return Output{Data: nil}, nil
	}

	// Deduplicate by type (same type can match multiple times)
	seen := make(map[string]bool)
	var unique []string
	for _, f := range findings {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}

	return Output{
		Data: map[string]any{
			"findings": unique,
			"count":    len(unique),
		},
	}, nil
}
