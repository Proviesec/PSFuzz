package modules

// Config holds all module-related settings. Kept in the modules package so the
// module system stays separate from the main config; the main config only
// embeds or references this struct.
type Config struct {
	// Modules is the list of enabled response-analysis module names (e.g. fingerprint,cors,ai,urlextract,links).
	Modules []string
	// AIPrompt is the custom prompt for the AI module; placeholders: {{status}}, {{method}}, {{url}}, {{body}}. Empty = default.
	AIPrompt string
	// AIProvider is the AI backend for the ai module: openai | ollama | gemini. Default openai.
	AIProvider string
	// AIEndpoint overrides the API base URL (e.g. http://localhost:11434 for Ollama).
	AIEndpoint string
	// AIModel overrides the model name (default per provider: gpt-4o-mini, llama3.1, gemini-1.5-flash).
	AIModel string
	// AIMaxTokens is the max tokens for the AI module response (0 = use default 150).
	AIMaxTokens int
	// EnqueueModuleUrls is a comma-separated list of module names whose "urls" output is queued for scanning (e.g. urlextract,links).
	EnqueueModuleUrls string
	// ExtractedURLsFile, if set, is the path where all extracted URLs (from any module with "urls" output) are written, one per line.
	ExtractedURLsFile string
}
