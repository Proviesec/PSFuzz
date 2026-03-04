package modules

import (
	"context"
	"strings"
)

// CORSAnalyzer reads CORS-related headers from the response and reports findings.
// It does not send a separate request with Origin; it only inspects the current response.
type CORSAnalyzer struct{}

func init() {
	Register("cors", func(*Config) Analyzer { return CORSAnalyzer{} })
}

func (CORSAnalyzer) Name() string { return "cors" }

func (CORSAnalyzer) Analyze(ctx context.Context, in Input) (Output, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		default:
		}
	}
	// Normalize header keys (HTTP headers are case-insensitive)
	get := func(name string) string {
		for k, v := range in.Headers {
			if strings.EqualFold(k, name) {
				return v
			}
		}
		return ""
	}
	acao := get("Access-Control-Allow-Origin")
	acac := get("Access-Control-Allow-Credentials")
	acam := get("Access-Control-Allow-Methods")
	acah := get("Access-Control-Allow-Headers")

	if acao == "" && acac == "" && acam == "" && acah == "" {
		return Output{Data: nil}, nil
	}

	data := map[string]any{
		"access_control_allow_origin":      acao,
		"access_control_allow_credentials": acac,
		"access_control_allow_methods":    acam,
		"access_control_allow_headers":    acah,
	}
	// Simple risk hint
	risky := acao == "*" || (acao != "" && acac == "true")
	data["potentially_permissive"] = risky
	return Output{Data: data}, nil
}
