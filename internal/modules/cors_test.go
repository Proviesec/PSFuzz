package modules

import (
	"context"
	"testing"
)

func TestCORSAnalyzer_NoCORSHeaders(t *testing.T) {
	in := Input{StatusCode: 200, Headers: map[string]string{"Content-Type": "text/html"}, Body: ""}
	out, err := CORSAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data != nil {
		t.Errorf("expected nil when no CORS headers, got %v", out.Data)
	}
}

func TestCORSAnalyzer_PermissiveOrigin(t *testing.T) {
	in := Input{
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
		},
		Body: "",
	}
	out, err := CORSAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected CORS data")
	}
	if v, _ := out.Data["potentially_permissive"].(bool); !v {
		t.Error("expected potentially_permissive true for * with credentials")
	}
}
