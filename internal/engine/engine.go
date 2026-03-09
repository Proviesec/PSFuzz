// Package engine runs the fuzzing scan: produces tasks from wordlists and base URLs, runs concurrent workers, applies filters and response modules, and builds the report.
package engine

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Proviesec/PSFuzz/internal/config"
	"github.com/Proviesec/PSFuzz/internal/encoder"
	"github.com/Proviesec/PSFuzz/internal/filter"
	"github.com/Proviesec/PSFuzz/internal/httpx"
	"github.com/Proviesec/PSFuzz/internal/modules"
)

const (
	wafAdaptiveBaseDelay    = 200 * time.Millisecond
	jitterEWMAAlpha         = 0.3
	jitterEWMAOneMinusAlpha = 0.7
	progressTickInterval    = 2 * time.Second
	calibrationWordLen      = 20
	wafHashRepeatThreshold  = 3

	// StatusCount keys for non-HTTP status (report.StatusCount)
	statusCountError     = "Error"
	statusCountReadError = "ReadError"
)

// Task is a single fuzz request (URL, depth, placeholder values, optional method/headers). Created by the producer and processed by workers.
type Task struct {
	URL      string
	Depth    int
	Values   map[string]string
	Method   string
	Headers  map[string]string
	IsBypass bool
}

// Result is one reported finding: URL, status, metrics, and optional module_data from response analyzers.
type Result struct {
	URL         string                    `json:"url"`
	StatusCode  int                       `json:"status_code"`
	Status      string                    `json:"status"`
	ContentType string                    `json:"content_type,omitempty"`
	RedirectURL string                    `json:"redirect_url,omitempty"`
	Length      int                       `json:"length"`
	Words       int                       `json:"words"`
	Lines       int                       `json:"lines"`
	TimeMS      int                       `json:"time_ms"`
	Depth       int                       `json:"depth"`
	Timestamp   time.Time                 `json:"timestamp"`
	Truncated   bool                      `json:"truncated,omitempty"`
	Confidence  float64                   `json:"confidence,omitempty"`
	Interesting []string                  `json:"interesting,omitempty"`
	Inputs      map[string]string         `json:"input,omitempty"`
	Position    int                       `json:"position,omitempty"`
	ModuleData  map[string]map[string]any `json:"module_data"`
}

// Report holds scan metadata and all results. Written by output.Write in the requested format (JSON, HTML, etc.).
type Report struct {
	TargetURL      string         `json:"target_url"`
	WordlistSource string         `json:"wordlist_source"`
	WordlistCount  int            `json:"wordlist_count"`
	TotalRequests  int64          `json:"total_requests"`
	Duration       time.Duration  `json:"duration"`
	StartedAt      time.Time      `json:"started_at,omitempty"`
	EndedAt        time.Time      `json:"ended_at,omitempty"`
	StatusCount    map[string]int `json:"status_count"`
	Results        []Result       `json:"results"`
	DiscoveredDirs []string       `json:"discovered_dirs"`
	Modules        []string       `json:"modules,omitempty"`
	ExtractedURLs  []string       `json:"extracted_urls,omitempty"`
	Commandline    string         `json:"commandline,omitempty"`
}

// recordFilteredHit increments StatusCount for a status and is used when a response is filtered out (no result added).
func recordFilteredHit(report *Report, mu *sync.Mutex, status string) {
	mu.Lock()
	report.StatusCount[status]++
	mu.Unlock()
}

// runState holds shared state for a single Run (counters, WAF/baseline maps). Passed to processTask to avoid long parameter lists.
type runState struct {
	mu              *sync.Mutex
	totalReq        *atomic.Int64
	totalHits       *atomic.Int64
	totalErr        *atomic.Int64
	lastStatus      *atomic.Int64
	suspiciousCount *atomic.Int64
	wafHashCounts   map[string]int
	hostSuspicion   map[string]int
	hostLatency     map[string]float64
	baselines       map[string]baselineFingerprint
}

// Engine runs the fuzzing scan: enqueues tasks, runs workers, applies filters and modules, builds the report.
type Engine struct {
	cfg       *config.Config
	client    *httpx.Client
	filter    filter.AllowFilter
	wordlists []config.ResolvedWordlist
	testLen   int
	rawReq    *rawRequest
	cancel    context.CancelFunc
	auditFile *os.File
	auditMu   sync.Mutex
	analyzers []modules.Analyzer
}

func New(cfg *config.Config) (*Engine, error) {
	if cfg == nil {
		return nil, errors.New("config must not be nil")
	}
	client, err := httpx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("http client: %w", err)
	}
	return &Engine{
		cfg:     cfg,
		client:  client,
		filter:  filter.New(cfg),
		testLen: -1,
	}, nil
}

