package llm

import (
	"errors"
	"testing"
)

func TestGetAPIKey_MissingOpenAI(t *testing.T) {
	// Clear OPENAI_API_KEY so we test the "missing" path
	t.Setenv("OPENAI_API_KEY", "")
	key, err := GetAPIKey(ProviderOpenAI)
	if err == nil {
		t.Fatal("expected error when OPENAI_API_KEY not set")
	}
	if !IsMissingKeyError(err) {
		t.Errorf("expected ErrMissingAPIKey, got %v", err)
	}
	if key != "" {
		t.Errorf("expected empty key, got %q", key)
	}
}

func TestGetAPIKey_UnsupportedProvider(t *testing.T) {
	_, err := GetAPIKey(Provider("invalid"))
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if errors.Is(err, ErrMissingAPIKey) {
		t.Errorf("expected non-MissingAPIKey error for invalid provider, got %v", err)
	}
}

func TestNormalizeProvider(t *testing.T) {
	if normalizeProvider("") != ProviderOpenAI {
		t.Error("empty provider should default to openai")
	}
	if normalizeProvider("  OPENAI  ") != ProviderOpenAI {
		t.Error("openai should normalize")
	}
	if normalizeProvider("ollama") != ProviderOllama {
		t.Error("ollama should normalize")
	}
}
