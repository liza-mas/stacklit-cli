package hint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/glincker/stacklit/internal/schema"
)

// AddFeature returns the mechanical guidance hint for where to add a feature.
func AddFeature(modules map[string]schema.ModuleInfo, entrypoints []string) string {
	candidates := make([]string, 0, len(modules))
	for name := range modules {
		if isHandlerModule(name) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)
	if len(candidates) > 0 {
		if len(entrypoints) > 0 {
			return fmt.Sprintf("Add handler in %s, register in %s", candidates[0], entrypoints[0])
		}
		return fmt.Sprintf("Add handler in %s", candidates[0])
	}
	if len(entrypoints) > 0 {
		return fmt.Sprintf("Start from entrypoint %s", entrypoints[0])
	}
	return ""
}

func isHandlerModule(name string) bool {
	base := filepath.Base(name)
	return base == "api" || base == "handler" || base == "handlers" ||
		strings.Contains(name, "/api") || strings.Contains(name, "/handler")
}
