// Package llm provides a shared layer for calling LLM backends (OpenAI, Ollama, Gemini).
// Use it from Explore AI, the AI response module, or any new module that needs to call an LLM.
//
// Example (new module or one-off call):
//
//	cfg := llm.Config{
//	    Provider:  llm.ProviderOpenAI,
//	    MaxTokens: 150,
//	}
//	key, err := llm.GetAPIKey(cfg.Provider)
//	if err != nil {
//	    return err // or report "skipped: " + err.Error()
//	}
//	content, err := llm.Call(ctx, cfg, key, []llm.Message{{Role: "user", Content: prompt}})
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Provider identifies the LLM backend.
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderOllama Provider = "ollama"
	ProviderGemini Provider = "gemini"
)

// ErrMissingAPIKey is returned by GetAPIKey when the provider requires an API key but none is set.
var ErrMissingAPIKey = errors.New("llm: API key required but not set")

// DefaultTimeout is the default timeout for API calls.
const DefaultTimeout = 20 * time.Second

// Default max tokens per provider when Config.MaxTokens <= 0.
const (
	DefaultMaxTokensOpenAI = 500
	DefaultMaxTokensOllama = 500
	DefaultMaxTokensGemini = 500
)

// Config configures an LLM call. Endpoint and Model can be empty to use built-in defaults.
type Config struct {
	Provider  Provider
	Endpoint  string // optional; e.g. http://localhost:11434 for Ollama
	Model     string // optional; e.g. gpt-4o-mini, llama3.1, gemini-1.5-flash
	MaxTokens int    // max tokens to generate; if <= 0, provider default is used
	Timeout   time.Duration
}

// Message is a single chat message (user or assistant).
type Message struct {
	Role    string // "user", "assistant", or "model" (Gemini)
	Content string
}

// GetAPIKey returns the API key for the given provider from the environment.
// For OpenAI: OPENAI_API_KEY; for Gemini: GEMINI_API_KEY or GOOGLE_API_KEY; for Ollama: OLLAMA_API_KEY (optional).
// Returns ErrMissingAPIKey when the provider requires a key and it is not set.
func GetAPIKey(p Provider) (string, error) {
	p = normalizeProvider(p)
	switch p {
	case ProviderOpenAI:
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return "", fmt.Errorf("%w (set OPENAI_API_KEY for provider %s)", ErrMissingAPIKey, p)
		}
		return key, nil
	case ProviderOllama:
		return os.Getenv("OLLAMA_API_KEY"), nil
	case ProviderGemini:
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			key = os.Getenv("GOOGLE_API_KEY")
		}
		if key == "" {
			return "", fmt.Errorf("%w (set GEMINI_API_KEY or GOOGLE_API_KEY for provider %s)", ErrMissingAPIKey, p)
		}
		return key, nil
	default:
		return "", fmt.Errorf("llm: unsupported provider %q (openai, ollama, gemini)", p)
	}
}

// Call sends the messages to the configured provider and returns the assistant's reply content.
// apiKey must be set for OpenAI and Gemini; for Ollama it can be empty.
func Call(ctx context.Context, cfg Config, apiKey string, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("llm: at least one message required")
	}
	p := normalizeProvider(cfg.Provider)
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		switch p {
		case ProviderOpenAI:
			maxTokens = DefaultMaxTokensOpenAI
		case ProviderOllama:
			maxTokens = DefaultMaxTokensOllama
		case ProviderGemini:
			maxTokens = DefaultMaxTokensGemini
		default:
			maxTokens = 500
		}
	}
	switch p {
	case ProviderOpenAI:
		return callOpenAI(ctx, cfg, apiKey, messages, maxTokens, timeout)
	case ProviderOllama:
		return callOllama(ctx, cfg, apiKey, messages, maxTokens, timeout)
	case ProviderGemini:
		return callGemini(ctx, cfg, apiKey, messages, maxTokens, timeout)
	default:
		return "", fmt.Errorf("llm: unsupported provider %q", cfg.Provider)
	}
}

func normalizeProvider(p Provider) Provider {
	s := strings.TrimSpace(strings.ToLower(string(p)))
	if s == "" {
		return ProviderOpenAI
	}
	return Provider(s)
}

// NormalizeProviderFromString normalizes a provider string (trim, lower; empty -> openai) and returns the Provider.
// Use this when reading provider from config (e.g. in modules) to avoid duplicating normalization logic.
func NormalizeProviderFromString(s string) Provider {
	return normalizeProvider(Provider(strings.TrimSpace(strings.ToLower(s))))
}

func defaultEndpoint(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "https://api.openai.com"
	case ProviderOllama:
		return "http://localhost:11434"
	case ProviderGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	default:
		return ""
	}
}

func defaultModel(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderOllama:
		return "llama3.1"
	case ProviderGemini:
		return "gemini-1.5-flash"
	default:
		return ""
	}
}

func resolveEndpoint(cfg Config) string {
	if e := strings.TrimSpace(cfg.Endpoint); e != "" {
		return strings.TrimSuffix(e, "/")
	}
	return defaultEndpoint(normalizeProvider(cfg.Provider))
}

func resolveModel(cfg Config) string {
	if m := strings.TrimSpace(cfg.Model); m != "" {
		return m
	}
	return defaultModel(normalizeProvider(cfg.Provider))
}

func callOpenAI(ctx context.Context, cfg Config, apiKey string, messages []Message, maxTokens int, timeout time.Duration) (string, error) {
	base := resolveEndpoint(cfg)
	model := resolveModel(cfg)
	msgs := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	reqBody := map[string]any{
		"model":      model,
		"messages":   msgs,
		"max_tokens": maxTokens,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("openai decode: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	return out.Choices[0].Message.Content, nil
}

func callOllama(ctx context.Context, cfg Config, apiKey string, messages []Message, maxTokens int, timeout time.Duration) (string, error) {
	base := resolveEndpoint(cfg)
	model := resolveModel(cfg)
	msgs := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	reqBody := map[string]any{
		"model":    model,
		"messages": msgs,
		"stream":   false,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ollama decode: %w", err)
	}
	return out.Message.Content, nil
}

func callGemini(ctx context.Context, cfg Config, apiKey string, messages []Message, maxTokens int, timeout time.Duration) (string, error) {
	base := resolveEndpoint(cfg)
	model := resolveModel(cfg)
	contents := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": m.Content}},
		})
	}
	reqBody := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": maxTokens,
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/models/"+model+":generateContent", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("gemini decode: %w", err)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

// IsMissingKeyError returns true if err indicates the API key was not set.
func IsMissingKeyError(err error) bool {
	return errors.Is(err, ErrMissingAPIKey)
}
