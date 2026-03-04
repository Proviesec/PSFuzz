package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// requestDataForSave returns the value to persist for "data": when saving config.
// If body was loaded from a file (RequestDataPath set), we save "@path" so reloading reads the file again.
func requestDataForSave(cfg *Config) string {
	if cfg.RequestDataPath != "" {
		return "@" + cfg.RequestDataPath
	}
	return cfg.RequestData
}

// Save writes cfg to path as indented JSON. Credentials and login body are not persisted.
func Save(cfg *Config, path string) error {
	payload := map[string]any{
		"url":                      cfg.URL,
		"urlList":                  cfg.URLs,
		"dirlist":                  cfg.Wordlist,
		"extensions":               cfg.Extensions,
		"defaultExtensions":        cfg.UseDefaultExtensions,
		"inputMode":                cfg.InputMode,
		"wordlistCase":             cfg.RandomizeWordlistCase,
		"ignoreWordlistComments":   cfg.IgnoreWordlistComments,
		"autoWildcard":             cfg.AutoWildcard,
		"maxResponseSize":          cfg.MaxResponseSize,
		"minResponseSize":          cfg.MinResponseSize,
		"concurrency":              cfg.Concurrency,
		"depth":                    cfg.Depth,
		"recursionSmart":           cfg.RecursionSmart,
		"recursionStatusCodes":     rangesToString(cfg.RecursionStatus),
		"redirect":                 cfg.FollowRedirects,
		"output":                   cfg.OutputBase,
		"outputFormat":             cfg.OutputFormat,
		"throttleRate":             cfg.ThrottleRPS,
		"delay":                    formatDelay(cfg.DelayMin, cfg.DelayMax),
		"filterStatusCode":         rangesToString(cfg.FilterStatus),
		"filterStatusCodeNot":      rangesToString(cfg.FilterStatusNot),
		"filterLength":             rangesToString(cfg.FilterLength),
		"filterLengthNot":         rangesToString(cfg.FilterLengthNot),
		"filterWords":              rangesToString(cfg.FilterWords),
		"filterWordsNot":          rangesToString(cfg.FilterWordsNot),
		"filterLines":             rangesToString(cfg.FilterLines),
		"filterLinesNot":          rangesToString(cfg.FilterLinesNot),
		"filterTime":              rangesToString(cfg.FilterTime),
		"filterTimeNot":           rangesToString(cfg.FilterTimeNot),
		"filterMatchWord":         cfg.FilterMatchWord,
		"filterMatchRegex":        regexToString(cfg.FilterMatchRegex),
		"filterMatchRegexNot":     regexToString(cfg.FilterMatchRegexNot),
		"filterMatchRegexTextOnly": cfg.FilterRegexTextOnly,
		"filterContentType":       strings.Join(cfg.FilterContentTypes, ","),
		"filterDuplicates":        cfg.FilterDuplicates,
		"duplicateThreshold":      cfg.DuplicateThreshold,
		"nearDuplicates":          cfg.NearDuplicates,
		"nearDuplicateLenBucket":   cfg.NearDuplicateLenBucket,
		"nearDuplicateWordBucket":  cfg.NearDuplicateWordBucket,
		"nearDuplicateLineBucket":  cfg.NearDuplicateLineBucket,
		"blockWords":               strings.Join(cfg.BlockWords, ","),
		"interestingStrings":       strings.Join(cfg.InterestingStrings, ","),
		"showStatus":               cfg.ShowStatus,
		"onlydomains":              cfg.OnlyDomains,
		"checkBackslash":           cfg.CheckBackslash,
		"bypass":                   cfg.Bypass,
		"bypassTooManyRequests":    cfg.BypassTooManyRequests,
		"filterTestLength":         cfg.FilterTestLength,
		"filterWrongStatus200":     cfg.FilterWrongStatus200,
		"filterWrongSubdomain":     cfg.FilterWrongSubdomain,
		"filterPossible404":        cfg.FilterPossible404,
		"autoCalibrate":            cfg.AutoCalibrate,
		"autoCalibrateN":           cfg.AutoCalibrateN,
		"requestAddHeader":         mapToString(cfg.RequestHeaders, ":"),
		"cookies":                  mapToString(cfg.RequestCookies, "="),
		"requestAddAgent":          cfg.RequestUserAgent,
		"randomUserAgent":          cfg.RandomUserAgent,
		"method":                   cfg.RequestMethod,
		"data":                     requestDataForSave(cfg),
		"proxy":                    cfg.Proxy,
		"requestFile":              cfg.RequestFile,
		"replayProxy":              cfg.ReplayProxy,
		"replayOnMatch":            cfg.ReplayOnMatch,
		"requestProto":             cfg.RequestProto,
		"proxyUser":                cfg.ProxyUser,
		"proxyPass":                cfg.ProxyPass,
		"resumeFile":               cfg.ResumeFile,
		"resumeEvery":              cfg.ResumeEvery,
		"verbs":                    strings.Join(cfg.Verbs, ","),
		"autoVerbs":                cfg.AutoVerbs,
		"stopOnStatus":             rangesToString(cfg.StopOnStatus),
		"stopOnErrors":             cfg.StopOnErrors,
		"stopOnMatches":            cfg.StopOnMatches,
		"bypassBudget":             cfg.BypassBudget,
		"bypassRatioLimit":         cfg.BypassRatioLimit,
		"wafAdaptive":              cfg.WAFAdaptive,
		"wafSlowdownThreshold":     cfg.WAFSlowdownThreshold,
		"wafSlowdownFactor":        cfg.WAFSlowdownFactor,
		"jitterProfile":            cfg.JitterProfile,
		"jitterThresholdMs":        cfg.JitterThresholdMS,
		"jitterFactor":             cfg.JitterFactor,
		"basicAuthUser":            cfg.BasicAuthUser,
		"basicAuthPass":            cfg.BasicAuthPass,
		"safeMode":                 cfg.SafeMode,
		"allowedHosts":             strings.Join(cfg.AllowedHosts, ","),
		"excludePaths":             strings.Join(cfg.ExcludePaths, ","),
		"quiet":                    cfg.Quiet,
		"dumpResponses":            cfg.DumpResponses,
		"dumpDir":                  cfg.DumpDir,
		"retryCount":               cfg.RetryCount,
		"retryBackoffMs":           int(cfg.RetryBackoff / time.Millisecond),
		"generate_payload":         cfg.GeneratePayload,
		"generate_payload_length":  cfg.GeneratePayloadLength,
		"aiPrompt":                 cfg.ModuleConfig.AIPrompt,
		"aiProvider":               cfg.ModuleConfig.AIProvider,
		"aiEndpoint":               cfg.ModuleConfig.AIEndpoint,
		"aiModel":                  cfg.ModuleConfig.AIModel,
		"aiMaxTokens":              cfg.ModuleConfig.AIMaxTokens,
		"modules":                  strings.Join(cfg.ModuleConfig.Modules, ","),
		"maxtime":                  cfg.MaxTime,
		"maxtimeJob":               cfg.MaxTimeJob,
		"recursionStrategy":        cfg.RecursionStrategy,
		"http2":                    cfg.UseHTTP2,
		"vhost":                    cfg.VHostFuzz,
		"auditLog":                 cfg.AuditLogPath,
		"auditMaxBodySize":         cfg.AuditMaxBodySize,
		"enqueueModuleUrls":        cfg.ModuleConfig.EnqueueModuleUrls,
		"extractedUrlsFile":        cfg.ModuleConfig.ExtractedURLsFile,
		"insecureSkipVerify":       cfg.InsecureSkipVerify,
		"loginUrl":                 cfg.LoginURL,
		"loginMethod":              cfg.LoginMethod,
		"loginUser":                "", // never persist credentials to config file
		"loginPass":                "",
		"loginBody":                "", // may contain secrets
		"loginContentType":         cfg.LoginContentType,
		"encoders":                 encodersToString(cfg.Encoders),
		"exploreAI":                cfg.ExploreAI,
		"exploreAIWordlistsDir":    cfg.ExploreAIWordlistsDir,
		"exploreAIWordlistMap":     cfg.ExploreAIWordlistMap,
		"exploreAIProfile":         cfg.ExploreAIProfile,
		"exploreAINoCache":         cfg.ExploreAINoCache,
		"exploreAIProvider":        cfg.ExploreAIProvider,
		"exploreAIEndpoint":        cfg.ExploreAIEndpoint,
		"exploreAIModel":           cfg.ExploreAIModel,
		"exploreAIMaxTokens":       cfg.ExploreAIMaxTokens,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func encodersToString(m map[string][]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		chain := m[k]
		if len(chain) == 0 {
			continue
		}
		parts = append(parts, k+":"+strings.Join(chain, ","))
	}
	return strings.Join(parts, ";")
}

func mapToString(m map[string]string, sep string) string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, k+sep+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func rangesToString(ranges []Range) string {
	if len(ranges) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		if r.Min == r.Max {
			parts = append(parts, strconv.Itoa(r.Min))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", r.Min, r.Max))
		}
	}
	return strings.Join(parts, ",")
}

func regexToString(rx *regexp.Regexp) string {
	if rx == nil {
		return ""
	}
	return rx.String()
}

func formatDelay(min, max time.Duration) string {
	if min == 0 && max == 0 {
		return ""
	}
	if min == max {
		return min.String()
	}
	return fmt.Sprintf("%s-%s", min, max)
}