// Run resolves wordlists, starts worker goroutines, and processes tasks until context is done or stop conditions are met.
// Returns the report and any error from wordlist resolution or scan setup; context cancellation is not reported as an error.
func (e *Engine) Run(ctx context.Context) (*Report, error) {
	if ctx == nil {
		return nil, errors.New("context must not be nil")
	}
	defer e.client.Close()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	e.cancel = cancel
	if e.cfg.MaxTime > 0 {
		timer := time.AfterFunc(time.Duration(e.cfg.MaxTime)*time.Second, cancel)
		defer timer.Stop()
	}

	wordlists, err := config.ResolveWordlists(ctx, e.cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve wordlists: %w", err)
	}
	e.wordlists = wordlists
	if e.cfg.FilterTestLength {
		if tl, err := e.computeTestLength(ctx); err == nil {
			e.testLen = tl
		}
	}
	if e.cfg.AutoCalibrate {
		e.applyAutoCalibration(ctx)
	}
	e.analyzers = modules.Enabled(&e.cfg.ModuleConfig)
	if e.cfg.RequestFile != "" {
		req, err := parseRawRequest(e.cfg.RequestFile)
		if err != nil {
			return nil, fmt.Errorf("parse request file: %w", err)
		}
		e.rawReq = req
	}
	if e.cfg.ResumeFile != "" {
		// Resume state loaded later when populating visited map
	}

	baseURLs := e.cfg.URLs
	if len(baseURLs) == 0 {
		baseURLs = []string{e.cfg.URL}
	}

	started := time.Now()
	wordlistSource := ""
	wordlistCount := 0
	if len(wordlists) > 0 {
		wordlistSource = wordlists[0].Source
		wordlistCount = len(wordlists[0].Words)
	}
	targetURL := e.cfg.URL
	if targetURL == "" && len(baseURLs) > 0 {
		if len(baseURLs) == 1 {
			targetURL = baseURLs[0]
		} else {
			targetURL = "multiple"
		}
	}
	report := &Report{
		TargetURL:      targetURL,
		WordlistSource: wordlistSource,
		WordlistCount:  wordlistCount,
		StatusCount:    map[string]int{},
		Modules:        append([]string(nil), e.cfg.ModuleConfig.Modules...),
		StartedAt:      started,
	}

	if e.cfg.LoginURL != "" {
		if err := doLogin(runCtx, e.client, e.cfg); err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
	}

	if e.cfg.AuditLogPath != "" {
		f, err := os.OpenFile(e.cfg.AuditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("audit log: %w", err)
		}
		defer func() { _ = f.Close(); e.auditFile = nil }()
		e.auditFile = f
	}

	taskCh := make(chan Task, e.cfg.Concurrency*2)
	var taskWg sync.WaitGroup
	var closeOnce sync.Once
	closeTaskCh := func() {
		closeOnce.Do(func() {
			taskWg.Wait()
			close(taskCh)
		})
	}
	var totalReq atomic.Int64
	var totalHits atomic.Int64
	var totalErr atomic.Int64
	var lastStatus atomic.Int64
	var processed atomic.Int64
	var totalEnqueued atomic.Int64
	var bypassCount atomic.Int64
	var suspiciousCount atomic.Int64
	var mu sync.Mutex
	visited := map[string]struct{}{}
	wafHashCounts := map[string]int{}
	hostSuspicion := map[string]int{}
	hostLatency := map[string]float64{}
	baselines := map[string]baselineFingerprint{}
	if e.cfg.ResumeFile != "" {
		for k := range loadResume(e.cfg.ResumeFile) {
			visited[k] = struct{}{}
		}
	}
	dirs := map[string]struct{}{}
	st := &runState{
		mu:              &mu,
		totalReq:        &totalReq,
		totalHits:       &totalHits,
		totalErr:        &totalErr,
		lastStatus:      &lastStatus,
		suspiciousCount: &suspiciousCount,
		wafHashCounts:   wafHashCounts,
		hostSuspicion:   hostSuspicion,
		hostLatency:     hostLatency,
		baselines:       baselines,
	}

	if len(wordlists) == 0 {
		return nil, fmt.Errorf("no wordlists resolved")
	}
	keywordList := wordlists[0]
	baseBody := e.cfg.RequestData
	baseHeaders := e.cfg.RequestHeaders
	if e.rawReq != nil {
		baseBody = e.rawReq.Body
		baseHeaders = mergeHeaders(baseHeaders, e.rawReq.Headers)
	}
	headerHasKeyword := headersContainAnyKeyword(baseHeaders, wordlists)
	bodyHasKeyword := containsAnyKeyword(baseBody, wordlists)

	enqueue := func(t Task, urlCoversAll bool) {
		if isExcludedURL(t.URL, e.cfg.ExcludePaths) {
			return
		}
		method, body, headers := e.requestPartsFor(t)
		key := t.URL
		if !urlCoversAll || t.Method != "" || len(t.Headers) > 0 {
			key = requestKey(t.URL, method, body, headers)
		}
		mu.Lock()
		if _, ok := visited[key]; ok {
			mu.Unlock()
			return
		}
		visited[key] = struct{}{}
		taskWg.Add(1)
		totalEnqueued.Add(1)
		mu.Unlock()
		select {
		case <-runCtx.Done():
			taskWg.Done()
			return
		case taskCh <- t:
		}
	}

	var wg sync.WaitGroup
	done := make(chan struct{})
	if !e.cfg.Quiet {
		go e.progressLoop(started, &totalReq, &totalHits, &totalErr, &lastStatus, &processed, &totalEnqueued, done)
	}

	// Producer in separate goroutine so it never blocks when the task channel buffer (Concurrency*2)
	// is full. Only the producer closes the channel when it exits (so no "send on closed channel" when
	// context is cancelled while producer is blocked on send).
	go e.runProducer(runCtx, baseURLs, wordlists, keywordList, headerHasKeyword, bodyHasKeyword, enqueue, closeTaskCh, &taskWg)

	for i := 0; i < e.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.runWorker(runCtx, taskCh, report, st, enqueue, keywordList, dirs, &mu, cancel, &processed, &bypassCount, visited, &taskWg)
		}()
	}

	wg.Wait()
	close(done)
	fmt.Fprintln(os.Stderr, "")
	if e.cfg.ResumeFile != "" {
		if err := writeResume(e.cfg.ResumeFile, visited); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to write final resume state: %v\n", err)
		}
	}
	report.TotalRequests = totalReq.Load()
	report.Duration = time.Since(started).Round(time.Millisecond)
	report.EndedAt = time.Now()
	for d := range dirs {
		report.DiscoveredDirs = append(report.DiscoveredDirs, d)
	}
	sort.Strings(report.DiscoveredDirs)
	report.ExtractedURLs = aggregateExtractedURLs(report.Results)
	return report, nil
}

