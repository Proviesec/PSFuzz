package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
)

func TestNewClient_MinimalConfig(t *testing.T) {
	cfg := &config.Config{
		Timeout:         10 * time.Second,
		RequestMethod:  "GET",
		RequestHeaders: map[string]string{},
		RequestCookies: map[string]string{},
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.http == nil {
		t.Fatal("http client not set")
	}
}

func TestValidateTarget_SafeModeBlocksLoopback(t *testing.T) {
	cfg := &config.Config{
		Timeout:  5 * time.Second,
		SafeMode: true,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = c.validateTarget(context.Background(), "http://127.0.0.1/")
	if err == nil {
		t.Error("expected error for 127.0.0.1 in safe mode")
	}
	err = c.validateTarget(context.Background(), "http://[::1]/")
	if err == nil {
		t.Error("expected error for ::1 in safe mode")
	}
	err = c.validateTarget(context.Background(), "http://localhost/")
	if err == nil {
		t.Error("expected error for localhost in safe mode")
	}
}

func TestValidateTarget_SafeModeAllowsPublic(t *testing.T) {
	cfg := &config.Config{
		Timeout:  5 * time.Second,
		SafeMode: true,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = c.validateTarget(context.Background(), "https://example.com/path")
	if err != nil {
		t.Errorf("expected nil for public host, got %v", err)
	}
}

func TestValidateTarget_NoSafeModeAllowsLoopback(t *testing.T) {
	cfg := &config.Config{
		Timeout:  5 * time.Second,
		SafeMode: false,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = c.validateTarget(context.Background(), "http://127.0.0.1/")
	if err != nil {
		t.Errorf("expected nil when safe mode off, got %v", err)
	}
}

func TestValidateTarget_AllowedHostsScope(t *testing.T) {
	cfg := &config.Config{
		Timeout:      5 * time.Second,
		SafeMode:    false,
		AllowedHosts: []string{"allowed.example.com", "other.example.com"},
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = c.validateTarget(context.Background(), "https://allowed.example.com/")
	if err != nil {
		t.Errorf("expected nil for allowed host, got %v", err)
	}
	err = c.validateTarget(context.Background(), "https://notallowed.example.com/")
	if err == nil {
		t.Error("expected error for host outside scope")
	}
}

func TestValidateTarget_MissingHost(t *testing.T) {
	cfg := &config.Config{Timeout: 5 * time.Second, SafeMode: false}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = c.validateTarget(context.Background(), "http:///path")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestDo_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	cfg := &config.Config{
		Timeout:         5 * time.Second,
		RequestMethod:   "GET",
		RequestHeaders:  map[string]string{},
		RequestCookies:  map[string]string{},
		SafeMode:        false,
		RetryCount:      0,
		RetryBackoff:    time.Millisecond,
		BypassTooManyRequests: false,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Do(context.Background(), RequestSpec{URL: ts.URL, Method: http.MethodGet})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDo_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Config{
		Timeout:         10 * time.Second,
		RequestMethod:  "GET",
		RequestHeaders: map[string]string{},
		RequestCookies: map[string]string{},
		SafeMode:       false,
		RetryCount:     0,
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = c.Do(ctx, RequestSpec{URL: ts.URL, Method: http.MethodGet})
	if err == nil {
		t.Error("expected error when context cancelled")
	}
	// err may be context.Canceled or wrapped by http client
	if err != nil && ctx.Err() != context.Canceled {
		t.Errorf("expected cancellation, ctx.Err()=%v err=%v", ctx.Err(), err)
	}
}

func TestRandomUserAgent_NilRand(t *testing.T) {
	ua := randomUserAgent(nil)
	if ua == "" {
		t.Error("expected default user agent")
	}
	if ua != "PSFuzz/1.0.0" {
		t.Errorf("expected PSFuzz/1.0.0, got %s", ua)
	}
}
