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