// runProducer fills taskCh with initial and recursion tasks for each base URL. It runs in a goroutine and closes taskCh when done (after all enqueued tasks are finished, via taskWg).
func (e *Engine) runProducer(ctx context.Context, baseURLs []string, wordlists []config.ResolvedWordlist, keywordList config.ResolvedWordlist, headerHasKeyword bool, bodyHasKeyword bool, enqueue func(Task, bool), closeTaskCh func(), taskWg *sync.WaitGroup) {
	defer closeTaskCh()
	for _, baseURL := range baseURLs {
		if ctx.Err() != nil {
			return
		}
		urlTemplate := baseURL
		taskHeaders := map[string]string{}
		if e.rawReq != nil {
			base := baseURL
			if host := e.rawReq.Headers["Host"]; host != "" && baseURL == e.cfg.URL {
				base = e.cfg.RequestProto + "://" + host
			}
			urlTemplate = joinBaseAndPathWithProto(base, e.rawReq.Path, e.cfg.RequestProto)
			if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
				taskHeaders["Host"] = u.Host
			}
		}
		urlHasKeyword := containsAnyKeyword(urlTemplate, wordlists)
		urlCoversAll := urlTemplateCoversAllKeywords(urlTemplate, wordlists)
		emitValues(e.cfg.InputMode, wordlists, func(values map[string]string) {
			encVals := encoder.ApplyToMap(e.cfg.Encoders, values)
			urlVal := urlTemplate
			switch {
			case urlHasKeyword:
				urlVal = applyTemplate(urlTemplate, encVals)
			case headerHasKeyword || bodyHasKeyword:
				// urlVal already urlTemplate
			default:
				urlVal = buildBaseURL(urlTemplate, encVals[keywordList.Keyword], e.cfg.CheckBackslash)
			}
			if len(e.cfg.Verbs) > 0 {
				for _, verb := range e.cfg.Verbs {
					enqueue(Task{URL: urlVal, Depth: 0, Values: values, Headers: taskHeaders, Method: verb}, urlCoversAll)
				}
			} else {
				enqueue(Task{URL: urlVal, Depth: 0, Values: values, Headers: taskHeaders}, urlCoversAll)
			}
		})
	}
}

