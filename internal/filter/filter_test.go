package filter

import (
	"regexp"
	"testing"

	"github.com/Proviesec/PSFuzz/internal/config"
)

func TestPipelineAllow_StatusFilter(t *testing.T) {
	cfg := &config.Config{
		FilterStatus: []config.Range{{Min: 200, Max: 200}, {Min: 404, Max: 404}},
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200, Length: 100, Words: 5, Lines: 1}) {
		t.Error("expected allow 200")
	}
	if !p.Allow(Input{StatusCode: 404, Length: 100}) {
		t.Error("expected allow 404")
	}
	if p.Allow(Input{StatusCode: 500, Length: 100}) {
		t.Error("expected reject 500")
	}
}

func TestPipelineAllow_LengthFilter(t *testing.T) {
	cfg := &config.Config{
		FilterStatus: []config.Range{{Min: 200, Max: 200}},
		FilterLength: []config.Range{{Min: 10, Max: 100}},
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200, Length: 50}) {
		t.Error("expected allow length 50")
	}
	if p.Allow(Input{StatusCode: 200, Length: 5}) {
		t.Error("expected reject length 5")
	}
	if p.Allow(Input{StatusCode: 200, Length: 200}) {
		t.Error("expected reject length 200")
	}
}

func TestPipelineAllow_ContentType(t *testing.T) {
	cfg := &config.Config{
		FilterStatus:      []config.Range{{Min: 200, Max: 200}},
		FilterContentTypes: []string{"text/html", "application/json"},
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200, ContentType: "text/html"}) {
		t.Error("expected allow text/html")
	}
	if !p.Allow(Input{StatusCode: 200, ContentType: "text/html; charset=utf-8"}) {
		t.Error("expected allow text/html with charset")
	}
	if p.Allow(Input{StatusCode: 200, ContentType: "image/png"}) {
		t.Error("expected reject image/png")
	}
}

func TestPipelineAllow_BlockWords(t *testing.T) {
	cfg := &config.Config{
		FilterStatus: []config.Range{{Min: 200, Max: 200}},
		BlockWords:   []string{"error", "forbidden"},
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200, Body: "hello world"}) {
		t.Error("expected allow body without block words")
	}
	if p.Allow(Input{StatusCode: 200, Body: "page error occurred"}) {
		t.Error("expected reject body with block word")
	}
	if p.Allow(Input{StatusCode: 200, Body: "FORBIDDEN"}) {
		t.Error("expected reject body with block word (case insensitive)")
	}
}

func TestPipelineAllow_MinResponseSize(t *testing.T) {
	cfg := &config.Config{
		FilterStatus:    []config.Range{{Min: 200, Max: 200}},
		MinResponseSize: 100,
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200, Length: 150}) {
		t.Error("expected allow length >= 100")
	}
	if p.Allow(Input{StatusCode: 200, Length: 50}) {
		t.Error("expected reject length < 100")
	}
}

func TestPipelineAllow_FilterStatusNot(t *testing.T) {
	cfg := &config.Config{
		FilterStatus:    []config.Range{{Min: 200, Max: 500}},
		FilterStatusNot: []config.Range{{Min: 404, Max: 404}},
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200}) {
		t.Error("expected allow 200")
	}
	if p.Allow(Input{StatusCode: 404}) {
		t.Error("expected reject 404 via FilterStatusNot")
	}
}

func TestPipelineAllow_ShowStatusBypassesFilter(t *testing.T) {
	cfg := &config.Config{
		ShowStatus:  true,
		FilterStatus: []config.Range{{Min: 200, Max: 200}},
	}
	p := New(cfg)

	// With ShowStatus, even non-matching status is allowed (filter only for display)
	if !p.Allow(Input{StatusCode: 500, Length: 100}) {
		t.Error("expected allow 500 when ShowStatus true")
	}
}

func TestPipelineAllow_FilterMatchRegex(t *testing.T) {
	cfg := &config.Config{
		FilterStatus:     []config.Range{{Min: 200, Max: 200}},
		FilterMatchRegex: regexp.MustCompile(`(?i)admin`),
	}
	p := New(cfg)

	if !p.Allow(Input{StatusCode: 200, Body: "admin panel"}) {
		t.Error("expected allow body matching regex")
	}
	if p.Allow(Input{StatusCode: 200, Body: "user dashboard"}) {
		t.Error("expected reject body not matching regex")
	}
}

func TestPipelineAllow_DuplicateFilter(t *testing.T) {
	cfg := &config.Config{
		FilterStatus:       []config.Range{{Min: 200, Max: 200}},
		FilterDuplicates:   true,
		DuplicateThreshold: 2,
	}
	p := New(cfg)

	body := "same content here"
	in := Input{StatusCode: 200, Body: body}
	if !p.Allow(in) {
		t.Error("first occurrence should be allowed")
	}
	if !p.Allow(in) {
		t.Error("second occurrence should be allowed (threshold 2)")
	}
	if p.Allow(in) {
		t.Error("third occurrence should be rejected")
	}
}

func TestPipelineAllow_NearDuplicates(t *testing.T) {
	cfg := &config.Config{
		FilterStatus:            []config.Range{{Min: 200, Max: 200}},
		NearDuplicates:          true,
		DuplicateThreshold:      2,
		NearDuplicateLenBucket:   100,
		NearDuplicateWordBucket: 10,
		NearDuplicateLineBucket: 5,
	}
	p := New(cfg)

	in := Input{StatusCode: 200, Length: 150, Words: 12, Lines: 3, ContentType: "text/html", Body: "x"}
	if !p.Allow(in) {
		t.Error("first occurrence should be allowed")
	}
	if !p.Allow(in) {
		t.Error("second occurrence should be allowed (threshold 2)")
	}
	if p.Allow(in) {
		t.Error("third occurrence (same bucket) should be rejected")
	}
}
