package purpose

import (
	"path/filepath"
	"strings"
)

// purposeMap contains common directory names and their mechanical purpose labels.
var purposeMap = map[string]string{
	"auth":       "Authentication and authorization",
	"api":        "API endpoints and handlers",
	"db":         "Database access layer",
	"models":     "Data models and types",
	"config":     "Configuration management",
	"components": "UI components",
	"hooks":      "React hooks",
	"cmd":        "Application entrypoints",
	"internal":   "Private application packages",
	"pkg":        "Public packages",
	"lib":        "Shared library code",
	"utils":      "Utility functions",
	"services":   "Business logic services",
	"middleware": "HTTP middleware",
	"cli":        "Command-line interface",
	"schema":     "Data schema definitions",
	"renderer":   "Output renderers",
	"walker":     "File system walker",
	"graph":      "Dependency graph",
	"engine":     "Core orchestration engine",
	"git":        "Git integration",
	"assets":     "Static assets",
	"parser":     "Source code parsers",
	"monorepo":   "Monorepo detection",
	"detect":     "Framework and tool detection",
	"summary":    "AI-powered codebase summaries",
}

// Infer returns the mechanical purpose label for a module path.
func Infer(name string) string {
	base := filepath.Base(name)
	if desc, ok := purposeMap[base]; ok {
		return desc
	}
	if base == "." || base == "" || base == "root" {
		return "Root package"
	}
	words := strings.Fields(strings.ReplaceAll(base, "_", " "))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