// runWorker consumes tasks from taskCh, calls processTask, enqueues recursion/bypass tasks, and updates report and shared state.
func (e *Engine) runWorker(ctx context.Context, taskCh <-chan Task, report *Report, st *runState, enqueue func(Task, bool), keywordList config.ResolvedWordlist, dirs map[string]struct{}, mu *sync.Mutex, cancel context.CancelFunc, processed *atomic.Int64, bypassCount *atomic.Int64, visited map[string]struct{}, taskWg *sync.WaitGroup) {
	for task := range taskCh {
		taskCtx := ctx
		var taskCancel context.CancelFunc
		if e.cfg.MaxTimeJob > 0 {
			taskCtx, taskCancel = context.WithTimeout(ctx, time.Duration(e.cfg.MaxTimeJob)*time.Second)
		}
		statusCode, hasResp, urlsToEnqueue := e.processTask(taskCtx, task, report, st)
		if taskCancel != nil {
			taskCancel()
		}
		if hasResp && len(urlsToEnqueue) > 0 {
			for _, u := range urlsToEnqueue {
				enqueue(Task{URL: u, Depth: task.Depth + 1, Values: copyValues(task.Values), Method: task.Method, Headers: task.Headers}, false)
			}
		}
		if e.cfg.ResumeFile != "" && e.cfg.ResumeEvery > 0 && processed.Load()%int64(e.cfg.ResumeEvery) == 0 {
			mu.Lock()
			snapshot := make(map[string]struct{}, len(visited))
			for k := range visited {
				snapshot[k] = struct{}{}
			}
			mu.Unlock()
			if err := writeResume(e.cfg.ResumeFile, snapshot); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write resume checkpoint: %v\n", err)
			}
		}
		if hasResp && len(e.cfg.StopOnStatus) > 0 && config.MatchAnyRange(e.cfg.StopOnStatus, statusCode) {
			cancel()
		}
		if e.cfg.StopOnMatches > 0 && int(st.totalHits.Load()) >= e.cfg.StopOnMatches {
			cancel()
		}
		if hasResp && task.Depth < e.cfg.Depth {
			if base, recurse := shouldRecurse(task.URL, task.Depth, statusCode, e.cfg); recurse {
				mu.Lock()
				dirs[base] = struct{}{}
				mu.Unlock()
				for _, w := range keywordList.Words {
					encVals := encoder.ApplyToMap(e.cfg.Encoders, map[string]string{keywordList.Keyword: w})
					encodedWord := encVals[keywordList.Keyword]
					if encodedWord == "" {
						encodedWord = w
					}
					next := buildRecursiveURL(base, encodedWord)
					values := copyValues(task.Values)
					values[keywordList.Keyword] = w
					if _, ok := values["FUZZ"]; !ok {
						values["FUZZ"] = w
					}
					if len(e.cfg.Verbs) > 0 {
						for _, verb := range e.cfg.Verbs {
							enqueue(Task{URL: next, Depth: task.Depth + 1, Values: values, Method: verb}, false)
						}
					} else {
						enqueue(Task{URL: next, Depth: task.Depth + 1, Values: values, Method: task.Method}, false)
					}
				}
			}
		}
		if hasResp && !task.IsBypass && e.cfg.Bypass && (statusCode == http.StatusUnauthorized || statusCode == http.StatusPaymentRequired || statusCode == http.StatusForbidden) {
			if e.cfg.WAFAdaptive && st.totalReq.Load() > 0 && float64(st.suspiciousCount.Load())/float64(st.totalReq.Load()) > 0.3 {
				// skip bypass when WAF suspicion is high
			} else if e.cfg.BypassBudget > 0 && int(bypassCount.Load()) >= e.cfg.BypassBudget {
				// skip if budget exceeded
			} else if e.cfg.BypassRatioLimit > 0 && float64(bypassCount.Load())/float64(st.totalReq.Load()+1) > e.cfg.BypassRatioLimit {
				// skip if ratio exceeded
			} else {
				for _, extra := range bypassVariants(task.URL) {
					bypassCount.Add(1)
					enqueue(Task{
						URL:      extra.URL,
						Depth:    task.Depth,
						Values:   copyValues(task.Values),
						Method:   extra.Method,
						Headers:  extra.Headers,
						IsBypass: true,
					}, false)
				}
			}
		}
		if e.cfg.WAFAdaptive && int(st.suspiciousCount.Load()) >= e.cfg.WAFSlowdownThreshold {
			e.client.SetScale(e.cfg.WAFSlowdownFactor)
		}
		taskWg.Done()
	}
}

