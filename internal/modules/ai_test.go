package modules

import (
	"context"
	"testing"
)

func TestAIAnalyzer_NoAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "") // Explicitly unset for this test
	// Without OPENAI_API_KEY, AI module returns skipped reason in Data (for report visibility)
	in := Input{StatusCode: 200, Body: "test"}
	out, err := AIAnalyzer{}.Analyze(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("expected non-nil Data with skip reason when no API key")
	}
	if _, ok := out.Data["skipped"]; !ok {
		t.Errorf("expected Data to contain 'skipped' key when no API key, got %v", out.Data)
	}
}
