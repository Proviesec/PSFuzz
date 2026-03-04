package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// applyFileConfig reads a JSON config file and applies all set fields to cfg.
// Used when -cf / -configfile is given; CLI flags applied later override file values.
func applyFileConfig(cfg *Config, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	var f fileConfig
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := applyFileTargetAndWordlist(cfg, &f); err != nil {
		return err
	}
	applyFileOutputAndRate(cfg, &f)
	if err := applyFileFilter(cfg, &f); err != nil {
		return err
	}
	if err := applyFileRequest(cfg, &f); err != nil {
		return err
	}
	applyFileExploreAI(cfg, &f)
	return nil
}

func applyFileTargetAndWordlist(cfg *Config, f *fileConfig) error {
	if f.URL != nil {
		cfg.URL = *f.URL
	}
	if f.URLList != nil {
		cfg.URLs = loadURLList(*f.URLList)
	}
	if f.Dirlist != nil {
		cfg.Wordlist = *f.Dirlist
	}
	if f.Extensions != nil {
		cfg.Extensions = normalizeExtensions(ParseCSV(*f.Extensions))
	}
	if f.UseDefaultExtensions != nil {
		cfg.UseDefaultExtensions = *f.UseDefaultExtensions
	}
	if f.InputMode != nil {
		cfg.InputMode = strings.ToLower(strings.TrimSpace(*f.InputMode))
	}
	if f.IgnoreWordlistComments != nil {
		cfg.IgnoreWordlistComments = *f.IgnoreWordlistComments
	}
	if f.AutoWildcard != nil {
		cfg.AutoWildcard = *f.AutoWildcard
	}
	if f.MaxResponseSize != nil {
		cfg.MaxResponseSize = *f.MaxResponseSize
	}
	if f.MinResponseSize != nil {
		cfg.MinResponseSize = *f.MinResponseSize
	}
	if f.SaveConfigPath != nil {
		cfg.SaveConfigPath = *f.SaveConfigPath
	}
	if f.Concurrency != nil {
		cfg.Concurrency = *f.Concurrency
	}
	if f.Depth != nil {
		cfg.Depth = *f.Depth
	}
	if f.RecursionLevel != nil {
		cfg.Depth = *f.RecursionLevel
	}
	if f.RecursionSmart != nil {
		cfg.RecursionSmart = *f.RecursionSmart
	}
	if f.RecursionStatusCode != nil {
		if err := applyStatusRangesFromPtr(&cfg.RecursionStatus, f.RecursionStatusCode); err != nil {
			return err
		}
	}
	if f.Redirect != nil {
		cfg.FollowRedirects = *f.Redirect
	}
	return nil
}

func applyFileOutputAndRate(cfg *Config, f *fileConfig) {
	if f.Output != nil {
		cfg.OutputBase = strings.TrimSuffix(*f.Output, ".txt")
	}
	if f.OutputFormat != nil {
		cfg.OutputFormat = *f.OutputFormat
	}
	if f.ThrottleRate != nil {
		cfg.ThrottleRPS = *f.ThrottleRate
	}
	if f.Delay != nil {
		if min, max, err := parseDelay(*f.Delay); err == nil {
			cfg.DelayMin = min
			cfg.DelayMax = max
		}
	}
}

