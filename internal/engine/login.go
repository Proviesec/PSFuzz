package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Proviesec/PSFuzz/internal/config"
	"github.com/Proviesec/PSFuzz/internal/httpx"
)

// doLogin performs a single login request to cfg.LoginURL, then merges any
// Set-Cookie from the response into cfg.RequestCookies so subsequent fuzz
// requests use the session. Called at the start of Run() when LoginURL is set.
func doLogin(ctx context.Context, client *httpx.Client, cfg *config.Config) error {
	if cfg.LoginURL == "" {
		return nil
	}
	if _, err := url.ParseRequestURI(cfg.LoginURL); err != nil {
		return err
	}
	method := strings.TrimSpace(strings.ToUpper(cfg.LoginMethod))
	if method == "" {
		method = http.MethodPost
	}
	if method != http.MethodGet && method != http.MethodPost {
		method = http.MethodPost
	}

	var body string
	headers := make(map[string]string)
	if cfg.LoginBody != "" {
		body = cfg.LoginBody
		ct := cfg.LoginContentType
		if ct == "" {
			ct = "application/x-www-form-urlencoded"
		}
		headers["Content-Type"] = ct
	} else if cfg.LoginUser != "" || cfg.LoginPass != "" {
		form := url.Values{}
		form.Set("username", cfg.LoginUser)
		form.Set("password", cfg.LoginPass)
		body = form.Encode()
		ct := cfg.LoginContentType
		if ct == "" {
			ct = "application/x-www-form-urlencoded"
		}
		headers["Content-Type"] = ct
	}

	spec := httpx.RequestSpec{
		URL:     cfg.LoginURL,
		Method:  method,
		Body:    body,
		Headers: headers,
	}
	result, err := client.Do(ctx, spec)
	if err != nil {
		return fmt.Errorf("login request to %s: %w", cfg.LoginURL, err)
	}
	resp := result.Resp
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// Check for authentication failure
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("login failed: received status %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("login request failed: received status %d", resp.StatusCode)
	}

	if cfg.RequestCookies == nil {
		cfg.RequestCookies = make(map[string]string)
	}
	for _, c := range resp.Cookies() {
		cfg.RequestCookies[c.Name] = c.Value
	}
	return nil
}
