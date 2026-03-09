package config

import (
	"time"

	"github.com/Proviesec/PSFuzz/internal/modules"
)

// defaultConfig returns a Config with all fields set to their default values.
func defaultConfig() *Config {
	return &Config{
		Wordlist:                "default",
		Extensions:              nil,
		UseDefaultExtensions:    false,
		InputMode:               "clusterbomb",
		IgnoreWordlistComments:  false,
		AutoWildcard:            false,
		MaxResponseSize:         0,
		MinResponseSize:         0,
		SaveConfigPath:          "",
		Concurrency:             40,
		Depth:                   0,
		RecursionSmart:          true,
		RecursionStatus:         []StatusRange{{Min: 200, Max: 200}, {Min: 301, Max: 302}, {Min: 403, Max: 403}},
		FollowRedirects:         false,
		OutputBase:              "output",
		OutputFormat:            "txt",
		Timeout:                 30 * time.Second,
		DelayMin:                0,
		DelayMax:                0,
		ThrottleRPS:             0,
		DuplicateThreshold:      10,
		FilterContentTypes:      nil,
		FilterStatus:            defaultMatcher(),
		ShowStatus:              false,
		NearDuplicates:          false,
		NearDuplicateLenBucket:  20,
		NearDuplicateWordBucket: 5,
		NearDuplicateLineBucket: 5,
		BlockWords:              nil,
		InterestingStrings:      nil,
		OnlyDomains:             false,
		CheckBackslash:          false,
		Bypass:                  false,
		BypassTooManyRequests:   false,
		FilterTestLength:        false,
		FilterWrongStatus200:    false,
		FilterWrongSubdomain:    false,
		FilterPossible404:       false,
		AutoCalibrate:           false,
		AutoCalibrateN:          2,
		RequestHeaders:          map[string]string{},
		RequestCookies:          map[string]string{},
		RequestUserAgent:        "PSFuzz/" + Version,
		RandomUserAgent:         false,
		RandomizeWordlistCase:   "",
		RequestMethod:           "",
		RequestData:             "",
		RequestDataPath:         "",
		Proxy:                   "",
		RequestFile:             "",
		ReplayProxy:             "",
		ReplayOnMatch:           true,
		RequestProto:            "",
		ProxyUser:               "",
		ProxyPass:               "",
		ResumeFile:              "",
		ResumeEvery:             0,
		Verbs:                   nil,
		AutoVerbs:               false,
		StopOnStatus:            nil,
		StopOnErrors:            false,
		StopOnMatches:           0,
		BypassBudget:            3,
		BypassRatioLimit:        0.4,
		WAFAdaptive:             false,
		WAFSlowdownThreshold:    50,
		WAFSlowdownFactor:       2.0,
		JitterProfile:           false,
		JitterThresholdMS:       800,
		JitterFactor:            1.0,
		SafeMode:                true,
		ExcludePaths:            nil,
		Quiet:                   false,
		DumpResponses:           false,
		DumpDir:                 "",
		RetryCount:              2,
		RetryBackoff:            500 * time.Millisecond,
		GeneratePayload:         false,
		GeneratePayloadLength:   20000,
		ModuleConfig:            modules.Config{},
		MaxTime:                 0,
		MaxTimeJob:              0,
		RecursionStrategy:       "default",
		UseHTTP2:                false,
		VHostFuzz:               false,
		AuditLogPath:            "",
		AuditMaxBodySize:        0,
		InsecureSkipVerify:     false,
		LoginURL:                "",
		LoginMethod:             "POST",
		LoginUser:               "",
		LoginPass:               "",
		LoginBody:               "",
		LoginContentType:        "application/x-www-form-urlencoded",
		Encoders:                nil,
		ExploreAI:               false,
		ExploreAIWordlistsDir:   "",
		ExploreAIWordlistMap:    make(map[string]string),
		ExploreAIProfile:        "balanced",
		ExploreAINoCache:        false,
		ExploreAIProvider:       "openai",
		ExploreAIEndpoint:       "",
		ExploreAIModel:          "",
		ExploreAIMaxTokens:      0,
	}
}

// defaultMatcher returns the default status code filter (match set): 200-299, 301, 302, 307, 401, 403, 405, 500.
func defaultMatcher() []StatusRange {
	return []StatusRange{
		{Min: 200, Max: 299}, // 2xx
		{Min: 301, Max: 302},
		{Min: 307, Max: 307},
		{Min: 401, Max: 401},
		{Min: 403, Max: 403},
		{Min: 405, Max: 405},
		{Min: 500, Max: 500},
	}
}

// defaultExtensions returns the default set of extensions used when -ext-defaults is set.
func defaultExtensions() []string {
	return []string{".yml", ".yaml", ".php", ".aspx", ".jsp", ".html", ".js"}
}

// defaultVerbs returns the HTTP methods used when -auto-verb is set and no -verbs given.
func defaultVerbs() []string {
	return []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
}