func applyFileFilter(cfg *Config, f *fileConfig) error {
	if f.FilterStatusCode != nil {
		if err := applyStatusRangesFromPtr(&cfg.FilterStatus, f.FilterStatusCode); err != nil {
			return err
		}
	}
	if f.FilterStatusCodeNot != nil {
		if err := applyStatusRangesFromPtr(&cfg.FilterStatusNot, f.FilterStatusCodeNot); err != nil {
			return err
		}
	}
	if f.FilterLength != nil {
		if err := applyRangesFromPtr(&cfg.FilterLength, f.FilterLength); err != nil {
			return err
		}
	}
	if f.FilterLengthNot != nil {
		if err := applyRangesFromPtr(&cfg.FilterLengthNot, f.FilterLengthNot); err != nil {
			return err
		}
	}
	if f.FilterWords != nil {
		if err := applyRangesFromPtr(&cfg.FilterWords, f.FilterWords); err != nil {
			return err
		}
	}
	if f.FilterWordsNot != nil {
		if err := applyRangesFromPtr(&cfg.FilterWordsNot, f.FilterWordsNot); err != nil {
			return err
		}
	}
	if f.FilterLines != nil {
		if err := applyRangesFromPtr(&cfg.FilterLines, f.FilterLines); err != nil {
			return err
		}
	}
	if f.FilterLinesNot != nil {
		if err := applyRangesFromPtr(&cfg.FilterLinesNot, f.FilterLinesNot); err != nil {
			return err
		}
	}
	if f.FilterTime != nil {
		if err := applyRangesFromPtr(&cfg.FilterTime, f.FilterTime); err != nil {
			return err
		}
	}
	if f.FilterTimeNot != nil {
		if err := applyRangesFromPtr(&cfg.FilterTimeNot, f.FilterTimeNot); err != nil {
			return err
		}
	}
	if f.FilterMatchWord != nil {
		cfg.FilterMatchWord = *f.FilterMatchWord
	}
	if f.FilterMatchRegex != nil {
		rx, err := regexp.Compile(*f.FilterMatchRegex)
		if err != nil {
			return fmt.Errorf("compile filterMatchRegex: %w", err)
		}
		cfg.FilterMatchRegex = rx
	}
	if f.FilterMatchRegexNot != nil {
		rx, err := regexp.Compile(*f.FilterMatchRegexNot)
		if err != nil {
			return fmt.Errorf("compile filterMatchRegexNot: %w", err)
		}
		cfg.FilterMatchRegexNot = rx
	}
	if f.FilterRegexTextOnly != nil {
		cfg.FilterRegexTextOnly = *f.FilterRegexTextOnly
	}
	if f.FilterContentType != nil {
		cfg.FilterContentTypes = ParseCSV(*f.FilterContentType)
	}
	if f.FilterDuplicates != nil {
		cfg.FilterDuplicates = *f.FilterDuplicates
	}
	if f.DuplicateThreshold != nil {
		cfg.DuplicateThreshold = *f.DuplicateThreshold
	}
	if f.NearDuplicates != nil {
		cfg.NearDuplicates = *f.NearDuplicates
	}
	if f.NearDuplicateLenBucket != nil {
		cfg.NearDuplicateLenBucket = *f.NearDuplicateLenBucket
	}
	if f.NearDuplicateWordBucket != nil {
		cfg.NearDuplicateWordBucket = *f.NearDuplicateWordBucket
	}
	if f.NearDuplicateLineBucket != nil {
		cfg.NearDuplicateLineBucket = *f.NearDuplicateLineBucket
	}
	if f.BlockWords != nil {
		cfg.BlockWords = loadWordList(*f.BlockWords)
	}
	if f.InterestingStrings != nil {
		cfg.InterestingStrings = loadWordList(*f.InterestingStrings)
	}
	if f.ShowStatus != nil {
		cfg.ShowStatus = *f.ShowStatus
	}
	if f.OnlyDomains != nil {
		cfg.OnlyDomains = *f.OnlyDomains
	}
	if f.CheckBackslash != nil {
		cfg.CheckBackslash = *f.CheckBackslash
	}
	if f.Bypass != nil {
		cfg.Bypass = *f.Bypass
	}
	if f.BypassTooManyReq != nil {
		cfg.BypassTooManyRequests = *f.BypassTooManyReq
	}
	if f.FilterTestLength != nil {
		cfg.FilterTestLength = *f.FilterTestLength
	}
	if f.FilterWrongStatus200 != nil {
		cfg.FilterWrongStatus200 = *f.FilterWrongStatus200
	}
	if f.FilterWrongSubdomain != nil {
		cfg.FilterWrongSubdomain = *f.FilterWrongSubdomain
	}
	if f.FilterPossible404 != nil {
		cfg.FilterPossible404 = *f.FilterPossible404
	}
	if f.AutoCalibrate != nil {
		cfg.AutoCalibrate = *f.AutoCalibrate
	}
	if f.AutoCalibrateN != nil {
		cfg.AutoCalibrateN = *f.AutoCalibrateN
	}
	return nil
}

