package encoder

import (
	"testing"
)

func TestChain(t *testing.T) {
	tests := []struct {
		value    string
		names    []string
		want     string
		wantErr  bool
	}{
		{"a/b", []string{"urlencode"}, "a%2Fb", false},
		{"a b", []string{"urlencode"}, "a+b", false},
		{"a/b", []string{"doubleurlencode"}, "a%252Fb", false},
		{"x", []string{"base64encode"}, "eA==", false},
		{"eA==", []string{"base64decode"}, "x", false},
		{"<script>", []string{"htmlencode"}, "&lt;script&gt;", false},
		{"a/b", []string{"urlencode", "base64encode"}, "YSUyRmI=", false}, // a%2Fb -> base64
		{"x", nil, "x", false},
		{"x", []string{"unknown"}, "x", false},
	}
	for _, tt := range tests {
		got, err := Chain(tt.value, tt.names)
		if (err != nil) != tt.wantErr {
			t.Errorf("Chain(%q, %v) err = %v", tt.value, tt.names, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Chain(%q, %v) = %q, want %q", tt.value, tt.names, got, tt.want)
		}
	}
}

func TestApplyToMap(t *testing.T) {
	encoders := map[string][]string{
		"FUZZ": {"urlencode"},
		"X":    {"base64encode"},
	}
	values := map[string]string{"FUZZ": "a/b", "X": "1", "Y": "unchanged"}
	got := ApplyToMap(encoders, values)
	if got["FUZZ"] != "a%2Fb" {
		t.Errorf("FUZZ = %q, want a%%2Fb", got["FUZZ"])
	}
	if got["X"] != "MQ==" {
		t.Errorf("X = %q, want MQ==", got["X"])
	}
	if got["Y"] != "unchanged" {
		t.Errorf("Y = %q, want unchanged", got["Y"])
	}
}

func TestApplyToMap_empty(t *testing.T) {
	values := map[string]string{"FUZZ": "a"}
	got := ApplyToMap(nil, values)
	if got["FUZZ"] != "a" {
		t.Errorf("nil encoders: got %q", got["FUZZ"])
	}
	got = ApplyToMap(map[string][]string{}, values)
	if got["FUZZ"] != "a" {
		t.Errorf("empty encoders: got %q", got["FUZZ"])
	}
}
