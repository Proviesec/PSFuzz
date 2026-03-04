package modules

import (
	"context"
	"testing"
)

func TestHeadersAnalyzer_AllMissing(t *testing.T) {
	in := Input{StatusCode: 200, Headers: map[string]string{"Content-Type": "text/html"}, Body: ""}
	out, err := HeadersAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected headers data")
	}
	missing, ok := out.Data["missing_headers"].([]string)
	if !ok || len(missing) < 4 {
		t.Errorf("expected at least 4 missing headers, got %v", out.Data["missing_headers"])
	}
	if _, has := out.Data["issues"]; !has {
		t.Error("expected issues when headers missing")
	}
}

func TestHeadersAnalyzer_AllPresent(t *testing.T) {
	in := Input{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Security-Policy":   "default-src 'self'",
			"X-Frame-Options":           "DENY",
			"Strict-Transport-Security": "max-age=31536000",
			"X-Content-Type-Options":    "nosniff",
			"Set-Cookie":                "sid=abc; Secure; HttpOnly",
		},
		Body: "",
	}
	out, err := HeadersAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected headers data")
	}
	missing, _ := out.Data["missing_headers"].([]string)
	if len(missing) != 0 {
		t.Errorf("expected no missing headers, got %v", missing)
	}
	if v, _ := out.Data["cookie_has_secure"].(bool); !v {
		t.Error("expected cookie_has_secure true")
	}
	if v, _ := out.Data["cookie_has_httponly"].(bool); !v {
		t.Error("expected cookie_has_httponly true")
	}
}

func TestHeadersAnalyzer_CookieWithoutSecure(t *testing.T) {
	in := Input{
		StatusCode: 200,
		Headers:    map[string]string{"Set-Cookie": "session=xyz; HttpOnly; Path=/"},
		Body:       "",
	}
	out, err := HeadersAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected headers data")
	}
	if v, _ := out.Data["cookie_has_secure"].(bool); v {
		t.Error("expected cookie_has_secure false when Secure not in Set-Cookie")
	}
	if v, _ := out.Data["cookie_has_httponly"].(bool); !v {
		t.Error("expected cookie_has_httponly true")
	}
}
