package config

import (
	"regexp"
	"strings"
	"time"
)

// applyCLIConfig applies only flags that were explicitly set (visited) from cli to cfg.
// Used after optional config file load so CLI overrides file values.
func applyCLIConfig(cfg *Config, cli *cliConfig, visited map[string]bool) error {
	if isSet(visited, "u", "url") {
		cfg.URL = cli.URL
	}
	if isSet(visited, "list", "url-list") {
		cfg.URLs = loadURLList(cli.URLList)
	}
	if isSet(visited, "d", "dirlist", "w") {
		cfg.Wordlist = cli.Dirlist
	}
	if isSet(visited, "e", "ext") {
		cfg.Extensions = normalizeExtensions(ParseCSV(cli.Extensions))
	}
	if isSet(visited, "ext-defaults") {
		cfg.UseDefaultExtensions = cli.UseDefaultExtensions
	}
	if isSet(visited, "mode", "input-mode") {
		cfg.InputMode = strings.ToLower(strings.TrimSpace(cli.InputMode))
	}
	if isSet(visited, "ic") {
		cfg.IgnoreWordlistComments = cli.IgnoreWordlistComments
	}
	if isSet(visited, "awc", "auto-wildcard") {
		cfg.AutoWildcard = cli.AutoWildcard
	}
	if isSet(visited, "max-size") {
		cfg.MaxResponseSize = cli.MaxResponseSize
	}
	if isSet(visited, "min-size") {
		cfg.MinResponseSize = cli.MinResponseSize
	}
	if isSet(visited, "save-config") {
		cfg.SaveConfigPath = cli.SaveConfigPath
	}
	if isSet(visited, "c", "concurrency") {
		cfg.Concurrency = cli.Concurrency
	}
	if isSet(visited, "D", "depth") {
		cfg.Depth = cli.Depth
	}
	if isSet(visited, "rs", "recursionSmart") {
		cfg.RecursionSmart = cli.RecursionSmart
	}
	if isSet(visited, "rsc", "recursionStatusCodes") {
		if err := applyStatusRanges(&cfg.RecursionStatus, cli.RecursionStatusCode); err != nil {
			return err
		}
	}
	if isSet(visited, "r", "redirect") {
		cfg.FollowRedirects = cli.Redirect
	}
	if isSet(visited, "o", "output") {
		cfg.OutputBase = strings.TrimSuffix(strings.TrimSuffix(cli.Output, ".txt"), ".json")
		cfg.OutputBase = strings.TrimSuffix(cfg.OutputBase, ".html")
	}
	if isSet(visited, "of", "outputFormat") {
		cfg.OutputFormat = cli.OutputFormat
	}
	if isSet(visited, "tr", "throttleRate", "rate") {
		cfg.ThrottleRPS = cli.ThrottleRate
	}
	if isSet(visited, "p") {
		if min, max, err := parseDelay(cli.Delay); err == nil {
			cfg.DelayMin = min
			cfg.DelayMax = max
		} else {
			return err
		}
	}
	if isSet(visited, "fsc", "filterStatusCode", "mc") {
		if err := applyStatusRanges(&cfg.FilterStatus, cli.FilterStatusCode); err != nil {
			return err
		}
	}
	if isSet(visited, "fscn", "filterStatusCodeNot", "fc") {
		if err := applyStatusRanges(&cfg.FilterStatusNot, cli.FilterStatusCodeNot); err != nil {
			return err
		}
	}
	if isSet(visited, "fl", "filterLength", "ms") {
		if err := applyRanges(&cfg.FilterLength, cli.FilterLength); err != nil {
			return err
		}
	}
	if isSet(visited, "fln", "filterLengthNot", "fs") {
		if err := applyRanges(&cfg.FilterLengthNot, cli.FilterLengthNot); err != nil {
			return err
		}
	}
	if isSet(visited, "mw", "filterWords") {
		if err := applyRanges(&cfg.FilterWords, cli.FilterWords); err != nil {
			return err
		}
	}
	if isSet(visited, "fw", "filterWordsNot") {
		if err := applyRanges(&cfg.FilterWordsNot, cli.FilterWordsNot); err != nil {
			return err
		}
	}
	if isSet(visited, "ml", "filterLines") {
		if err := applyRanges(&cfg.FilterLines, cli.FilterLines); err != nil {
			return err
		}
	}
	if isSet(visited, "fls", "filterLinesNot") {
		if err := applyRanges(&cfg.FilterLinesNot, cli.FilterLinesNot); err != nil {
			return err
		}
	}
	if isSet(visited, "mt", "filterTime") {
		if err := applyRanges(&cfg.FilterTime, cli.FilterTime); err != nil {
			return err
		}
	}
	if isSet(visited, "ft", "filterTimeNot") {
		if err := applyRanges(&cfg.FilterTimeNot, cli.FilterTimeNot); err != nil {
			return err
		}
	}
	if isSet(visited, "nd") {
		cfg.NearDuplicates = cli.NearDuplicates
	}
	if isSet(visited, "nd-len") {
		cfg.NearDuplicateLenBucket = cli.NearDuplicateLenBucket
	}
	if isSet(visited, "nd-words") {
		cfg.NearDuplicateWordBucket = cli.NearDuplicateWordBucket
	}
	if isSet(visited, "nd-lines") {
		cfg.NearDuplicateLineBucket = cli.NearDuplicateLineBucket
	}
	if isSet(visited, "bw") {
		cfg.BlockWords = loadWordList(cli.BlockWords)
	}
	if isSet(visited, "is") {
		cfg.InterestingStrings = loadWordList(cli.InterestingStrings)
	}
	if isSet(visited, "fm", "filterMatchWord") {
		cfg.FilterMatchWord = cli.FilterMatchWord
	}
	if isSet(visited, "mr", "filterMatchRegex") {
		rx, err := regexp.Compile(cli.FilterMatchRegex)
		if err != nil {
			return err
		}
		cfg.FilterMatchRegex = rx
	}
	if isSet(visited, "mrn", "filterMatchRegexNot", "fr") {
		rx, err := regexp.Compile(cli.FilterMatchRegexNot)
		if err != nil {
			return err
		}
		cfg.FilterMatchRegexNot = rx
	}
	if isSet(visited, "mrt", "filterMatchRegexTextOnly") {
		cfg.FilterRegexTextOnly = cli.FilterRegexTextOnly
	}
	if isSet(visited, "f", "filterContentType") {
		cfg.FilterContentTypes = ParseCSV(cli.FilterContentType)
	}
	if isSet(visited, "fd", "filterDuplicates") {
		cfg.FilterDuplicates = cli.FilterDuplicates
	}
	if isSet(visited, "dt", "duplicateThreshold") {
		cfg.DuplicateThreshold = cli.DuplicateThreshold
	}
	if isSet(visited, "s", "showStatus") {
		cfg.ShowStatus = cli.ShowStatus
	}
	if isSet(visited, "od", "onlydomains") {
		cfg.OnlyDomains = cli.OnlyDomains
	}
	if isSet(visited, "cb", "checkBackslash") {
		cfg.CheckBackslash = cli.CheckBackslash
	}
	if isSet(visited, "b", "bypass") {
		cfg.Bypass = cli.Bypass
	}
	if isSet(visited, "btr", "bypassTooManyRequests") {
		cfg.BypassTooManyRequests = cli.BypassTooManyRequests
	}
	if isSet(visited, "t", "filterTestLength") {
		cfg.FilterTestLength = cli.FilterTestLength
	}
	if isSet(visited, "fws", "filterWrongStatus200") {
		cfg.FilterWrongStatus200 = cli.FilterWrongStatus200
	}
	if isSet(visited, "fwd", "filterWrongSubdomain") {
		cfg.FilterWrongSubdomain = cli.FilterWrongSubdomain
	}
	if isSet(visited, "p404", "filterPossible404") {
		cfg.FilterPossible404 = cli.FilterPossible404
	}
	if isSet(visited, "ac") {
		cfg.AutoCalibrate = cli.AutoCalibrate
	}
	if isSet(visited, "acn") {
		cfg.AutoCalibrateN = cli.AutoCalibrateN
	}
	if isSet(visited, "rah", "requestAddHeader") {
		cfg.RequestHeaders = ParseKV(cli.Headers, ":")
	}
	if isSet(visited, "H") {
		cfg.RequestHeaders = ParseKV(cli.Headers, ":")
	}
	if isSet(visited, "C", "cookie") {
		cfg.RequestCookies = ParseKV(cli.Cookies, "=")
	}
	if isSet(visited, "raa", "requestAddAgent") {
		cfg.RequestUserAgent = cli.UserAgent
	}
	if isSet(visited, "random-ua") {
		cfg.RandomUserAgent = cli.RandomUserAgent
	}
	if isSet(visited, "wordlist-case") {
		cfg.RandomizeWordlistCase = strings.ToLower(strings.TrimSpace(cli.RandomizeWordlistCase))
	}
	if isSet(visited, "X", "method") {
		cfg.RequestMethod = cli.Method
	}
	if isSet(visited, "data") {
		if err := setRequestDataFromSpec(cfg, strings.TrimSpace(cli.Data)); err != nil {
			return err
		}
	}
	if isSet(visited, "x", "proxy") {
		cfg.Proxy = cli.Proxy
	}
	if isSet(visited, "request") {
		cfg.RequestFile = cli.RequestFile
	}
	if isSet(visited, "replay-proxy") {
		cfg.ReplayProxy = cli.ReplayProxy
	}
	if isSet(visited, "replay-on-match") {
		cfg.ReplayOnMatch = cli.ReplayOnMatch
	}
	if isSet(visited, "request-proto") {
		cfg.RequestProto = cli.RequestProto
	}
	if isSet(visited, "proxy-user") {
		cfg.ProxyUser = cli.ProxyUser
	}
	if isSet(visited, "proxy-pass") {
		cfg.ProxyPass = cli.ProxyPass
	}
	if isSet(visited, "resume") {
		cfg.ResumeFile = cli.ResumeFile
	}
	if isSet(visited, "resume-every") {
		cfg.ResumeEvery = cli.ResumeEvery
	}
	if isSet(visited, "verbs") {
		cfg.Verbs = ParseCSV(cli.Verbs)
	}
	if isSet(visited, "auto-verb") {
		cfg.AutoVerbs = cli.AutoVerbs
	}
	if isSet(visited, "sf") {
		if err := applyRanges(&cfg.StopOnStatus, cli.StopOnStatus); err != nil {
			return err
		}
	}
	if isSet(visited, "se") {
		cfg.StopOnErrors = cli.StopOnErrors
	}
	if isSet(visited, "sa") {
		cfg.StopOnMatches = cli.StopOnMatches
	}
	if isSet(visited, "bypass-budget") {
		cfg.BypassBudget = cli.BypassBudget
	}
	if isSet(visited, "bypass-ratio") {
		cfg.BypassRatioLimit = cli.BypassRatioLimit
	}
	if isSet(visited, "waf-adaptive") {
		cfg.WAFAdaptive = cli.WAFAdaptive
	}
	if isSet(visited, "waf-threshold") {
		cfg.WAFSlowdownThreshold = cli.WAFSlowdownThreshold
	}
	if isSet(visited, "waf-factor") {
		cfg.WAFSlowdownFactor = cli.WAFSlowdownFactor
	}
	if isSet(visited, "jitter") {
		cfg.JitterProfile = cli.JitterProfile
	}
	if isSet(visited, "jitter-threshold") {
		cfg.JitterThresholdMS = cli.JitterThresholdMS
	}
	if isSet(visited, "jitter-factor") {
		cfg.JitterFactor = cli.JitterFactor
	}
	if isSet(visited, "bau", "basicAuthUser") {
		cfg.BasicAuthUser = cli.BasicAuthUser
	}
	if isSet(visited, "bap", "basicAuthPass") {
		cfg.BasicAuthPass = cli.BasicAuthPass
	}
	if isSet(visited, "safe") {
		cfg.SafeMode = cli.SafeMode
	}
	if isSet(visited, "allow-hosts") {
		cfg.AllowedHosts = ParseCSV(cli.AllowedHosts)
	}
	if isSet(visited, "exclude-paths") {
		cfg.ExcludePaths = loadPathList(cli.ExcludePaths)
	}
	if isSet(visited, "q", "quiet") {
		cfg.Quiet = cli.Quiet
	}
	if isSet(visited, "dump") {
		cfg.DumpResponses = cli.DumpResponses
	}
	if isSet(visited, "dump-dir") {
		cfg.DumpDir = cli.DumpDir
	}
	if isSet(visited, "retry") {
		cfg.RetryCount = cli.RetryCount
	}
	if isSet(visited, "retry-backoff-ms") {
		cfg.RetryBackoff = time.Duration(cli.RetryBackoffMS) * time.Millisecond
	}
	if isSet(visited, "timeout") {
		cfg.Timeout = time.Duration(cli.TimeoutSeconds) * time.Second
	}
	if isSet(visited, "g", "generate_payload") {
		cfg.GeneratePayload = cli.GeneratePayload
	}
	if isSet(visited, "gpl", "generate_payload_length") {
		cfg.GeneratePayloadLength = cli.GeneratePayloadLength
	}
	if isSet(visited, "modules") {
		cfg.ModuleConfig.Modules = ParseCSV(cli.Modules)
	}
	if isSet(visited, "ai-prompt") {
		cfg.ModuleConfig.AIPrompt = cli.AIPrompt
	}
	if isSet(visited, "ai-provider") {
		cfg.ModuleConfig.AIProvider = strings.TrimSpace(strings.ToLower(cli.AIProvider))
	}
	if isSet(visited, "ai-endpoint") {
		cfg.ModuleConfig.AIEndpoint = strings.TrimSpace(cli.AIEndpoint)
	}
	if isSet(visited, "ai-model") {
		cfg.ModuleConfig.AIModel = strings.TrimSpace(cli.AIModel)
	}
	if isSet(visited, "ai-max-tokens") {
		cfg.ModuleConfig.AIMaxTokens = cli.AIMaxTokens
	}
	if isSet(visited, "maxtime") {
		cfg.MaxTime = cli.MaxTime
	}
	if isSet(visited, "maxtime-job") {
		cfg.MaxTimeJob = cli.MaxTimeJob
	}
	if isSet(visited, "recursion-strategy") {
		cfg.RecursionStrategy = strings.ToLower(strings.TrimSpace(cli.RecursionStrategy))
	}
	if isSet(visited, "http2") {
		cfg.UseHTTP2 = cli.UseHTTP2
	}
	if isSet(visited, "vhost") {
		cfg.VHostFuzz = cli.VHostFuzz
	}
	if isSet(visited, "audit-log") {
		cfg.AuditLogPath = cli.AuditLogPath
	}
	if isSet(visited, "audit-max-body") {
		cfg.AuditMaxBodySize = cli.AuditMaxBodySize
	}
	if isSet(visited, "enqueue-module-urls") {
		cfg.ModuleConfig.EnqueueModuleUrls = strings.TrimSpace(cli.EnqueueModuleUrls)
	}
	if isSet(visited, "extracted-urls-file") {
		cfg.ModuleConfig.ExtractedURLsFile = strings.TrimSpace(cli.ExtractedURLsFile)
	}
	if isSet(visited, "insecure", "k") {
		cfg.InsecureSkipVerify = cli.InsecureSkipVerify
	}
	if isSet(visited, "login-url") {
		cfg.LoginURL = strings.TrimSpace(cli.LoginURL)
	}
	if isSet(visited, "login-method") {
		cfg.LoginMethod = strings.TrimSpace(cli.LoginMethod)
		if cfg.LoginMethod == "" {
			cfg.LoginMethod = "POST"
		}
	}
	if isSet(visited, "login-user") {
		cfg.LoginUser = strings.TrimSpace(cli.LoginUser)
	}
	if isSet(visited, "login-pass") {
		cfg.LoginPass = strings.TrimSpace(cli.LoginPass)
	}
	if isSet(visited, "login-body") {
		cfg.LoginBody = strings.TrimSpace(cli.LoginBody)
	}
	if isSet(visited, "login-content-type") {
		cfg.LoginContentType = strings.TrimSpace(cli.LoginContentType)
	}
	if isSet(visited, "enc") {
		cfg.Encoders = ParseEncoders(cli.Encoders)
	}
	if isSet(visited, "explore-ai") {
		cfg.ExploreAI = cli.ExploreAI
	}
	if isSet(visited, "explore-ai-wordlists-dir") {
		cfg.ExploreAIWordlistsDir = strings.TrimSpace(cli.ExploreAIWordlistsDir)
	}
	if isSet(visited, "explore-ai-wordlist") {
		cfg.ExploreAIWordlistMap = ParseExploreAIWordlistMap(cli.ExploreAIWordlistMap)
	}
	if isSet(visited, "explore-ai-profile") {
		cfg.ExploreAIProfile = strings.TrimSpace(strings.ToLower(cli.ExploreAIProfile))
	}
	if isSet(visited, "explore-ai-no-cache") {
		cfg.ExploreAINoCache = cli.ExploreAINoCache
	}
	if isSet(visited, "explore-ai-provider") {
		cfg.ExploreAIProvider = strings.TrimSpace(strings.ToLower(cli.ExploreAIProvider))
	}
	if isSet(visited, "explore-ai-endpoint") {
		cfg.ExploreAIEndpoint = strings.TrimSpace(cli.ExploreAIEndpoint)
	}
	if isSet(visited, "explore-ai-model") {
		cfg.ExploreAIModel = strings.TrimSpace(cli.ExploreAIModel)
	}
	if isSet(visited, "explore-ai-max-tokens") {
		cfg.ExploreAIMaxTokens = cli.ExploreAIMaxTokens
	}
	return nil
}

// isSet returns true if any of the given flag names was set by the user (visited).
func isSet(m map[string]bool, names ...string) bool {
	for _, n := range names {
		if m[n] {
			return true
		}
	}
	return false
}
