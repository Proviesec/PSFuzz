# LLM layer

Shared package for calling LLM backends (OpenAI, Ollama, Gemini). Used by **Explore AI** and the **AI response module** (`-modules ai`). Any new module that needs to call an LLM can use this package instead of duplicating API logic.

## API

- **`llm.Provider`** – `openai`, `ollama`, `gemini`
- **`llm.Config`** – Provider, Endpoint, Model, MaxTokens, Timeout (optional overrides)
- **`llm.Message`** – Role + Content (for chat)
- **`llm.GetAPIKey(provider)`** – Resolves API key from env; returns `ErrMissingAPIKey` when required but not set
- **`llm.Call(ctx, cfg, apiKey, messages)`** – Sends messages and returns the assistant’s reply text
- **`llm.IsMissingKeyError(err)`** – True if the error is “API key not set”

## Example: new module that uses the LLM

Copy-paste skeleton. Add your config fields to `modules.Config` and wire flags in `internal/config` as needed.

```go
package modules

import (
	"context"
	"strings"

	"github.com/Proviesec/PSFuzz/internal/llm"
)

type MyLLMAnalyzer struct {
	Prompt    string
	Provider  string
	Endpoint  string
	Model     string
	MaxTokens int
}

func init() {
	Register("myllm", func(c *Config) Analyzer {
		return MyLLMAnalyzer{
			Prompt:    c.MyLLMPrompt,
			Provider:  c.MyLLMProvider,
			Endpoint:  c.MyLLMEndpoint,
			Model:     c.MyLLMModel,
			MaxTokens: c.MyLLMMaxTokens,
		}
	})
}

func (MyLLMAnalyzer) Name() string { return "myllm" }

func (a MyLLMAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	p := llm.NormalizeProviderFromString(a.Provider)
	apiKey, err := llm.GetAPIKey(p)
	if err != nil {
		return Output{Data: map[string]any{"skipped": err.Error()}}, nil
	}
	maxTokens := a.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 200
	}
	cfg := llm.Config{
		Provider:  p,
		Endpoint:  a.Endpoint,
		Model:     a.Model,
		MaxTokens: maxTokens,
	}
	content, err := llm.Call(ctx, cfg, apiKey, []llm.Message{
		{Role: "user", Content: a.Prompt + "\n\n" + in.Body},
	})
	if err != nil {
		return Output{Data: map[string]any{"error": "api", "message": err.Error()}}, nil
	}
	return Output{Data: map[string]any{"result": strings.TrimSpace(content)}}, nil
}
```

Then in `internal/modules/config.go` add your fields (e.g. `MyLLMPrompt`, `MyLLMProvider`, …), and in `internal/config` add the corresponding CLI flags and file config so they are applied to `cfg.ModuleConfig`.

See **`internal/modules/ai.go`** for the full AI module implementation.
