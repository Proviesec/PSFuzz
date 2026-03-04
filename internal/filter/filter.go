// Package filter implements response filtering (status, length, words, regex, duplicates) so only allowed findings are reported.
package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Proviesec/PSFuzz/internal/config"
)

type Input struct {
	StatusCode  int
	Length      int
	Body        string
	ContentType string
	Words       int
	Lines       int
	TimeMS      int
}

// AllowFilter decides whether a response passes the filter (should be reported as a hit). *Pipeline implements AllowFilter.
type AllowFilter interface {
	Allow(in Input) bool
}

type Pipeline struct {
	cfg            *config.Config
	duplicates     map[string]int
	nearDuplicates map[string]int
	mu             sync.Mutex
}

func New(cfg *config.Config) *Pipeline {
	return &Pipeline{cfg: cfg, duplicates: map[string]int{}, nearDuplicates: map[string]int{}}
}

func (p *Pipeline) Allow(in Input) bool {
	if p.cfg == nil {
		return false
	}
	if len(p.cfg.FilterStatus) > 0 && !config.MatchAnyRange(p.cfg.FilterStatus, in.StatusCode) {
		if !p.cfg.ShowStatus {
			return false
		}
	}
	if len(p.cfg.FilterStatusNot) > 0 && config.MatchAnyRange(p.cfg.FilterStatusNot, in.StatusCode) {
		return false
	}
	if len(p.cfg.FilterLength) > 0 && !config.MatchAnyRange(p.cfg.FilterLength, in.Length) {
		return false
	}
	if len(p.cfg.FilterLengthNot) > 0 && config.MatchAnyRange(p.cfg.FilterLengthNot, in.Length) {
		return false
	}
	if len(p.cfg.FilterWords) > 0 && !config.MatchAnyRange(p.cfg.FilterWords, in.Words) {
		return false
	}
	if len(p.cfg.FilterWordsNot) > 0 && config.MatchAnyRange(p.cfg.FilterWordsNot, in.Words) {
		return false
	}
	if len(p.cfg.FilterLines) > 0 && !config.MatchAnyRange(p.cfg.FilterLines, in.Lines) {
		return false
	}
	if len(p.cfg.FilterLinesNot) > 0 && config.MatchAnyRange(p.cfg.FilterLinesNot, in.Lines) {
		return false
	}
	if len(p.cfg.FilterTime) > 0 && !config.MatchAnyRange(p.cfg.FilterTime, in.TimeMS) {
		return false
	}
	if len(p.cfg.FilterTimeNot) > 0 && config.MatchAnyRange(p.cfg.FilterTimeNot, in.TimeMS) {
		return false
	}
	if p.cfg.MinResponseSize > 0 && in.Length < p.cfg.MinResponseSize {
		return false
	}
	if len(p.cfg.FilterContentTypes) > 0 {
		ct := strings.ToLower(strings.TrimSpace(in.ContentType))
		matched := false
		for _, allowed := range p.cfg.FilterContentTypes {
			a := strings.ToLower(strings.TrimSpace(allowed))
			if a != "" && (ct == a || strings.HasPrefix(ct, a+";")) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if p.cfg.FilterMatchWord != "" && !strings.Contains(in.Body, p.cfg.FilterMatchWord) {
		return false
	}
	if p.cfg.FilterMatchRegex != nil || p.cfg.FilterMatchRegexNot != nil {
		body := in.Body
		if p.cfg.FilterRegexTextOnly {
			body = stripHTML(body)
		}
		if p.cfg.FilterMatchRegex != nil && !p.cfg.FilterMatchRegex.MatchString(body) {
			return false
		}
		if p.cfg.FilterMatchRegexNot != nil && p.cfg.FilterMatchRegexNot.MatchString(body) {
			return false
		}
	}
	if len(p.cfg.BlockWords) > 0 {
		lower := strings.ToLower(in.Body)
		for _, word := range p.cfg.BlockWords {
			if word == "" {
				continue
			}
			if strings.Contains(lower, word) {
				return false
			}
		}
	}
	if p.cfg.FilterDuplicates {
		fingerprint := bodyFingerprint(in.Body)
		p.mu.Lock()
		p.duplicates[fingerprint]++
		count := p.duplicates[fingerprint]
		p.mu.Unlock()
		if count > p.cfg.DuplicateThreshold {
			return false
		}
	}
	if p.cfg.NearDuplicates {
		signature := nearSignature(in, p.cfg.NearDuplicateLenBucket, p.cfg.NearDuplicateWordBucket, p.cfg.NearDuplicateLineBucket)
		p.mu.Lock()
		p.nearDuplicates[signature]++
		count := p.nearDuplicates[signature]
		p.mu.Unlock()
		if count > p.cfg.DuplicateThreshold {
			return false
		}
	}
	return true
}

// bodyFingerprint returns a short hash of the normalized body for duplicate detection.
// Uses first 8 bytes of SHA-256 to keep memory and comparison cost low while remaining sufficiently unique.
func bodyFingerprint(body string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(body)), " ")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:8])
}

func nearSignature(in Input, lenBucket, wordBucket, lineBucket int) string {
	if lenBucket <= 0 {
		lenBucket = 20
	}
	if wordBucket <= 0 {
		wordBucket = 5
	}
	if lineBucket <= 0 {
		lineBucket = 5
	}
	ct := strings.ToLower(strings.TrimSpace(in.ContentType))
	if idx := strings.Index(ct, ";"); idx > -1 {
		ct = ct[:idx]
	}
	return strings.Join([]string{
		statusClass(in.StatusCode),
		ct,
		intBucket(in.Length, lenBucket),
		intBucket(in.Words, wordBucket),
		intBucket(in.Lines, lineBucket),
	}, "|")
}

func statusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "other"
	}
}

func intBucket(v int, size int) string {
	if size <= 0 {
		return "0"
	}
	return strconv.Itoa((v / size) * size)
}

var (
	scriptRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	tagRe    = regexp.MustCompile(`<[^>]+>`)
	spaceRe  = regexp.MustCompile(`\s+`)
)

func stripHTML(s string) string {
	s = scriptRe.ReplaceAllString(s, " ")
	s = styleRe.ReplaceAllString(s, " ")
	s = tagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