// urlsFromData returns URLs from a module data map's "urls" key ([]string or []any), trimmed; if skipHash is true, entries starting with '#' are omitted.
func urlsFromData(data map[string]any, skipHash bool) []string {
	if data == nil {
		return nil
	}
	urlsVal, ok := data["urls"]
	if !ok {
		return nil
	}
	var out []string
	switch v := urlsVal.(type) {
	case []string:
		for _, u := range v {
			u = strings.TrimSpace(u)
			if u != "" && (!skipHash || u[0] != '#') {
				out = append(out, u)
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" && (!skipHash || s[0] != '#') {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// aggregateExtractedURLs collects all URLs from any module that outputs "urls" in module_data (deduplicated).
// So extracted_urls is driven by enabled modules (e.g. urlextract, links), not a separate module.
func aggregateExtractedURLs(results []Result) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, r := range results {
		if r.ModuleData == nil {
			continue
		}
		for _, data := range r.ModuleData {
			for _, u := range urlsFromData(data, false) {
				if _, ok := seen[u]; !ok {
					seen[u] = struct{}{}
					out = append(out, u)
				}
			}
		}
	}
	return out
}

func (e *Engine) processTask(ctx context.Context, task Task, report *Report, st *runState) (statusCode int, hasResp bool, urlsToEnqueue []string) {
	host := hostFromURL(task.URL)
	extraDelay := e.computeExtraDelay(host, st)

	method, body, headers := e.requestPartsFor(task)
	spec := httpx.RequestSpec{
		URL:     task.URL,
		Method:  method,
		Body:    body,
		Headers: headers,
		Delay:   extraDelay,
	}
	start := time.Now()
	result, err := e.client.Do(ctx, spec)
	st.totalReq.Add(1)
	if err != nil {
		e.handleRequestError(task, report, st, err)
		return 0, false, nil
	}
	resp := result.Resp
	finalURL := result.FinalURL
	defer resp.Body.Close()
	if e.cfg.ReplayProxy != "" && !e.cfg.ReplayOnMatch {
		e.client.Replay(ctx, spec, e.cfg.ReplayProxy)
	}

	respBody, truncated, err := readBodyWithLimit(resp.Body, config.EffectiveMaxResponseSize(e.cfg))
	if err != nil {
		st.mu.Lock()
		report.StatusCount[statusCountReadError]++
		st.mu.Unlock()
		return resp.StatusCode, true, nil
	}
	if e.auditFile != nil {
		e.writeAuditEntry(&spec, resp.StatusCode, resp.Header, respBody)
	}
	elapsedMS := int(time.Since(start).Milliseconds())
	st.lastStatus.Store(int64(resp.StatusCode))
	e.updateJitterLatency(host, elapsedMS, st)

	text := string(respBody)
	lowerText := strings.ToLower(text)
	status := resp.Status
	contentType := resp.Header.Get("Content-Type")
	redirectURL := resolveRedirectURL(task.URL, finalURL, resp)
	words := countWords(text)
	lines := countLines(text)

	if e.cfg.AutoWildcard && host != "" {
		baseline := e.ensureBaseline(ctx, host, task, st)
		if baseline.Valid && matchesBaseline(baseline, resp.StatusCode, len(respBody), words, lines) {
			recordFilteredHit(report, st.mu, status)
			return resp.StatusCode, true, nil
		}
	}

	suspect := e.trackWAFSuspicion(host, resp, lowerText, st)
	title := extractTitle(text, lowerText)
	interesting := findInteresting(lowerText, e.cfg.InterestingStrings)

	if filtered, code := e.applyFilters(task, report, st, resp.StatusCode, status, title, text, contentType, respBody, words, lines, elapsedMS, truncated, suspect, interesting); filtered {
		return code, true, nil
	}

	confidence := computeConfidence(resp.StatusCode, title, len(respBody), words, lines, contentType, truncated, suspect)
	moduleData := runModules(ctx, e.analyzers, task, method, resp, text, contentType, len(respBody), words, lines)

	e.recordResult(task, report, st, resp.StatusCode, status, contentType, redirectURL, len(respBody), words, lines, elapsedMS, truncated, confidence, interesting, moduleData)

	if e.cfg.DumpResponses {
		dumpPath := e.cfg.DumpDir
		if dumpPath == "" {
			dumpPath = e.cfg.OutputBase + "_responses"
		}
		_ = dumpResponse(dumpPath, task.URL, method, resp.StatusCode, contentType, respBody)
	}
	st.totalHits.Add(1)
	e.printResult(task, status, len(respBody), words)
	if e.cfg.ReplayProxy != "" && e.cfg.ReplayOnMatch {
		e.client.Replay(ctx, spec, e.cfg.ReplayProxy)
	}
	urlsToEnqueue = extractURLsFromModuleData(moduleData, e.cfg.ModuleConfig.EnqueueModuleUrls)
	return resp.StatusCode, true, urlsToEnqueue
}

func hostFromURL(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return u.Host
	}
	return ""
}

func (e *Engine) computeExtraDelay(host string, st *runState) time.Duration {
	var extraDelay time.Duration
	if e.cfg.WAFAdaptive && host != "" {
		st.mu.Lock()
		sCount := st.hostSuspicion[host]
		st.mu.Unlock()
		if sCount > 0 {
			extraDelay = time.Duration(float64(wafAdaptiveBaseDelay) * float64(sCount) * e.cfg.WAFSlowdownFactor)
		}
	}
	if e.cfg.JitterProfile && host != "" {
		st.mu.Lock()
		ewma := st.hostLatency[host]
		st.mu.Unlock()
		if ewma > float64(e.cfg.JitterThresholdMS) {
			delta := (ewma - float64(e.cfg.JitterThresholdMS)) * e.cfg.JitterFactor
			if delta > 0 {
				extraDelay += time.Duration(delta) * time.Millisecond
			}
		}
	}
	return extraDelay
}

func (e *Engine) handleRequestError(task Task, report *Report, st *runState, err error) {
	st.totalErr.Add(1)
	st.lastStatus.Store(-1)
	if e.cfg.ShowStatus && !e.cfg.Quiet {
		fmt.Printf("%s [ERR] %v\n", task.URL, err)
	}
	st.mu.Lock()
	report.StatusCount[statusCountError]++
	st.mu.Unlock()
	if e.cfg.StopOnErrors {
		e.cancel()
	}
}

func (e *Engine) updateJitterLatency(host string, elapsedMS int, st *runState) {
	if !e.cfg.JitterProfile || host == "" {
		return
	}
	st.mu.Lock()
	prev := st.hostLatency[host]
	if prev == 0 {
		st.hostLatency[host] = float64(elapsedMS)
	} else {
		st.hostLatency[host] = jitterEWMAAlpha*float64(elapsedMS) + jitterEWMAOneMinusAlpha*prev
	}
	st.mu.Unlock()
}

func resolveRedirectURL(taskURL, finalURL string, resp *http.Response) string {
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if loc := resp.Header.Get("Location"); loc != "" {
			return loc
		}
	}
	if finalURL != "" {
		reqNorm := strings.TrimSuffix(taskURL, "/")
		finalNorm := strings.TrimSuffix(finalURL, "/")
		if finalNorm != "" && finalNorm != reqNorm {
			return finalURL
		}
	}
	return ""
}

func (e *Engine) trackWAFSuspicion(host string, resp *http.Response, lowerText string, st *runState) bool {
	suspect := isWAFSuspect(resp.Header, lowerText) || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden
	if !suspect {
		return false
	}
	st.suspiciousCount.Add(1)
	if host != "" {
		st.mu.Lock()
		st.hostSuspicion[host]++
		st.mu.Unlock()
	}
	hash := sha1.Sum([]byte(lowerText))
	h := hex.EncodeToString(hash[:])
	st.mu.Lock()
	st.wafHashCounts[h]++
	count := st.wafHashCounts[h]
	st.mu.Unlock()
	if count > wafHashRepeatThreshold && host != "" {
		st.mu.Lock()
		st.hostSuspicion[host]++
		st.mu.Unlock()
	}
	return true
}

// applyFilters runs all configured response filters (interesting strings, test length, wrong status/subdomain, 404 title, and the user filter).
// Returns true if the response should be filtered out (not reported).
func (e *Engine) applyFilters(task Task, report *Report, st *runState, statusCode int, status, title, text, contentType string, respBody []byte, words, lines, elapsedMS int, truncated, suspect bool, interesting []string) (filtered bool, code int) {
	if len(e.cfg.InterestingStrings) > 0 && len(interesting) == 0 {
		recordFilteredHit(report, st.mu, status)
		return true, statusCode
	}
	if e.cfg.FilterTestLength && e.testLen > 0 && len(respBody) == e.testLen {
		recordFilteredHit(report, st.mu, status)
		return true, statusCode
	}
	if e.cfg.FilterWrongStatus200 && isWrongStatus200(title, len(respBody)) {
		recordFilteredHit(report, st.mu, status)
		return true, statusCode
	}
	if e.cfg.FilterWrongSubdomain && isWrongSubdomain(title, len(respBody), statusCode) {
		recordFilteredHit(report, st.mu, status)
		return true, statusCode
	}
	if e.cfg.FilterPossible404 && strings.Contains(title, "404") {
		recordFilteredHit(report, st.mu, status)
		return true, statusCode
	}
	if !e.filter.Allow(filter.Input{
		StatusCode:  statusCode,
		Length:      len(respBody),
		Body:        text,
		ContentType: contentType,
		Words:       words,
		Lines:       lines,
		TimeMS:      elapsedMS,
	}) {
		recordFilteredHit(report, st.mu, status)
		return true, statusCode
	}
	return false, statusCode
}

func (e *Engine) recordResult(task Task, report *Report, st *runState, statusCode int, status, contentType, redirectURL string, length, words, lines, elapsedMS int, truncated bool, confidence float64, interesting []string, moduleData map[string]map[string]any) {
	st.mu.Lock()
	report.StatusCount[status]++
	position := len(report.Results) + 1
	report.Results = append(report.Results, Result{
		URL:         task.URL,
		StatusCode:  statusCode,
		Status:      status,
		ContentType: contentType,
		RedirectURL: redirectURL,
		Length:      length,
		Words:       words,
		Lines:       lines,
		TimeMS:      elapsedMS,
		Depth:       task.Depth,
		Timestamp:   time.Now(),
		Truncated:   truncated,
		Confidence:  confidence,
		Interesting: interesting,
		Inputs:      copyValues(task.Values),
		Position:    position,
		ModuleData:  moduleData,
	})
	st.mu.Unlock()
}

func extractURLsFromModuleData(moduleData map[string]map[string]any, moduleNames string) []string {
	if moduleData == nil || moduleNames == "" {
		return nil
	}
	var out []string
	seen := make(map[string]struct{})
	for _, name := range strings.Split(moduleNames, ",") {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}
		data, ok := moduleData[name]
		if !ok {
			continue
		}
		for _, u := range urlsFromData(data, true) {
			if _, ok := seen[u]; !ok {
				seen[u] = struct{}{}
				out = append(out, u)
			}
		}
	}
	return out
}

