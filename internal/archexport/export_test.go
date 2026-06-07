package archexport

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/glincker/stacklit/internal/schema"
)

func TestBuildArchitectureExport(t *testing.T) {
	indexBytes := []byte(`{"modules":{}}`)
	generatedAt := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"root": {
				Purpose:  "Repository root scripts",
				FileList: []string{"install.sh"},
			},
			"internal/cli": {
				Purpose:  "Command layer",
				FileList: []string{"root.go", "derive.go"},
			},
		},
		Dependencies: schema.Dependencies{
			Edges: [][2]string{
				{"internal/cli", "internal/schema"},
				{"cmd/stacklit", "internal/cli"},
			},
			Entrypoints: []string{"cmd/stacklit/main.go"},
		},
		Structure: schema.Structure{
			Entrypoints: []string{"cmd/stacklit/main.go"},
		},
		Tech: schema.Tech{
			FrameworkPatterns: []schema.FrameworkPattern{
				{Name: "go", Entry: "server.go"},
			},
		},
	}

	export := Build("stacklit.json", indexBytes, idx, generatedAt, "test-version")

	if export.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, export.SchemaVersion)
	}
	if export.Generator.Name != "stacklit" || export.Generator.Version != "test-version" {
		t.Fatalf("unexpected generator: %+v", export.Generator)
	}
	if export.GeneratedAt != "2026-06-07T12:00:00Z" {
		t.Fatalf("unexpected generated_at: %s", export.GeneratedAt)
	}
	wantFingerprint := fmt.Sprintf("sha256:%x", sha256.Sum256(indexBytes))
	if export.Inputs.StacklitIndex.Fingerprint != wantFingerprint {
		t.Fatalf("expected fingerprint %q, got %q", wantFingerprint, export.Inputs.StacklitIndex.Fingerprint)
	}

	wantPackages := []Unit{
		{ID: "internal/cli", Name: "internal/cli", Description: "Command layer"},
		{ID: "root", Name: "root", Description: "Repository root scripts"},
	}
	if fmt.Sprint(export.Packages) != fmt.Sprint(wantPackages) {
		t.Fatalf("unexpected packages: %+v", export.Packages)
	}
	if fmt.Sprint(export.Components) != fmt.Sprint(wantPackages) {
		t.Fatalf("components should match packages in v1: %+v", export.Components)
	}

	wantMembership := []Membership{
		{Path: "internal/cli/derive.go", Package: "internal/cli", Component: "internal/cli"},
		{Path: "internal/cli/root.go", Package: "internal/cli", Component: "internal/cli"},
		{Path: "install.sh", Package: "root", Component: "root"},
	}
	if fmt.Sprint(export.Membership) != fmt.Sprint(wantMembership) {
		t.Fatalf("unexpected membership: %+v", export.Membership)
	}

	wantRelationships := []Relationship{
		{Source: "cmd/stacklit", Target: "internal/cli", Type: "package_dependency"},
		{Source: "internal/cli", Target: "internal/schema", Type: "package_dependency"},
	}
	if fmt.Sprint(export.Relationships) != fmt.Sprint(wantRelationships) {
		t.Fatalf("unexpected relationships: %+v", export.Relationships)
	}

	wantEntryPoints := []EntryPoint{
		{Path: "cmd/stacklit/main.go", Kind: "command", Source: "parser", Confidence: parserConfidence},
		{Path: "server.go", Kind: "service", Source: "framework_pattern", Confidence: frameworkConfidence},
	}
	if fmt.Sprint(export.EntryPoints) != fmt.Sprint(wantEntryPoints) {
		t.Fatalf("unexpected entry points: %+v", export.EntryPoints)
	}
}

func TestBuildArchitectureExportUsesEmptyArrays(t *testing.T) {
	export := Build("stacklit.json", nil, &schema.Index{}, time.Time{}, "dev")

	if export.Packages == nil {
		t.Fatal("packages should be an empty array, not null")
	}
	if export.Components == nil {
		t.Fatal("components should be an empty array, not null")
	}
	if export.Membership == nil {
		t.Fatal("membership should be an empty array, not null")
	}
	if export.Relationships == nil {
		t.Fatal("relationships should be an empty array, not null")
	}
	if export.EntryPoints == nil {
		t.Fatal("entry_points should be an empty array, not null")
	}
}
