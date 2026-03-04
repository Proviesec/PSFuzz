package modules

import (
	"context"
	"testing"
)

func TestAuthAnalyzer_NoHints(t *testing.T) {
	in := Input{StatusCode: 200, Body: "<html><body>Hello world</body></html>"}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data != nil {
		t.Errorf("expected nil when no auth hints, got %v", out.Data)
	}
}

func TestAuthAnalyzer_LoginForm(t *testing.T) {
	in := Input{
		StatusCode: 200,
		Body:       `<form action="/login" method="post"><input type="text" name="user"/><input type="password" name="password"/><button>Sign in</button></form>`,
	}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when login form present")
	}
	hints, _ := out.Data["hints"].([]string)
	found := false
	for _, h := range hints {
		if h == "login_form" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected login_form in hints, got %v", hints)
	}
}

func TestAuthAnalyzer_Status401(t *testing.T) {
	in := Input{StatusCode: 401, Body: "Unauthorized"}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data for 401")
	}
	hints, _ := out.Data["hints"].([]string)
	found := false
	for _, h := range hints {
		if h == "status_401" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected status_401 in hints, got %v", hints)
	}
}

func TestAuthAnalyzer_RedirectToLogin(t *testing.T) {
	in := Input{
		StatusCode: 302,
		Headers:    map[string]string{"Location": "https://example.com/login?next=/admin"},
		Body:       "",
	}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data for 302 to login")
	}
	hints, _ := out.Data["hints"].([]string)
	found := false
	for _, h := range hints {
		if h == "redirect_to_login" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected redirect_to_login in hints, got %v", hints)
	}
}

func TestAuthAnalyzer_SessionExpired(t *testing.T) {
	in := Input{StatusCode: 200, Body: "Your session has expired. Please log in again."}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when session expired text present")
	}
	hints, _ := out.Data["hints"].([]string)
	found := false
	for _, h := range hints {
		if h == "session_or_login_required" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected session_or_login_required in hints, got %v", hints)
	}
}

func TestAuthAnalyzer_LogoutPresent(t *testing.T) {
	in := Input{StatusCode: 200, Body: `<a href="/logout">Log out</a>`}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when logout present")
	}
	hints, _ := out.Data["hints"].([]string)
	found := false
	for _, h := range hints {
		if h == "logout_present" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected logout_present in hints, got %v", hints)
	}
}

func TestAuthAnalyzer_AuthCookieSet(t *testing.T) {
	in := Input{
		StatusCode: 200,
		Headers:    map[string]string{"Set-Cookie": "session_id=abc123; Path=/; HttpOnly"},
		Body:       "",
	}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data when auth-like cookie set")
	}
	hints, _ := out.Data["hints"].([]string)
	found := false
	for _, h := range hints {
		if h == "auth_cookie_set" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected auth_cookie_set in hints, got %v", hints)
	}
}

func TestAuthAnalyzer_Deduplicates(t *testing.T) {
	in := Input{
		StatusCode: 401,
		Body:       "Unauthorized. Please log in. Session expired.",
	}
	out, err := AuthAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected data")
	}
	if c, ok := out.Data["count"].(int); !ok || c < 1 {
		t.Errorf("expected count >= 1, got %v", out.Data["count"])
	}
}

func TestEnabled_Auth(t *testing.T) {
	mc := &Config{Modules: []string{"auth", "fingerprint"}}
	out := Enabled(mc)
	if len(out) != 2 {
		t.Errorf("expected 2 analyzers, got %d", len(out))
	}
	names := make(map[string]bool)
	for _, a := range out {
		names[a.Name()] = true
	}
	if !names["auth"] || !names["fingerprint"] {
		t.Errorf("expected auth and fingerprint, got %v", names)
	}
}