func runModules(ctx context.Context, analyzers []modules.Analyzer, task Task, method string, resp *http.Response, text, contentType string, length, words, lines int) map[string]map[string]any {
	if len(analyzers) == 0 {
		return nil
	}
	headerMap := headerMapFromHTTPHeader(resp.Header)
	if method == "" {
		method = task.Method
	}
	in := modules.Input{
		URL:         task.URL,
		Method:      method,
		StatusCode:  resp.StatusCode,
		Headers:     headerMap,
		Body:        text,
		ContentType: contentType,
		Length:      length,
		Words:       words,
		Lines:       lines,
	}
	return modules.Run(ctx, analyzers, in)
}

// buildTestRequestParts builds URL, method, body, and headers for a calibration/test request (used by computeTestLength and applyAutoCalibration).
func (e *Engine) buildTestRequestParts(values map[string]string, urlWord string) (testURL string, method string, body string, headers map[string]string) {
	template := e.cfg.URL
	if e.rawReq != nil {
		base := e.cfg.URL
		if host := e.rawReq.Headers["Host"]; host != "" {
			base = e.cfg.RequestProto + "://" + host
		}
		template = joinBaseAndPathWithProto(base, e.rawReq.Path, e.cfg.RequestProto)
	}
	testURL = applyTemplate(buildBaseURL(template, urlWord, e.cfg.CheckBackslash), values)
	method = e.cfg.RequestMethod
	body = e.cfg.RequestData
	headers = e.cfg.RequestHeaders
	if e.rawReq != nil {
		method = e.rawReq.Method
		body = e.rawReq.Body
		headers = mergeHeaders(headers, e.rawReq.Headers)
	}
	return testURL, method, applyTemplate(body, values), applyHeaderTemplate(headers, values)
}

