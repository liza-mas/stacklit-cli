package hint

import (
	"testing"

	"github.com/glincker/stacklit/internal/schema"
)

func TestAddFeatureSelectsHandlerModuleDeterministically(t *testing.T) {
	modules := map[string]schema.ModuleInfo{
		"internal/handlers": {},
		"internal/api":      {},
		"internal/engine":   {},
	}

	got := AddFeature(modules, []string{"cmd/stacklit/main.go"})

	if got != "Add handler in internal/api, register in cmd/stacklit/main.go" {
		t.Fatalf("expected lexicographically first handler module, got %q", got)
	}
}
