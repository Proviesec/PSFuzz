package modules

import (
	"context"
	"testing"
)

func TestFingerprintAnalyzer_HeadersCaseInsensitive(t *testing.T) {
	// Go http.Header canonicalizes to "Server", not "server"
	in := Input{
		StatusCode: 200,
		Headers:    map[string]string{"Server": "nginx/1.20", "X-Powered-By": "PHP/8.0"},
		Body:       "",
	}
	out, err := FingerprintAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected fingerprint data")
	}
	tech, _ := out.Data["technologies"].([]string)
	found := false
	for _, name := range tech {
		if name == "nginx" || name == "php" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected nginx or php from headers, got %v", tech)
	}
}
