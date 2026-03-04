package modules

import (
	"context"
	"testing"
)

func TestSecretsAnalyzer_NoSecrets(t *testing.T) {
	in := Input{StatusCode: 200, Body: "Hello world. No secrets here."}
	out, err := SecretsAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data != nil {
		t.Errorf("expected nil when no secrets, got %v", out.Data)
	}
}

func TestSecretsAnalyzer_AWSKey(t *testing.T) {
	// Fake pattern only - not a real key
	in := Input{StatusCode: 200, Body: `{"config":{"access_key":"AKIAIOSFODNN7EXAMPLE"}}`}
	out, err := SecretsAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when AWS key pattern present")
	}
	findings, ok := out.Data["findings"].([]string)
	if !ok || len(findings) == 0 {
		t.Fatalf("expected findings, got %v", out.Data["findings"])
	}
	found := false
	for _, f := range findings {
		if f == "aws_access_key" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected aws_access_key in findings, got %v", findings)
	}
}

func TestSecretsAnalyzer_JWT(t *testing.T) {
	// Fake JWT shape (header.payload.sig) - not a real token
	in := Input{StatusCode: 200, Body: `token=eyJhbGciOiJIUzI1.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`}
	out, err := SecretsAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when JWT present")
	}
	findings, _ := out.Data["findings"].([]string)
	found := false
	for _, f := range findings {
		if f == "jwt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected jwt in findings, got %v", findings)
	}
}

func TestSecretsAnalyzer_PasswordInResponse(t *testing.T) {
	in := Input{StatusCode: 200, Body: `{"user":"admin","password":"super_secret_123"}`}
	out, err := SecretsAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when password in body")
	}
	findings, _ := out.Data["findings"].([]string)
	found := false
	for _, f := range findings {
		if f == "password_in_response" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected password_in_response in findings, got %v", findings)
	}
}

func TestSecretsAnalyzer_GenericSecret(t *testing.T) {
	in := Input{StatusCode: 200, Body: `"api_key":"abcdefghijklmnopqrstuvwxyz123456"`}
	out, err := SecretsAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when api_key pattern present")
	}
	findings, _ := out.Data["findings"].([]string)
	found := false
	for _, f := range findings {
		if f == "generic_secret" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected generic_secret in findings, got %v", findings)
	}
}

func TestSecretsAnalyzer_Deduplicates(t *testing.T) {
	in := Input{StatusCode: 200, Body: `key1=AKIAIOSFODNN7EXAMPLE key2=AKIAIOSFODNN7EXAMPLE`}
	out, err := SecretsAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data")
	}
	findings, _ := out.Data["findings"].([]string)
	count := 0
	for _, f := range findings {
		if f == "aws_access_key" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected single aws_access_key entry (deduped), got %d", count)
	}
	if c, ok := out.Data["count"].(int); !ok || c != 1 {
		t.Errorf("expected count=1, got %v", out.Data["count"])
	}
}

func TestEnabled_Secrets(t *testing.T) {
	mc := &Config{Modules: []string{"secrets", "fingerprint"}}
	out := Enabled(mc)
	if len(out) != 2 {
		t.Errorf("expected 2 analyzers, got %d", len(out))
	}
	names := make(map[string]bool)
	for _, a := range out {
		names[a.Name()] = true
	}
	if !names["secrets"] || !names["fingerprint"] {
		t.Errorf("expected secrets and fingerprint, got %v", names)
	}
}
