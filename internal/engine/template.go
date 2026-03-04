package engine

import (
	"strings"

	"github.com/Proviesec/PSFuzz/internal/config"
)

func containsAnyKeyword(template string, lists []config.ResolvedWordlist) bool {
	if strings.Contains(template, "#PSFUZZ#") || strings.Contains(template, "FUZZ") {
		return true
	}
	for _, wl := range lists {
		if strings.Contains(template, wl.Keyword) {
			return true
		}
	}
	return false
}

func headersContainAnyKeyword(headers map[string]string, lists []config.ResolvedWordlist) bool {
	for _, v := range headers {
		if containsAnyKeyword(v, lists) {
			return true
		}
	}
	return false
}

func urlTemplateCoversAllKeywords(template string, lists []config.ResolvedWordlist) bool {
	for _, wl := range lists {
		if wl.Keyword == "FUZZ" {
			if strings.Contains(template, "#PSFUZZ#") || strings.Contains(template, "FUZZ") {
				continue
			}
			return false
		}
		if !strings.Contains(template, wl.Keyword) {
			return false
		}
	}
	return true
}

func applyTemplate(template string, values map[string]string) string {
	if template == "" || len(values) == 0 {
		return strings.ReplaceAll(template, "#PSFUZZ#", "")
	}
	out := strings.ReplaceAll(template, "#PSFUZZ#", values["FUZZ"])
	for k, v := range values {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}

func applyHeaderTemplate(headers map[string]string, values map[string]string) map[string]string {
	if len(headers) == 0 || len(values) == 0 {
		return headers
	}
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		out[k] = applyTemplate(v, values)
	}
	return out
}

func copyValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func emitCombinations(lists []config.ResolvedWordlist, emit func(map[string]string)) {
	if len(lists) == 0 {
		return
	}
	var walk func(int, map[string]string)
	walk = func(idx int, acc map[string]string) {
		if idx == len(lists) {
			copyMap := make(map[string]string, len(acc)+1)
			for k, v := range acc {
				copyMap[k] = v
			}
			if _, ok := copyMap["FUZZ"]; !ok && len(lists) > 0 {
				copyMap["FUZZ"] = copyMap[lists[0].Keyword]
			}
			emit(copyMap)
			return
		}
		list := lists[idx]
		for _, w := range list.Words {
			acc[list.Keyword] = w
			walk(idx+1, acc)
		}
	}
	walk(0, map[string]string{})
}

func emitValues(mode string, lists []config.ResolvedWordlist, emit func(map[string]string)) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "pitchfork":
		emitPitchfork(lists, emit)
	case "sniper":
		emitSniper(lists, emit)
	default:
		emitCombinations(lists, emit)
	}
}

func emitPitchfork(lists []config.ResolvedWordlist, emit func(map[string]string)) {
	if len(lists) == 0 {
		return
	}
	minLen := -1
	for _, list := range lists {
		if minLen == -1 || len(list.Words) < minLen {
			minLen = len(list.Words)
		}
	}
	if minLen <= 0 {
		return
	}
	for i := 0; i < minLen; i++ {
		values := make(map[string]string, len(lists)+1)
		for _, list := range lists {
			values[list.Keyword] = list.Words[i]
		}
		if _, ok := values["FUZZ"]; !ok {
			values["FUZZ"] = values[lists[0].Keyword]
		}
		emit(values)
	}
}

func emitSniper(lists []config.ResolvedWordlist, emit func(map[string]string)) {
	if len(lists) == 0 {
		return
	}
	base := make(map[string]string, len(lists)+1)
	for _, list := range lists {
		if len(list.Words) > 0 {
			base[list.Keyword] = list.Words[0]
		} else {
			base[list.Keyword] = ""
		}
	}
	if _, ok := base["FUZZ"]; !ok {
		base["FUZZ"] = base[lists[0].Keyword]
	}
	for _, list := range lists {
		for _, w := range list.Words {
			values := make(map[string]string, len(base)+1)
			for k, v := range base {
				values[k] = v
			}
			values[list.Keyword] = w
			if _, ok := values["FUZZ"]; !ok {
				values["FUZZ"] = values[lists[0].Keyword]
			}
			emit(values)
		}
	}
}
