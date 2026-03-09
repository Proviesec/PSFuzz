package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Proviesec/PSFuzz/internal/llm"
)

const (
	aiMaxBodyLen       = 3000
	aiTimeout          = 15 * time.Second
	aiDefaultMaxTokens = 150
	aiDefaultPrompt    = "You are a cyber security expert. Context: HTTP status {{status}}, method {{method}}{{url_line}}. Do you see anything unusual on this page from a security perspective? (e.g. debug info, admin/login hints, error messages with internal details, suspicious patterns, or nothing notable.) Reply in one short sentence.\n\n--- Response Body ---\n\n{{body}}"
)

// AIAnalyzer sends a truncated response body to an AI backend (openai, ollama, gemini) and returns a short verdict.
// Uses the shared llm package; provider and API key from config (openai: OPENAI_API_KEY; gemini: GEMINI_API_KEY or GOOGLE_API_KEY; ollama: usually no key).
// When the API key is missing or the API call fails, the module returns structured error info in Data ("skipped" or "error"/"message") so reports show the reason.
type AIAnalyzer struct {
	Prompt    string
	Provider  string
	Endpoint  string
	Model     string
	MaxTokens int
}

func init() {
	Register("ai", func(c *Config) Analyzer {
		return AIAnalyzer{
			Prompt:    c.AIPrompt,
			Provider:  c.AIProvider,
			Endpoint:  c.AIEndpoint,
			Model:     c.AIModel,
			MaxTokens: c.AIMaxTokens,
		}
	})
}

func (AIAnalyzer) Name() string { return "ai" }

func (a AIAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}
	p := llm.NormalizeProviderFromString(a.Provider)
	apiKey, err := llm.GetAPIKey(p)
	if err != nil {
		// Report why the module was skipped (e.g. OPENAI_API_KEY not set) so it appears in module_data / report
		return Output{Data: map[string]any{"skipped": err.Error()}}, nil
	}
	if p != llm.ProviderOllama && apiKey == "" {
		return Output{Data: map[string]any{"skipped": "API key required but not set"}}, nil
	}
	body := in.Body
	// Truncate at rune boundary to avoid invalid UTF-8
	if len(body) > aiMaxBodyLen*4 {
		body = body[:aiMaxBodyLen*4]
	}
	if runes := []rune(body); len(runes) > aiMaxBodyLen {
		body = string(runes[:aiMaxBodyLen]) + "...[truncated]"
	}
	tpl := a.Prompt
	if tpl == "" {
		tpl = aiDefaultPrompt
	}
	urlLine := ""
	if in.URL != "" {
		urlLine = ", URL: " + in.URL
	}
	prompt := strings.NewReplacer(
		"{{status}}", fmt.Sprintf("%d", in.StatusCode),
		"{{method}}", in.Method,
		"{{url}}", in.URL,
		"{{url_line}}", urlLine,
		"{{body}}", body,
	).Replace(tpl)

	maxTokens := a.MaxTokens
	if maxTokens <= 0 {
		maxTokens = aiDefaultMaxTokens
	}
	cfg := llm.Config{
		Provider:  p,
		Endpoint:  a.Endpoint,
		Model:     a.Model,
		MaxTokens: maxTokens,
		Timeout:   aiTimeout,
	}
	content, err := llm.Call(ctx, cfg, apiKey, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return Output{Data: map[string]any{"error": "api", "message": err.Error()}}, nil
	}
	verdict := strings.TrimSpace(content)
	if verdict == "" {
		return Output{Data: nil}, nil
	}
	return Output{Data: map[string]any{"verdict": verdict}}, nil
}
