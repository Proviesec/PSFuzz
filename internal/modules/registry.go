package modules

import (
	"strings"
	"sync"
)

var (
	registry   = make(map[string]func(*Config) Analyzer)
	registryMu sync.RWMutex
)

// Register adds a response module. Call from init() in your module file so that
// new modules can be added without editing this file. name is the module name
// (e.g. "links"); factory receives the module config and returns the Analyzer.
func Register(name string, factory func(*Config) Analyzer) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Enabled returns the list of analyzers for the given module config. Only
// names in mc.Modules that have been registered are included (unknown names
// are skipped). Duplicate names appear only once (first occurrence wins).
func Enabled(mc *Config) []Analyzer {
	if mc == nil || len(mc.Modules) == 0 {
		return nil
	}
	registryMu.RLock()
	defer registryMu.RUnlock()
	var out []Analyzer
	seen := make(map[string]bool)
	for _, name := range mc.Modules {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" || seen[name] {
			continue
		}
		factory, ok := registry[name]
		if !ok {
			continue
		}
		seen[name] = true
		out = append(out, factory(mc))
	}
	return out
}
