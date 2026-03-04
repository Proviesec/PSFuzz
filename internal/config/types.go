package config

import (
	"regexp"
	"time"

	"github.com/Proviesec/PSFuzz/internal/modules"
)

const (
	DefaultPayloadURL   = "https://raw.githubusercontent.com/Proviesec/directory-payload-list/main/directory-full-list.txt"
	FavPayloadURL       = "https://raw.githubusercontent.com/Proviesec/directory-files-payload-lists/main/directory-proviesec-favorite-list.txt"
	SubdomainPayloadURL = "https://raw.githubusercontent.com/Proviesec/subdomain_wordlist/main/subdomain_good-large_wordlist.txt"
)

// Valid output format values for OutputFormat.
var validOutputFormats = map[string]bool{
	"txt": true, "json": true, "html": true, "csv": true, "ndjson": true, "ffufjson": true,
}

// Config holds all PSFuzz options: target URLs, wordlist, filters, request settings, modules, and explore-AI options.
type Config struct {
	URL                     string
	URLs                    []string
	Wordlist                string
	Wordlists               []WordlistSpec
	Extensions              []string
	UseDefaultExtensions    bool
	InputMode               string
	IgnoreWordlistComments  bool
	AutoWildcard            bool
	MaxResponseSize         int
	MinResponseSize         int
	SaveConfigPath          string
	Concurrency             int
	Depth                   int
	RecursionSmart          bool
	RecursionStatus         []StatusRange
	FollowRedirects         bool
	OutputBase              string
	OutputFormat            string
	ThrottleRPS             int
	Timeout                 time.Duration
	DelayMin                time.Duration
	DelayMax                time.Duration
	ConfigFilePath          string
	FilterStatus            []StatusRange
	FilterStatusNot         []StatusRange
	FilterLength            []Range
	FilterLengthNot         []Range
	FilterWords             []Range
	FilterWordsNot          []Range
	FilterLines             []Range
	FilterLinesNot          []Range
	FilterTime              []Range
	FilterTimeNot           []Range
	FilterMatchWord         string
	FilterMatchRegex        *regexp.Regexp
	FilterMatchRegexNot     *regexp.Regexp
	FilterRegexTextOnly     bool
	FilterContentTypes      []string
	FilterDuplicates        bool
	DuplicateThreshold      int
	NearDuplicates          bool
	NearDuplicateLenBucket  int
	NearDuplicateWordBucket int
	NearDuplicateLineBucket int
	BlockWords              []string
	InterestingStrings      []string
	ShowStatus              bool
	OnlyDomains             bool
	CheckBackslash          bool
	Bypass                  bool
	BypassTooManyRequests   bool
	FilterTestLength        bool
	FilterWrongStatus200    bool
	FilterWrongSubdomain    bool
	FilterPossible404       bool
	AutoCalibrate           bool
	AutoCalibrateN          int
	RequestHeaders          map[string]string
	RequestCookies          map[string]string
	RequestUserAgent        string
	RandomUserAgent         bool
	RandomizeWordlistCase   string
	RequestMethod           string
	RequestData             string   // inline body template, or content loaded from RequestDataPath
	RequestDataPath         string   // when set, body was loaded from this file (-d @path); saved as "@path" in config
	Proxy                   string
	RequestFile             string
	ReplayProxy             string
	ReplayOnMatch           bool
	RequestProto            string
	ProxyUser               string
	ProxyPass               string
	ResumeFile              string
	ResumeEvery             int
	Verbs                   []string
	AutoVerbs               bool
	StopOnStatus            []Range
	StopOnErrors            bool
	StopOnMatches           int
	BypassBudget            int
	BypassRatioLimit        float64
	WAFAdaptive             bool
	WAFSlowdownThreshold    int
	WAFSlowdownFactor       float64
	JitterProfile           bool
	JitterThresholdMS       int
	JitterFactor            float64
	BasicAuthUser           string
	BasicAuthPass           string
	SafeMode                bool
	AllowedHosts            []string
	ExcludePaths            []string
	Quiet                   bool
	DumpResponses           bool
	DumpDir                 string
	RetryCount              int
	RetryBackoff            time.Duration
	GeneratePayload         bool
	GeneratePayloadLength   int
	ModuleConfig            modules.Config // module-related settings; defined in internal/modules to keep modules separate
	MaxTime                 int            // max scan duration in seconds (0 = disabled)
	MaxTimeJob              int            // max duration per task in seconds (0 = disabled)
	RecursionStrategy       string         // "default" or "greedy"
	UseHTTP2                bool
	VHostFuzz               bool
	AuditLogPath            string // if set, log every request+response to this file (NDJSON)
	AuditMaxBodySize        int    // max request/response body size to store in audit log (0 = no limit)
	InsecureSkipVerify     bool   // skip TLS certificate verification (-insecure)
	LoginURL                string // if set, perform login once and use session cookies for all requests
	LoginMethod             string // HTTP method for login (default POST)
	LoginUser               string // form field username (or use login-body for custom)
	LoginPass               string // form field password
	LoginBody               string // raw body for login (overrides user/pass form); may contain placeholders
	LoginContentType        string // Content-Type for login (default application/x-www-form-urlencoded)
	Encoders                map[string][]string // keyword -> encoder chain (e.g. FUZZ -> [urlencode, base64encode])
	ExploreAI               bool              // if true, probe base URL, fingerprint+headers, call AI backend (openai/ollama/gemini) for wordlist recommendation, then run scan
	ExploreAIWordlistsDir   string            // if set with ExploreAI, wordlist is resolved from this dir (suggested_wordlist or wordlist_type.txt) and scan runs with it
	ExploreAIWordlistMap    map[string]string  // optional: name (e.g. wordpress, typo3) -> path or URL; checked before ExploreAIWordlistsDir
	ExploreAIProfile        string            // quick | balanced | thorough; influences AI suggestion (wordlist size, extensions)
	ExploreAINoCache        bool              // if true, do not read or write Explore AI cache (fresh API call every time)
	ExploreAIProvider       string            // openai | ollama | gemini; which AI backend to use
	ExploreAIEndpoint       string            // optional: override API base URL (e.g. Ollama http://localhost:11434, or proxy for OpenAI)
	ExploreAIModel          string            // optional: model name (e.g. gpt-4o-mini, llama3.1, gemini-1.5-flash)
	ExploreAIMaxTokens      int               // max tokens for Explore AI response (0 = use default 500)
}

