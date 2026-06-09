package insights

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glincker/stacklit/internal/schema"
)

func TestApplyOverridesPurposeHintsAndArchitecture(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/engine": {Purpose: "Core orchestration engine"},
			"internal/cli":    {Purpose: "Command-line interface"},
		},
		Hints: schema.Hints{
			TestCmd:    "go test ./...",
			EnvVars:    []string{"DATABASE_URL"},
			DoNotTouch: []string{"vendor/"},
		},
	}
	file := &File{
		Purpose: map[string]string{
			"internal/engine": "Index generation pipeline",
			"cli":             "CLI commands and wiring",
		},
		Hints: schema.Hints{
			TestCmd:    "make test",
			EnvVars:    []string{"ANTHROPIC_API_KEY"},
			DoNotTouch: []string{".github/"},
		},
		Architecture: schema.Architecture{
			Pattern: "Cobra CLI",
			Summary: "Small CLI built around an indexing engine.",
		},
	}

	Apply(idx, file)

	if got := idx.Modules["internal/engine"].Purpose; got != "Index generation pipeline" {
		t.Fatalf("expected exact purpose override, got %q", got)
	}
	if got := idx.Modules["internal/cli"].Purpose; got != "CLI commands and wiring" {
		t.Fatalf("expected basename purpose override, got %q", got)
	}
	if idx.Hints.TestCmd != "make test" {
		t.Fatalf("expected test command override, got %q", idx.Hints.TestCmd)
	}
	if len(idx.Hints.EnvVars) != 2 {
		t.Fatalf("expected env vars to be unioned, got %v", idx.Hints.EnvVars)
	}
	if idx.Architecture.Summary == "" || idx.Architecture.Pattern == "" {
		t.Fatalf("expected architecture to be applied, got %+v", idx.Architecture)
	}
}

func TestSeedFromIndexPreservesExistingValuesAndPrunes(t *testing.T) {
	file := &File{
		Purpose: map[string]string{
			"internal/cli":     "Curated CLI purpose",
			"internal/removed": "Removed module",
		},
		Hints: schema.Hints{TestCmd: "make test"},
	}
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/cli":    {Purpose: "Command-line interface"},
			"internal/engine": {Purpose: "Core orchestration engine"},
		},
		Hints: schema.Hints{
			TestCmd:    "go test ./...",
			AddFeature: "Add commands in internal/cli",
		},
		Architecture: schema.Architecture{Summary: "Existing index summary"},
	}

	SeedFromIndex(file, idx, true)

	if got := file.Purpose["internal/cli"]; got != "Curated CLI purpose" {
		t.Fatalf("expected existing purpose to be preserved, got %q", got)
	}
	if got := file.Purpose["internal/engine"]; got != "Core orchestration engine" {
		t.Fatalf("expected new purpose to be seeded, got %q", got)
	}
	if _, ok := file.Purpose["internal/removed"]; ok {
		t.Fatal("expected removed module to be pruned")
	}
	if got := file.Hints.TestCmd; got != "make test" {
		t.Fatalf("expected existing hint to be preserved, got %q", got)
	}
	if got := file.Hints.AddFeature; got != "Add commands in internal/cli" {
		t.Fatalf("expected missing hint to be seeded, got %q", got)
	}
	if got := file.Architecture.Summary; got != "Existing index summary" {
		t.Fatalf("expected missing architecture summary to be seeded, got %q", got)
	}
}

func TestMergeAppliesGeneratedInsights(t *testing.T) {
	target := &File{
		Purpose: map[string]string{
			"internal/cli":    "Curated CLI purpose",
			"internal/engine": "Core orchestration engine",
		},
		Hints: schema.Hints{
			AddFeature: "Add handler in internal/api, register in cmd/stacklit/main.go",
			TestCmd:    "go test ./...",
			EnvVars:    []string{"ANTHROPIC_API_KEY"},
		},
		Architecture: schema.Architecture{
			Pattern: "Curated architecture pattern",
			Summary: "Old summary",
		},
	}
	source := &File{
		Purpose: map[string]string{
			"internal/cli":     "CLI commands and wiring",
			"internal/engine":  "Index generation pipeline",
			"internal/summary": "AI insight generation",
			"internal/empty":   "",
		},
		Hints: schema.Hints{
			AddFeature: "Add commands in internal/cli",
			TestCmd:    "make test",
			EnvVars:    []string{"STACKLIT_SUMMARY_CMD", "ANTHROPIC_API_KEY"},
		},
		Architecture: schema.Architecture{
			Pattern: "Generated architecture pattern",
			Summary: "Generated architecture summary",
		},
	}
	idx := &schema.Index{
		Structure: schema.Structure{Entrypoints: []string{"cmd/stacklit/main.go"}},
		Modules: map[string]schema.ModuleInfo{
			"internal/api":      {Purpose: "API endpoints and handlers"},
			"internal/handlers": {Purpose: "Handlers"},
		},
	}

	Merge(target, source, idx)

	if got := target.Purpose["internal/cli"]; got != "Curated CLI purpose" {
		t.Fatalf("expected curated purpose to be preserved, got %q", got)
	}
	if got := target.Purpose["internal/engine"]; got != "Index generation pipeline" {
		t.Fatalf("expected mechanical purpose to be replaced, got %q", got)
	}
	if got := target.Purpose["internal/summary"]; got != "AI insight generation" {
		t.Fatalf("expected missing purpose to be filled from generated value, got %q", got)
	}
	if _, ok := target.Purpose["internal/empty"]; ok {
		t.Fatal("expected empty generated purpose to be ignored")
	}
	if got := target.Hints.TestCmd; got != "go test ./..." {
		t.Fatalf("expected existing test command to be preserved, got %q", got)
	}
	if got := target.Hints.AddFeature; got != "Add commands in internal/cli" {
		t.Fatalf("expected mechanical add_feature hint to be replaced, got %q", got)
	}
	if len(target.Hints.EnvVars) != 2 {
		t.Fatalf("expected env vars to be unioned, got %v", target.Hints.EnvVars)
	}
	if got := target.Architecture.Summary; got != "Generated architecture summary" {
		t.Fatalf("expected generated architecture summary, got %q", got)
	}
	if got := target.Architecture.Pattern; got != "Curated architecture pattern" {
		t.Fatalf("expected curated architecture pattern to be preserved, got %q", got)
	}
}

func TestWriteEndsWithNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stacklit-insights.json")
	if err := Write(path, &File{
		Purpose: map[string]string{"internal/engine": "Index generation pipeline"},
	}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading insights file: %v", err)
	}
	if raw[len(raw)-1] != '\n' {
		t.Fatalf("insights output should end with newline, got final byte %q", raw[len(raw)-1])
	}
}
