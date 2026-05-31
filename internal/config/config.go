package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the settings loaded from a .stacklitrc.json file.
type Config struct {
	Ignore       []string     `json:"ignore,omitempty"`
	MaxDepth     int          `json:"max_depth,omitempty"`
	MaxModules   int          `json:"max_modules,omitempty"`
	MaxExports   int          `json:"max_exports,omitempty"`
	ParseWorkers int          `json:"parse_workers,omitempty"`
	Output       OutputConfig `json:"output,omitempty"`
}

// OutputConfig controls where output files are written.
type OutputConfig struct {
	JSON    string `json:"json,omitempty"`
	Mermaid string `json:"mermaid,omitempty"`
	HTML    string `json:"html,omitempty"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxDepth:     4,
		MaxModules:   200,
		MaxExports:   10,
		ParseWorkers: 1,
		Output: OutputConfig{
			JSON:    "stacklit.json",
			Mermaid: "DEPENDENCIES.md",
			HTML:    "stacklit.html",
		},
	}
}

// Load reads .stacklitrc.json from root and merges it over the defaults.
// If the file does not exist or cannot be parsed, defaults are returned.
func Load(root string) *Config {
	cfg, _ := load(root, false)
	return cfg
}

// LoadValidated reads .stacklitrc.json from root and validates settings that
// need explicit user-facing errors while preserving Load's defaulting behavior.
func LoadValidated(root string) (*Config, error) {
	return load(root, true)
}

func load(root string, validate bool) (*Config, error) {
	cfg := DefaultConfig()

	path := filepath.Join(root, ".stacklitrc.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}

	// Unmarshal on top of cfg so existing defaults survive missing keys.
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, nil
	}

	if validate && cfg.ParseWorkers < 1 {
		return nil, fmt.Errorf("parse_workers must be at least 1")
	}

	// Re-apply defaults for any zero values introduced by an explicit null/0.
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 4
	}
	if cfg.MaxModules == 0 {
		cfg.MaxModules = 200
	}
	if cfg.MaxExports == 0 {
		cfg.MaxExports = 10
	}
	if cfg.ParseWorkers == 0 {
		cfg.ParseWorkers = 1
	}
	if cfg.Output.JSON == "" {
		cfg.Output.JSON = "stacklit.json"
	}
	if cfg.Output.Mermaid == "" {
		cfg.Output.Mermaid = "DEPENDENCIES.md"
	}
	if cfg.Output.HTML == "" {
		cfg.Output.HTML = "stacklit.html"
	}

	return cfg, nil
}

// ScanIgnore returns ignore patterns plus Stacklit output files so generated artifacts
// never feed back into the next scan. All patterns are normalized to forward slashes
// so they match the walker's normalized paths on every OS.
func (c *Config) ScanIgnore() []string {
	ignore := make([]string, 0, len(c.Ignore)+3)
	for _, pat := range c.Ignore {
		ignore = append(ignore, toSlash(pat))
	}
	for _, out := range []string{c.Output.JSON, c.Output.Mermaid, c.Output.HTML} {
		if out == "" {
			continue
		}
		ignore = append(ignore, toSlash(out))
	}
	return ignore
}

// toSlash normalizes both OS path separators and literal backslashes to forward
// slashes. filepath.ToSlash only converts os.PathSeparator which is '/' on
// Unix, leaving Windows-style backslashes untouched on non-Windows builds.
func toSlash(p string) string {
	p = filepath.ToSlash(p)
	return strings.ReplaceAll(p, "\\", "/")
}