// WordlistSpec describes a single wordlist source and its placeholder keyword.
type WordlistSpec struct {
	Keyword string
	Path    string
}

// fileConfig is the JSON shape for -cf config file; pointers mean "omit if not set".
type fileConfig struct {
	URL                     *string  `json:"url"`
	URLList                 *string  `json:"urlList"`
	Dirlist                 *string  `json:"dirlist"`
	Extensions              *string  `json:"extensions"`
	UseDefaultExtensions    *bool    `json:"defaultExtensions"`
	InputMode               *string  `json:"inputMode"`
	IgnoreWordlistComments  *bool    `json:"ignoreWordlistComments"`
	AutoWildcard            *bool    `json:"autoWildcard"`
	MaxResponseSize         *int     `json:"maxResponseSize"`
	MinResponseSize         *int     `json:"minResponseSize"`
	SaveConfigPath          *string  `json:"saveConfig"`
	Concurrency             *int     `json:"concurrency"`
	Depth                   *int     `json:"depth"`
	RecursionLevel          *int     `json:"recursionLevel"`
	RecursionSmart          *bool    `json:"recursionSmart"`
	RecursionStatusCode     *string  `json:"recursionStatusCodes"`
	Redirect                *bool    `json:"redirect"`
	Output                  *string  `json:"output"`
	OutputFormat            *string  `json:"outputFormat"`
	ThrottleRate            *int     `json:"throttleRate"`
	Delay                   *string  `json:"delay"`
	FilterStatusCode        *string  `json:"filterStatusCode"`
	FilterStatusCodeNot     *string  `json:"filterStatusCodeNot"`
	FilterLength            *string  `json:"filterLength"`
	FilterLengthNot         *string  `json:"filterLengthNot"`
	FilterWords             *string  `json:"filterWords"`
	FilterWordsNot          *string  `json:"filterWordsNot"`
	FilterLines             *string  `json:"filterLines"`
	FilterLinesNot          *string  `json:"filterLinesNot"`
	FilterTime              *string  `json:"filterTime"`
	FilterTimeNot           *string  `json:"filterTimeNot"`
	FilterMatchWord         *string  `json:"filterMatchWord"`
	FilterMatchRegex        *string  `json:"filterMatchRegex"`
	FilterMatchRegexNot     *string  `json:"filterMatchRegexNot"`
	FilterRegexTextOnly     *bool    `json:"filterMatchRegexTextOnly"`
	FilterContentType       *string  `json:"filterContentType"`
	FilterDuplicates        *bool    `json:"filterDuplicates"`
	DuplicateThreshold      *int     `json:"duplicateThreshold"`
	NearDuplicates          *bool    `json:"nearDuplicates"`
	NearDuplicateLenBucket  *int     `json:"nearDuplicateLenBucket"`
	NearDuplicateWordBucket *int     `json:"nearDuplicateWordBucket"`
	NearDuplicateLineBucket *int     `json:"nearDuplicateLineBucket"`
	BlockWords              *string  `json:"blockWords"`
	InterestingStrings      *string  `json:"interestingStrings"`
	ShowStatus              *bool    `json:"showStatus"`
	OnlyDomains             *bool    `json:"onlydomains"`
	CheckBackslash          *bool    `json:"checkBackslash"`
	Bypass                  *bool    `json:"bypass"`
	BypassTooManyReq        *bool    `json:"bypassTooManyRequests"`
	FilterTestLength        *bool    `json:"filterTestLength"`
	FilterWrongStatus200    *bool    `json:"filterWrongStatus200"`
	FilterWrongSubdomain    *bool    `json:"filterWrongSubdomain"`
	FilterPossible404       *bool    `json:"filterPossible404"`
	AutoCalibrate           *bool    `json:"autoCalibrate"`
	AutoCalibrateN          *int     `json:"autoCalibrateN"`
	Headers                 *string  `json:"requestAddHeader"`
	Cookies                 *string  `json:"cookies"`
	UserAgent               *string  `json:"requestAddAgent"`
	RandomUserAgent         *bool    `json:"randomUserAgent"`
	RandomizeWordlistCase   *string  `json:"wordlistCase"`
	Method                  *string  `json:"method"`
	Data                    *string  `json:"data"`
	Proxy                   *string  `json:"proxy"`
	RequestFile             *string  `json:"requestFile"`
	ReplayProxy             *string  `json:"replayProxy"`
	ReplayOnMatch           *bool    `json:"replayOnMatch"`
	RequestProto            *string  `json:"requestProto"`
	ProxyUser               *string  `json:"proxyUser"`
	ProxyPass               *string  `json:"proxyPass"`
	ResumeFile              *string  `json:"resumeFile"`
	ResumeEvery             *int     `json:"resumeEvery"`
	Verbs                   *string  `json:"verbs"`
	AutoVerbs               *bool    `json:"autoVerbs"`
	StopOnStatus            *string  `json:"stopOnStatus"`
	StopOnErrors            *bool    `json:"stopOnErrors"`
	StopOnMatches           *int     `json:"stopOnMatches"`
	BypassBudget            *int     `json:"bypassBudget"`
	BypassRatioLimit        *float64 `json:"bypassRatioLimit"`
	WAFAdaptive             *bool    `json:"wafAdaptive"`
	WAFSlowdownThreshold    *int     `json:"wafSlowdownThreshold"`
	WAFSlowdownFactor       *float64 `json:"wafSlowdownFactor"`
	JitterProfile           *bool    `json:"jitterProfile"`
	JitterThresholdMS       *int     `json:"jitterThresholdMs"`
	JitterFactor            *float64 `json:"jitterFactor"`
	BasicAuthUser           *string  `json:"basicAuthUser"`
	BasicAuthPass           *string  `json:"basicAuthPass"`
	SafeMode                *bool    `json:"safeMode"`
	AllowedHosts            *string  `json:"allowedHosts"`
	ExcludePaths            *string  `json:"excludePaths"`
	Quiet                   *bool    `json:"quiet"`
	DumpResponses           *bool    `json:"dumpResponses"`
	DumpDir                 *string  `json:"dumpDir"`
	RetryCount              *int     `json:"retryCount"`
	RetryBackoffMS          *int     `json:"retryBackoffMs"`
	GeneratePayload         *bool    `json:"generate_payload"`
	GeneratePayloadLen      *int     `json:"generate_payload_length"`
	Modules                 *string  `json:"modules"`
	AIPrompt                *string  `json:"aiPrompt"`
	AIProvider              *string  `json:"aiProvider"`
	AIEndpoint              *string  `json:"aiEndpoint"`
	AIModel                 *string  `json:"aiModel"`
	AIMaxTokens             *int     `json:"aiMaxTokens"`
	MaxTime                 *int     `json:"maxtime"`
	MaxTimeJob              *int     `json:"maxtimeJob"`
	RecursionStrategy       *string  `json:"recursionStrategy"`
	UseHTTP2                *bool    `json:"http2"`
	VHostFuzz               *bool    `json:"vhost"`
	AuditLogPath            *string  `json:"auditLog"`
	AuditMaxBodySize        *int     `json:"auditMaxBodySize"`
	EnqueueModuleUrls       *string  `json:"enqueueModuleUrls"`
	ExtractedURLsFile       *string  `json:"extractedUrlsFile"`
	InsecureSkipVerify     *bool    `json:"insecureSkipVerify"`
	LoginURL                *string  `json:"loginUrl"`
	LoginMethod             *string  `json:"loginMethod"`
	LoginUser               *string  `json:"loginUser"`
	LoginPass               *string  `json:"loginPass"`
	LoginBody               *string  `json:"loginBody"`
	LoginContentType        *string  `json:"loginContentType"`
	Encoders                *string  `json:"encoders"`
	ExploreAI               *bool            `json:"exploreAI"`
	ExploreAIWordlistsDir   *string          `json:"exploreAIWordlistsDir"`
	ExploreAIWordlistMap    map[string]string `json:"exploreAIWordlistMap"`
	ExploreAIProfile        *string          `json:"exploreAIProfile"`
	ExploreAINoCache        *bool            `json:"exploreAINoCache"`
	ExploreAIProvider       *string          `json:"exploreAIProvider"`
	ExploreAIEndpoint       *string          `json:"exploreAIEndpoint"`
	ExploreAIModel          *string          `json:"exploreAIModel"`
	ExploreAIMaxTokens      *int            `json:"exploreAIMaxTokens"`
}

