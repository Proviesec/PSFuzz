package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRawRequest_Valid(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "req.txt")
	content := "POST /api HTTP/1.1\nHost: example.com\nContent-Type: application/json\n\n{\"a\":1}"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	req, err := parseRawRequest(f)
	if err != nil {
		t.Fatalf("parseRawRequest: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("Method=%q, want POST", req.Method)
	}
	if req.Path != "/api" {
		t.Errorf("Path=%q, want /api", req.Path)
	}
	if req.Headers["Host"] != "example.com" {
		t.Errorf("Headers[Host]=%q, want example.com", req.Headers["Host"])
	}
	if req.Body != "{\"a\":1}" {
		t.Errorf("Body=%q, want {\"a\":1}", req.Body)
	}
}

func TestParseRawRequest_CRLFBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "req.txt")
	content := "GET / HTTP/1.1\r\nHost: x\r\n\r\nbody"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	req, err := parseRawRequest(f)
	if err != nil {
		t.Fatalf("parseRawRequest: %v", err)
	}
	if req.Method != "GET" || req.Body != "body" {
		t.Errorf("Method=%q Body=%q", req.Method, req.Body)
	}
}

func TestParseRawRequest_InvalidRequestLine(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "req.txt")
	if err := os.WriteFile(f, []byte("GET\nHost: x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := parseRawRequest(f)
	if err == nil {
		t.Fatal("expected error for single-field request line")
	}
}

func TestParseRawRequest_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(f, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := parseRawRequest(f)
	if err == nil {
		t.Fatal("expected error for empty request file")
	}
}
