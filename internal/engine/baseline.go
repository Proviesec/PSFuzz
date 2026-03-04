package engine

import (
	"context"
	"net/url"
	"strings"

	"github.com/Proviesec/PSFuzz/internal/httpx"
)

type baselineFingerprint struct {
	Status int
	Length int
	Words  int
	Lines  int
	Valid  bool
}

func matchesBaseline(b baselineFingerprint, status, length, words, lines int) bool {
	return b.Valid && b.Status == status && b.Length == length && b.Words == words && b.Lines == lines
}

func (e *Engine) ensureBaseline(ctx context.Context, host string, task Task, st *runState) baselineFingerprint {
	if !e.cfg.AutoWildcard || host == "" {
		return baselineFingerprint{}
	}
	st.mu.Lock()
	if b, ok := st.baselines[host]; ok {
		st.mu.Unlock()
		return b
	}
	st.baselines[host] = baselineFingerprint{Valid: false}
	st.mu.Unlock()

	randWord := e.randomString(12)
	randValues := randomizeValues(task.Values, randWord)
	baseURL := baselineURL(task.URL, randWord)
	baseTask := Task{
		URL:     baseURL,
		Values:  randValues,
		Method:  task.Method,
		Headers: task.Headers,
	}
	method, body, headers := e.requestPartsFor(baseTask)
	resp, err := e.client.Do(ctx, httpx.RequestSpec{
		URL:     baseURL,
		Method:  method,
		Body:    body,
		Headers: headers,
	})
	st.totalReq.Add(1)
	if err != nil {
		return baselineFingerprint{}
	}
	defer resp.Body.Close()
	data, _, err := readBodyWithLimit(resp.Body, e.cfg.MaxResponseSize)
	if err != nil {
		return baselineFingerprint{}
	}
	text := string(data)
	words := len(strings.Fields(text))
	lines := countLines(text)
	b := baselineFingerprint{
		Status: resp.StatusCode,
		Length: len(data),
		Words:  words,
		Lines:  lines,
		Valid:  true,
	}
	st.mu.Lock()
	st.baselines[host] = b
	st.mu.Unlock()
	return b
}

func randomizeValues(values map[string]string, word string) map[string]string {
	out := make(map[string]string, len(values)+1)
	if len(values) == 0 {
		out["FUZZ"] = word
		return out
	}
	for k := range values {
		out[k] = word
	}
	if _, ok := out["FUZZ"]; !ok {
		out["FUZZ"] = word
	}
	return out
}

func baselineURL(rawURL, word string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		if strings.HasSuffix(rawURL, "/") {
			return rawURL + word
		}
		return rawURL + "/" + word
	}
	path := u.Path
	if strings.HasSuffix(path, "/") {
		path = path + word
	} else {
		path = path + "/" + word
	}
	u.Path = path
	u.RawQuery = ""
	return u.String()
}