func applyFileRequest(cfg *Config, f *fileConfig) error {
	if f.Headers != nil {
		cfg.RequestHeaders = ParseKV(*f.Headers, ":")
	}
	if f.Cookies != nil {
		cfg.RequestCookies = ParseKV(*f.Cookies, "=")
	}
	if f.UserAgent != nil {
		cfg.RequestUserAgent = *f.UserAgent
	}
	if f.RandomUserAgent != nil {
		cfg.RandomUserAgent = *f.RandomUserAgent
	}
	if f.RandomizeWordlistCase != nil {
		cfg.RandomizeWordlistCase = strings.ToLower(strings.TrimSpace(*f.RandomizeWordlistCase))
	}
	if f.Method != nil {
		cfg.RequestMethod = *f.Method
	}
	if f.Data != nil {
		if err := setRequestDataFromSpec(cfg, strings.TrimSpace(*f.Data)); err != nil {
			return err
		}
	}
	if f.Proxy != nil {
		cfg.Proxy = *f.Proxy
	}
	if f.RequestFile != nil {
		cfg.RequestFile = *f.RequestFile
	}
	if f.ReplayProxy != nil {
		cfg.ReplayProxy = *f.ReplayProxy
	}
	if f.ReplayOnMatch != nil {
		cfg.ReplayOnMatch = *f.ReplayOnMatch
	}
	if f.RequestProto != nil {
		cfg.RequestProto = *f.RequestProto
	}
	if f.ProxyUser != nil {
		cfg.ProxyUser = *f.ProxyUser
	}
	if f.ProxyPass != nil {
		cfg.ProxyPass = *f.ProxyPass
	}
	if f.ResumeFile != nil {
		cfg.ResumeFile = *f.ResumeFile
	}
	if f.ResumeEvery != nil {
		cfg.ResumeEvery = *f.ResumeEvery
	}
	if f.Verbs != nil {
		cfg.Verbs = ParseCSV(*f.Verbs)
	}
	if f.AutoVerbs != nil {
		cfg.AutoVerbs = *f.AutoVerbs
	}
	if f.StopOnStatus != nil {
		if err := applyRangesFromPtr(&cfg.StopOnStatus, f.StopOnStatus); err != nil {
			return err
		}
	}
	if f.StopOnErrors != nil {
		cfg.StopOnErrors = *f.StopOnErrors
	}
	if f.StopOnMatches != nil {
		cfg.StopOnMatches = *f.StopOnMatches
	}
	if f.BypassBudget != nil {
		cfg.BypassBudget = *f.BypassBudget
	}
	if f.BypassRatioLimit != nil {
		cfg.BypassRatioLimit = *f.BypassRatioLimit
	}
	if f.WAFAdaptive != nil {
		cfg.WAFAdaptive = *f.WAFAdaptive
	}
	if f.WAFSlowdownThreshold != nil {
		cfg.WAFSlowdownThreshold = *f.WAFSlowdownThreshold
	}
	if f.WAFSlowdownFactor != nil {
		cfg.WAFSlowdownFactor = *f.WAFSlowdownFactor
	}
	if f.JitterProfile != nil {
		cfg.JitterProfile = *f.JitterProfile
	}
	if f.JitterThresholdMS != nil {
		cfg.JitterThresholdMS = *f.JitterThresholdMS
	}
	if f.JitterFactor != nil {
		cfg.JitterFactor = *f.JitterFactor
	}
	if f.BasicAuthUser != nil {
		cfg.BasicAuthUser = *f.BasicAuthUser
	}
	if f.BasicAuthPass != nil {
		cfg.BasicAuthPass = *f.BasicAuthPass
	}
	if f.SafeMode != nil {
		cfg.SafeMode = *f.SafeMode
	}
	if f.AllowedHosts != nil {
		cfg.AllowedHosts = ParseCSV(*f.AllowedHosts)
	}
	if f.ExcludePaths != nil {
		cfg.ExcludePaths = loadPathList(*f.ExcludePaths)
	}
	if f.Quiet != nil {
		cfg.Quiet = *f.Quiet
	}
	if f.DumpResponses != nil {
		cfg.DumpResponses = *f.DumpResponses
	}
	if f.DumpDir != nil {
		cfg.DumpDir = *f.DumpDir
	}
	if f.RetryCount != nil {
		cfg.RetryCount = *f.RetryCount
	}
	if f.RetryBackoffMS != nil {
		cfg.RetryBackoff = time.Duration(*f.RetryBackoffMS) * time.Millisecond
	}
	if f.GeneratePayload != nil {
		cfg.GeneratePayload = *f.GeneratePayload
	}
	if f.GeneratePayloadLen != nil {
		cfg.GeneratePayloadLength = *f.GeneratePayloadLen
	}
	if f.AIPrompt != nil {
		cfg.ModuleConfig.AIPrompt = *f.AIPrompt
	}
	if f.AIProvider != nil {
		cfg.ModuleConfig.AIProvider = strings.TrimSpace(strings.ToLower(*f.AIProvider))
	}
	if f.AIEndpoint != nil {
		cfg.ModuleConfig.AIEndpoint = strings.TrimSpace(*f.AIEndpoint)
	}
	if f.AIModel != nil {
		cfg.ModuleConfig.AIModel = strings.TrimSpace(*f.AIModel)
	}
	if f.AIMaxTokens != nil {
		cfg.ModuleConfig.AIMaxTokens = *f.AIMaxTokens
	}
	if f.Modules != nil {
		cfg.ModuleConfig.Modules = ParseCSV(*f.Modules)
	}
	if f.MaxTime != nil {
		cfg.MaxTime = *f.MaxTime
	}
	if f.MaxTimeJob != nil {
		cfg.MaxTimeJob = *f.MaxTimeJob
	}
	if f.RecursionStrategy != nil {
		cfg.RecursionStrategy = strings.ToLower(strings.TrimSpace(*f.RecursionStrategy))
	}
	if f.UseHTTP2 != nil {
		cfg.UseHTTP2 = *f.UseHTTP2
	}
	if f.VHostFuzz != nil {
		cfg.VHostFuzz = *f.VHostFuzz
	}
	if f.AuditLogPath != nil {
		cfg.AuditLogPath = *f.AuditLogPath
	}
	if f.AuditMaxBodySize != nil {
		cfg.AuditMaxBodySize = *f.AuditMaxBodySize
	}
	if f.EnqueueModuleUrls != nil {
		cfg.ModuleConfig.EnqueueModuleUrls = strings.TrimSpace(*f.EnqueueModuleUrls)
	}
	if f.ExtractedURLsFile != nil {
		cfg.ModuleConfig.ExtractedURLsFile = strings.TrimSpace(*f.ExtractedURLsFile)
	}
	if f.InsecureSkipVerify != nil {
		cfg.InsecureSkipVerify = *f.InsecureSkipVerify
	}
	if f.LoginURL != nil {
		cfg.LoginURL = strings.TrimSpace(*f.LoginURL)
	}
	if f.LoginMethod != nil {
		cfg.LoginMethod = strings.TrimSpace(*f.LoginMethod)
		if cfg.LoginMethod == "" {
			cfg.LoginMethod = "POST"
		}
	}
	if f.LoginUser != nil {
		cfg.LoginUser = strings.TrimSpace(*f.LoginUser)
	}
	if f.LoginPass != nil {
		cfg.LoginPass = strings.TrimSpace(*f.LoginPass)
	}
	if f.LoginBody != nil {
		cfg.LoginBody = strings.TrimSpace(*f.LoginBody)
	}
	if f.LoginContentType != nil {
		cfg.LoginContentType = strings.TrimSpace(*f.LoginContentType)
	}
	if f.Encoders != nil {
		cfg.Encoders = ParseEncoders(strings.TrimSpace(*f.Encoders))
	}
	return nil
}

