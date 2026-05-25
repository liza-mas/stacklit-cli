package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glincker/stacklit/internal/schema"
	"github.com/spf13/cobra"
)

func TestIndexInputFlags(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *cobra.Command
		wantDefault string
	}{
		{name: "view", cmd: viewCmd, wantDefault: defaultIndexPath},
		{name: "diff", cmd: newDiffCmd(), wantDefault: ""},
		{name: "derive", cmd: newDeriveCmd(), wantDefault: defaultIndexPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup("input")
			if flag == nil {
				t.Fatal("expected input flag to be registered")
			}
			if flag.Shorthand != "i" {
				t.Fatalf("expected -i shorthand, got %q", flag.Shorthand)
			}
			if flag.DefValue != tt.wantDefault {
				t.Fatalf("expected default %q, got %q", tt.wantDefault, flag.DefValue)
			}
		})
	}
}

func TestRemovedCommandsAreNotRegistered(t *testing.T) {
	removed := map[string]bool{
		"completion": true,
		"export":     true,
		"generate":   true,
		"init":       true,
		"serve":      true,
		"setup":      true,
	}

	for _, cmd := range rootCmd.Commands() {
		if removed[cmd.Name()] {
			t.Fatalf("command %q should not be registered", cmd.Name())
		}
	}
}

func TestDeriveDoesNotExposeInjectFlag(t *testing.T) {
	cmd := newDeriveCmd()
	if flag := cmd.Flags().Lookup("inject"); flag != nil {
		t.Fatal("derive should not expose --inject")
	}
}

func TestGenerateJSONExposesIndexingFlags(t *testing.T) {
	cmd := newGenerateJSONCmd()
	for _, name := range []string{"workspace", "multi", "insights"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("generate-json should expose --%s", name)
		}
	}
	if flag := cmd.Flags().Lookup("summary"); flag != nil {
		t.Fatal("generate-json should not expose --summary")
	}
}

func TestInsightsCommandsExposeDefaultPaths(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "init-insights", cmd: newInitInsightsCmd()},
		{name: "ai-summary", cmd: newAISummaryCmd()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.cmd.Flags().Lookup("input")
			if input == nil || input.DefValue != defaultIndexPath {
				t.Fatalf("expected default input %q, got %#v", defaultIndexPath, input)
			}
			output := tt.cmd.Flags().Lookup("output")
			if output == nil || output.DefValue != "stacklit-insights.json" {
				t.Fatalf("expected default output stacklit-insights.json, got %#v", output)
			}
		})
	}
}

func TestGenerateJSONUsesDefaultInsightsWhenPresent(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	if err := os.MkdirAll(filepath.Join(root, "internal", "cli"), 0755); err != nil {
		t.Fatalf("creating fixture module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "cli", "root.go"), []byte("package cli\n\nfunc Execute() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "stacklit-insights.json"), []byte(`{
  "purpose": {
    "cli": "Curated command layer"
  }
}`), 0644); err != nil {
		t.Fatalf("writing insights: %v", err)
	}

	cmd := newGenerateJSONCmd()
	cmd.SetArgs([]string{"-o", "stacklit.json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("generate-json returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "stacklit.json"))
	if err != nil {
		t.Fatalf("reading generated index: %v", err)
	}
	var idx schema.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatalf("parsing generated index: %v", err)
	}
	foundPurpose := false
	for _, mod := range idx.Modules {
		if mod.Purpose == "Curated command layer" {
			foundPurpose = true
			break
		}
	}
	if !foundPurpose {
		t.Fatalf("expected default insights to enrich generated index, got %+v", idx.Modules)
	}
}

func TestGenerateJSONProceedsWithoutDefaultInsights(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	cmd := newGenerateJSONCmd()
	cmd.SetArgs([]string{"-o", "stacklit.json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("generate-json returned error without default insights: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stacklit.json")); err != nil {
		t.Fatalf("expected generated index to exist: %v", err)
	}
}

func restoreWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})
}

func TestFindModulesSortedAndLimited(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"zeta/api":    {Purpose: "API handlers"},
			"alpha/api":   {Purpose: "API handlers"},
			"beta/api":    {Purpose: "API handlers"},
			"gamma/api":   {Purpose: "API handlers"},
			"delta/api":   {Purpose: "API handlers"},
			"epsilon/api": {Purpose: "API handlers"},
		},
	}

	got := findModules(idx, "api")
	if len(got) != 5 {
		t.Fatalf("expected 5 results, got %d", len(got))
	}

	want := []string{"alpha/api", "beta/api", "delta/api", "epsilon/api", "gamma/api"}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("result %d: expected %q, got %q", i, name, got[i].Name)
		}
	}
}

func TestFindModulesMatchesPurposeCaseInsensitive(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/auth": {Purpose: "Authentication and authorization"},
			"internal/db":   {Purpose: "Database access"},
		},
	}

	got := findModules(idx, "AUTHORIZATION")
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Name != "internal/auth" {
		t.Fatalf("expected internal/auth, got %q", got[0].Name)
	}
}

func TestDependencyEdgesReturnsEmptyForKnownIsolatedModule(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/isolated": {Purpose: "Isolated module"},
		},
	}

	edges, err := dependencyEdges(idx, "internal/isolated")
	if err != nil {
		t.Fatalf("dependencyEdges returned error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected no edges, got %v", edges)
	}
}

func TestDependencyEdgesErrorsForUnknownModule(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/known": {Purpose: "Known module"},
		},
	}

	_, err := dependencyEdges(idx, "internal/missing")
	if err == nil {
		t.Fatal("expected error for missing module")
	}
	if !strings.Contains(err.Error(), `module "internal/missing" not found`) {
		t.Fatalf("expected missing module error, got %v", err)
	}
}
