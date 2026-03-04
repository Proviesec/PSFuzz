package encoder

import (
	"encoding/base64"
	"net/url"
	"strings"
)

// Chain applies a list of encoder names to value in order.
// Unknown encoder names are skipped. Returns value unchanged if no encoders or all unknown.
// All built-in encoders are infallible; the returned error is reserved for future use (e.g. strict mode).
func Chain(value string, names []string) (string, error) {
	out := value
	for _, name := range names {
		n := strings.TrimSpace(strings.ToLower(name))
		if n == "" {
			continue
		}
		fn, ok := registry[n]
		if !ok {
			continue
		}
		out = fn(out)
	}
	return out, nil
}

// ApplyToMap returns a new map with each value run through its encoder chain.
// encoders maps keyword -> list of encoder names (e.g. "FUZZ" -> ["urlencode"]).
// Keywords not in encoders are copied as-is.
func ApplyToMap(encoders map[string][]string, values map[string]string) map[string]string {
	if len(encoders) == 0 || len(values) == 0 {
		return values
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		chain, ok := encoders[k]
		if !ok || len(chain) == 0 {
			out[k] = v
			continue
		}
		encoded, _ := Chain(v, chain)
		out[k] = encoded
	}
	return out
}

type fn func(string) string

var registry = map[string]fn{
	"urlencode":        urlEncode,
	"doubleurlencode":   doubleURLEncode,
	"base64encode":      base64Encode,
	"base64decode":      base64Decode,
	"htmlencode":        htmlEncode,
	"htmldoubleencode":  htmlDoubleEncode,
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}

func doubleURLEncode(s string) string {
	return url.QueryEscape(url.QueryEscape(s))
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func base64Decode(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return string(b)
}

func htmlEncode(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func htmlDoubleEncode(s string) string {
	return htmlEncode(htmlEncode(s))
}

// Registered returns the list of registered encoder names (e.g. for help).
func Registered() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