func applyFileExploreAI(cfg *Config, f *fileConfig) {
	if f.ExploreAI != nil {
		cfg.ExploreAI = *f.ExploreAI
	}
	if f.ExploreAIWordlistsDir != nil {
		cfg.ExploreAIWordlistsDir = strings.TrimSpace(*f.ExploreAIWordlistsDir)
	}
	if len(f.ExploreAIWordlistMap) > 0 {
		cfg.ExploreAIWordlistMap = make(map[string]string)
		for k, v := range f.ExploreAIWordlistMap {
			cfg.ExploreAIWordlistMap[strings.TrimSpace(strings.ToLower(k))] = strings.TrimSpace(v)
		}
	}
	if f.ExploreAIProfile != nil {
		cfg.ExploreAIProfile = strings.TrimSpace(strings.ToLower(*f.ExploreAIProfile))
	}
	if f.ExploreAINoCache != nil {
		cfg.ExploreAINoCache = *f.ExploreAINoCache
	}
	if f.ExploreAIProvider != nil {
		cfg.ExploreAIProvider = strings.TrimSpace(strings.ToLower(*f.ExploreAIProvider))
	}
	if f.ExploreAIEndpoint != nil {
		cfg.ExploreAIEndpoint = strings.TrimSpace(*f.ExploreAIEndpoint)
	}
	if f.ExploreAIModel != nil {
		cfg.ExploreAIModel = strings.TrimSpace(*f.ExploreAIModel)
	}
	if f.ExploreAIMaxTokens != nil {
		cfg.ExploreAIMaxTokens = *f.ExploreAIMaxTokens
	}
}