// cliConfig holds raw flag values before applying to Config; used by registerFlags and applyCLIConfig.
type cliConfig struct {
	URL                     string
	URLList                 string
	Dirlist                 string
	Extensions              string
	UseDefaultExtensions    bool
	InputMode               string
	IgnoreWordlistComments  bool
	AutoWildcard            bool
	MaxResponseSize         int
	MinResponseSize         int
	SaveConfigPath          string
	Concurrency             int
	Depth                   int
	RecursionSmart          bool
	RecursionStatusCode     string
	Redirect                bool
	Output                  string
	OutputFormat            string
	ThrottleRate            int
	FilterStatusCode        string
	FilterStatusCodeNot     string
	FilterLength            string
	FilterLengthNot         string
	FilterWords             string
	FilterWordsNot          string
	FilterLines             string
	FilterLinesNot          string
	FilterTime              string
	FilterTimeNot           string
	FilterMatchWord         string
	FilterMatchRegex        string
	FilterMatchRegexNot     string
	FilterRegexTextOnly     bool
	FilterContentType       string
	FilterDuplicates        bool
	DuplicateThreshold      int
	NearDuplicates          bool
	NearDuplicateLenBucket  int
	NearDuplicateWordBucket int
	NearDuplicateLineBucket int
	BlockWords              string
	InterestingStrings      string
	ShowStatus              bool
	OnlyDomains             bool
	CheckBackslash          bool
	Bypass                  bool
	BypassTooManyRequests   bool
	FilterTestLength        bool
	FilterWrongStatus200    bool
	FilterWrongSubdomain    bool
	FilterPossible404       bool
	AutoCalibrate           bool
	AutoCalibrateN          int
	Headers                 string
	Cookies                 string
	UserAgent               string
	RandomUserAgent         bool
	RandomizeWordlistCase   string
	Method                  string
	Data                    string
	Proxy                   string
	RequestFile             string
	ReplayProxy             string
	ReplayOnMatch           bool
	RequestProto            string
	ProxyUser               string
	ProxyPass               string
	ResumeFile              string
	ResumeEvery             int
	Verbs                   string
	AutoVerbs               bool
	StopOnStatus            string
	StopOnErrors            bool
	StopOnMatches           int
	BypassBudget            int
	BypassRatioLimit        float64
	WAFAdaptive             bool
	WAFSlowdownThreshold    int
	WAFSlowdownFactor       float64
	JitterProfile           bool
	JitterThresholdMS       int
	JitterFactor            float64
	BasicAuthUser           string
	BasicAuthPass           string
	SafeMode                bool
	AllowedHosts            string
	ExcludePaths            string
	Quiet                   bool
	DumpResponses           bool
	DumpDir                 string
	RetryCount              int
	RetryBackoffMS          int
	ConfigFilePath          string
	Preset                  string
	TimeoutSeconds          int
	Delay                   string
	GeneratePayload         bool
	GeneratePayloadLength   int
	Modules                 string
	AIPrompt                string
	AIProvider              string
	AIEndpoint              string
	AIModel                 string
	AIMaxTokens             int
	MaxTime                 int
	MaxTimeJob              int
	RecursionStrategy       string
	UseHTTP2                bool
	VHostFuzz               bool
	AuditLogPath            string
	AuditMaxBodySize        int
	EnqueueModuleUrls       string
	ExtractedURLsFile        string
	InsecureSkipVerify     bool
	LoginURL                string
	LoginMethod             string
	LoginUser               string
	LoginPass               string
	LoginBody               string
	LoginContentType        string
	Encoders                string
	ExploreAI               bool
	ExploreAIWordlistsDir   string
	ExploreAIWordlistMap    string
	ExploreAIProfile        string
	ExploreAINoCache        bool
	ExploreAIProvider       string
	ExploreAIEndpoint       string
	ExploreAIModel           string
	ExploreAIMaxTokens       int
}

// Range represents a min-max integer range (e.g. for status codes or response length).
type Range struct {
	Min int
	Max int
}

// StatusRange is an alias for Range used for HTTP status code ranges.
type StatusRange = Range
