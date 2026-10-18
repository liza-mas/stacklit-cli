package derive

import (
	"errors"
	"strings"
	"testing"

	"github.com/glincker/stacklit/internal/schema"
)

func TestCompactMapWithOptionsIncludesAISummary(t *testing.T) {
	idx := &schema.Index{
		Project: schema.Project{Name: "demo"},
		Tech:    schema.Tech{PrimaryLanguage: "go"},
		Modules: map[string]schema.ModuleInfo{
			"internal/cli": {Purpose: "CLI commands"},
		},
		Architecture: schema.Architecture{
			Summary: "A focused CLI around an indexing pipeline.",
		},
	}

	got, err := CompactMapWithOptions(idx, CompactMapOptions{IncludeAISummary: true})
	if err != nil {
		t.Fatalf("CompactMapWithOptions returned error: %v", err)
	}
	if !strings.Contains(got, "ai-summary:\nA focused CLI around an indexing pipeline.") {
		t.Fatalf("expected AI summary in output, got:\n%s", got)
	}
	if !strings.Contains(got, "\nmodules:\n") {
		t.Fatalf("expected module section to remain present, got:\n%s", got)
	}
}

func TestCompactMapWithOptionsRequiresAISummaryWhenRequested(t *testing.T) {
	idx := &schema.Index{
		Project: schema.Project{Name: "demo"},
		Tech:    schema.Tech{PrimaryLanguage: "go"},
		Modules: map[string]schema.ModuleInfo{},
	}

	_, err := CompactMapWithOptions(idx, CompactMapOptions{IncludeAISummary: true})
	if !errors.Is(err, ErrMissingAISummary) {
		t.Fatalf("expected ErrMissingAISummary, got %v", err)
	}
}

func TestCompactMapKeepsLongModuleNames(t *testing.T) {
	long := "submodules/execution-engine/internal/stacklitindex"
	idx := &schema.Index{
		Project: schema.Project{Name: "demo"},
		Tech:    schema.Tech{PrimaryLanguage: "go"},
		Modules: map[string]schema.ModuleInfo{
			long:  {Purpose: "Long"},
			"cli": {Purpose: "Short"},
		},
	}

	got := CompactMap(idx)
	if !strings.Contains(got, "\n  "+long+"  long\n") {
		t.Fatalf("expected full long module name, got:\n%s", got)
	}
	if want := "\n  cli" + strings.Repeat(" ", 30-len("cli")) + "  short\n"; !strings.Contains(got, want) {
		t.Fatalf("expected short module name padded to 30 chars, got:\n%s", got)
	}
}

func TestStripBlockRemovesInjectedBlockOnly(t *testing.T) {
	idx := &schema.Index{
		Project: schema.Project{Name: "demo"},
		Hints:   schema.Hints{EnvVars: []string{"STALE_VAR"}},
	}
	content := "# Notes\n\n" + InjectableBlock(idx) + "Uses KEPT_VAR.\n"

	got := StripBlock(content)
	if want := "# Notes\n\nUses KEPT_VAR.\n"; got != want {
		t.Fatalf("StripBlock = %q, want %q", got, want)
	}
	if plain := "no block here\n"; StripBlock(plain) != plain {
		t.Fatalf("StripBlock changed content without a block")
	}
}
