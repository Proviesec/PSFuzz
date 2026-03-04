package modules

import (
	"context"
)

// Input holds the request/response data passed to analyzers.
// Kept independent of engine so modules do not depend on engine or httpx.
type Input struct {
	URL         string
	Method      string
	StatusCode  int
	Headers     map[string]string
	Body        string
	ContentType string
	Length      int
	Words       int
	Lines       int
}

// Output is the result of one analyzer. Data is module-specific (e.g. "technologies": ["nginx","php"]).
type Output struct {
	Data map[string]any
}

// Analyzer is the interface for response-analysis modules (fingerprint, CORS, AI, etc.).
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, in Input) (Output, error)
}

// Run runs all analyzers and merges their outputs into a single map keyed by module name.
// Skips errors (failed module does not block others).
func Run(ctx context.Context, analyzers []Analyzer, in Input) map[string]map[string]any {
	if len(analyzers) == 0 {
		return nil
	}
	out := make(map[string]map[string]any)
	for _, a := range analyzers {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return out
			default:
			}
		}
		res, err := a.Analyze(ctx, in)
		if err != nil || res.Data == nil {
			continue
		}
		out[a.Name()] = res.Data
	}
	return out
}