func (e *Engine) computeTestLength(ctx context.Context) (int, error) {
	randWord := e.randomString(calibrationWordLen)
	values := map[string]string{"FUZZ": randWord}
	testURL, method, body, headers := e.buildTestRequestParts(values, randWord)
	result, err := e.client.Do(ctx, httpx.RequestSpec{
		URL:     testURL,
		Method:  method,
		Body:    body,
		Headers: headers,
	})
	if err != nil {
		return -1, fmt.Errorf("calibration request: %w", err)
	}
	defer result.Resp.Body.Close()
	bodyBytes, _, err := readBodyWithLimit(result.Resp.Body, config.EffectiveMaxResponseSize(e.cfg))
	if err != nil {
		return -1, fmt.Errorf("read calibration response: %w", err)
	}
	return len(bodyBytes), nil
}

func (e *Engine) applyAutoCalibration(ctx context.Context) {
	n := e.cfg.AutoCalibrateN
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		randWord := e.randomString(calibrationWordLen)
		values := map[string]string{"FUZZ": randWord}
		testURL, method, body, headers := e.buildTestRequestParts(values, randWord)
		result, err := e.client.Do(ctx, httpx.RequestSpec{
			URL:     testURL,
			Method:  method,
			Body:    body,
			Headers: headers,
		})
		if err != nil {
			continue
		}
		bodyBytes, _, err := readBodyWithLimit(result.Resp.Body, config.EffectiveMaxResponseSize(e.cfg))
		_ = result.Resp.Body.Close()
		if err != nil {
			continue
		}
		length := len(bodyBytes)
		text := string(bodyBytes)
		words := countWords(text)
		lines := countLines(text)
		addExactRange(&e.cfg.FilterLengthNot, length)
		addExactRange(&e.cfg.FilterWordsNot, words)
		addExactRange(&e.cfg.FilterLinesNot, lines)
	}
}

func (e *Engine) randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = letters[rand.IntN(len(letters))]
	}
	return string(out)
}

func (e *Engine) requestPartsFor(task Task) (string, string, map[string]string) {
	method := e.cfg.RequestMethod
	bodyTemplate := e.cfg.RequestData
	headers := e.cfg.RequestHeaders
	if e.rawReq != nil {
		method = e.rawReq.Method
		bodyTemplate = e.rawReq.Body
		headers = mergeHeaders(headers, e.rawReq.Headers)
	}
	if task.Method != "" {
		method = task.Method
	}
	encVals := encoder.ApplyToMap(e.cfg.Encoders, task.Values)
	body := applyTemplate(bodyTemplate, encVals)
	outHeaders := applyHeaderTemplate(headers, encVals)
	if len(task.Headers) > 0 {
		outHeaders = mergeHeaders(outHeaders, task.Headers)
	}
	if e.cfg.VHostFuzz && len(e.wordlists) > 0 && len(task.Values) > 0 {
		if v := task.Values[e.wordlists[0].Keyword]; v != "" {
			outHeaders["Host"] = v
		}
	}
	return method, body, outHeaders
}

func (e *Engine) printResult(task Task, status string, length int, words int) {
	if e.cfg.Quiet {
		return
	}
	target := task.URL
	if e.cfg.OnlyDomains {
		if u, err := url.Parse(task.URL); err == nil && u.Host != "" {
			target = u.Host
		}
	}
	fmt.Printf("%s [%s] len=%d words=%d depth=%d\n", target, status, length, words, task.Depth)
}

func (e *Engine) progressLoop(start time.Time, totalReq *atomic.Int64, totalHits *atomic.Int64, totalErr *atomic.Int64, lastStatus *atomic.Int64, processed *atomic.Int64, totalEnqueued *atomic.Int64, done <-chan struct{}) {
	ticker := time.NewTicker(progressTickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			req := totalReq.Load()
			hits := totalHits.Load()
			errs := totalErr.Load()
			last := lastStatus.Load()
			doneCount := processed.Load()
			enqueued := totalEnqueued.Load()
			elapsed := time.Since(start).Seconds()
			rate := 0.0
			if elapsed > 0 {
				rate = float64(req) / elapsed
			}
			remaining := enqueued - doneCount
			eta := "n/a"
			if rate > 0 && remaining > 0 {
				secs := float64(remaining) / rate
				eta = (time.Duration(secs) * time.Second).Round(time.Second).String()
			}
			lastStr := "-"
			if last < 0 {
				lastStr = "err"
			} else if last > 0 {
				lastStr = fmt.Sprintf("%d", last)
			}
			fmt.Fprintf(os.Stderr, "\r\033[2KRequests: %d | Hits: %d | Errors: %d | Rate: %.1f req/s", req, hits, errs, rate)
			fmt.Fprintf(os.Stderr, " | Queue: %d | ETA: %s | Last: %s", remaining, eta, lastStr)
		}
	}
}

func (e *Engine) writeAuditEntry(spec *httpx.RequestSpec, statusCode int, respHeader http.Header, respBody []byte) {
	if e.auditFile == nil {
		return
	}
	max := e.cfg.AuditMaxBodySize
	reqBody := spec.Body
	var respBodyStr string
	if max > 0 {
		if len(reqBody) > max {
			reqBody = reqBody[:max] + "...[truncated]"
		}
		if len(respBody) > max {
			respBodyStr = string(respBody[:max]) + "...[truncated]"
		} else {
			respBodyStr = string(respBody)
		}
	} else {
		respBodyStr = string(respBody)
	}
	respHdr := headerMapFromHTTPHeader(respHeader)
	entry := struct {
		Timestamp string         `json:"timestamp"`
		Request   map[string]any `json:"request"`
		Response  map[string]any `json:"response"`
	}{
		Timestamp: time.Now().Format(time.RFC3339),
		Request: map[string]any{
			"url":     spec.URL,
			"method":  spec.Method,
			"headers": spec.Headers,
			"body":    reqBody,
		},
		Response: map[string]any{
			"status_code": statusCode,
			"headers":     respHdr,
			"body":        respBodyStr,
		},
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	e.auditMu.Lock()
	_, _ = e.auditFile.Write(append(b, '\n'))
	e.auditMu.Unlock()
}
