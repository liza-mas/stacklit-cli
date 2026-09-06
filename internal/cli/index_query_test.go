package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glincker/stacklit/internal/archexport"
	"github.com/glincker/stacklit/internal/insights"
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
		{name: "export-architecture", cmd: newExportArchitectureCmd(), wantDefault: defaultIndexPath},
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

func TestExportArchitectureCommandIsRegistered(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "export-architecture" {
			return
		}
	}
	t.Fatal("expected export-architecture command to be registered")
}

func TestExportArchitectureWritesStdout(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "stacklit.json")
	writeArchitectureIndexFixture(t, indexPath)

	var stdout bytes.Buffer
	err := runExportArchitecture(&exportArchitectureOptions{input: indexPath}, &stdout, time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("export-architecture returned error: %v", err)
	}

	var export archexport.Export
	if err := json.Unmarshal(stdout.Bytes(), &export); err != nil {
		t.Fatalf("parsing architecture export: %v", err)
	}
	if export.SchemaVersion != archexport.SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", archexport.SchemaVersion, export.SchemaVersion)
	}
	if export.GeneratedAt != "2026-06-07T12:00:00Z" {
		t.Fatalf("unexpected generated_at: %s", export.GeneratedAt)
	}
	if len(export.Membership) != 1 || export.Membership[0].Path != "main.go" {
		t.Fatalf("expected root membership to use repo-relative path, got %+v", export.Membership)
	}
}

func TestExportArchitectureWritesOutputFile(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "stacklit.json")
	outputPath := filepath.Join(root, "architecture.json")
	writeArchitectureIndexFixture(t, indexPath)

	err := runExportArchitecture(
		&exportArchitectureOptions{input: indexPath, output: outputPath},
		&bytes.Buffer{},
		time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("export-architecture returned error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !bytes.HasSuffix(data, []byte("\n")) {
		t.Fatal("expected output JSON to end with newline")
	}
}

func TestExportArchitectureInvalidInputDoesNotTouchOutput(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "stacklit.json")
	outputPath := filepath.Join(root, "architecture.json")
	if err := os.WriteFile(indexPath, []byte("{"), 0644); err != nil {
		t.Fatalf("writing invalid index: %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("keep\n"), 0644); err != nil {
		t.Fatalf("writing existing output: %v", err)
	}

	err := runExportArchitecture(
		&exportArchitectureOptions{input: indexPath, output: outputPath},
		&bytes.Buffer{},
		time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("expected invalid index to fail")
	}
	data, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("reading output: %v", readErr)
	}
	if string(data) != "keep\n" {
		t.Fatalf("output file should be unchanged, got %q", string(data))
	}
}

func TestDeriveDoesNotExposeInjectFlag(t *testing.T) {
	cmd := newDeriveCmd()
	if flag := cmd.Flags().Lookup("inject"); flag != nil {
		t.Fatal("derive should not expose --inject")
	}
}

func writeArchitectureIndexFixture(t *testing.T, path string) {
	t.Helper()
	data := []byte(`{
  "project": {"name": "demo"},
  "tech": {
    "primary_language": "go",
    "framework_patterns": [{"name": "go", "entry": "server.go"}]
  },
  "structure": {"entrypoints": ["main.go"]},
  "modules": {
    "root": {
      "purpose": "Root files",
      "file_list": ["main.go"]
    }
  },
  "dependencies": {
    "edges": [],
    "entrypoints": ["main.go"]
  }
}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writing architecture index fixture: %v", err)
	}
}

func TestDeriveExposesAISummaryFlag(t *testing.T) {
	cmd := newDeriveCmd()
	if flag := cmd.Flags().Lookup("ai-summary"); flag == nil {
		t.Fatal("derive should expose --ai-summary")
	}
}

func TestDeriveAISummaryRequiresSummary(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "stacklit.json")
	if err := os.WriteFile(indexPath, []byte(`{
  "project": {"name": "demo"},
  "tech": {"primary_language": "go"},
  "modules": {}
}`), 0644); err != nil {
		t.Fatalf("writing index fixture: %v", err)
	}

	cmd := newDeriveCmd()
	cmd.SetArgs([]string{"-i", indexPath, "--ai-summary"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing AI summary error")
	}
	if !strings.Contains(err.Error(), "has no architecture.ai_summary") {
		t.Fatalf("expected missing AI summary message, got %v", err)
	}
}

func TestGenerateJSONExposesIndexingFlags(t *testing.T) {
	cmd := newGenerateJSONCmd()
	for _, name := range []string{"workspace", "multi", "insights", "parse-workers"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("generate-json should expose --%s", name)
		}
	}
	if flag := cmd.Flags().Lookup("parse-workers"); flag.Value.Type() != "int" {
		t.Fatalf("expected --parse-workers to be an int flag, got %q", flag.Value.Type())
	}
	if flag := cmd.Flags().Lookup("summary"); flag != nil {
		t.Fatal("generate-json should not expose --summary")
	}
}

func TestGenerateJSONRejectsInvalidParseWorkersFlag(t *testing.T) {
	for _, value := range []string{"0", "-2"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			restoreWorkingDirectory(t, root)
			writeGenerateJSONFixture(t, root)

			cmd := newGenerateJSONCmd()
			cmd.SetArgs([]string{"--parse-workers", value, "-o", "stacklit.json"})
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected invalid --parse-workers value to fail")
			}
			if !strings.Contains(err.Error(), "--parse-workers") {
				t.Fatalf("expected error to name --parse-workers, got %v", err)
			}
		})
	}
}

func TestGenerateJSONParseWorkersFlagPrecedesConfig(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	writeGenerateJSONFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, ".stacklitrc.json"), []byte(`{"parse_workers":0}`), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cmd := newGenerateJSONCmd()
	cmd.SetArgs([]string{"--parse-workers", "2", "-o", "stacklit.json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("generate-json with CLI parse worker override returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stacklit.json")); err != nil {
		t.Fatalf("expected generated index to exist: %v", err)
	}
}

func TestGenerateJSONMultiForwardsParseWorkersFlag(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)

	repoOne := filepath.Join(root, "repo-one")
	repoTwo := filepath.Join(root, "repo-two")
	writeGenerateJSONFixture(t, repoOne)
	writeGenerateJSONFixture(t, repoTwo)
	if err := os.WriteFile(filepath.Join(repoTwo, ".stacklitrc.json"), []byte(`{"parse_workers":0}`), 0644); err != nil {
		t.Fatalf("writing repo config: %v", err)
	}

	reposFile := filepath.Join(root, "repos.txt")
	if err := os.WriteFile(reposFile, []byte(repoOne+"\n"+repoTwo+"\n"), 0644); err != nil {
		t.Fatalf("writing repos file: %v", err)
	}

	cmd := newGenerateJSONCmd()
	cmd.SetArgs([]string{"--multi", reposFile, "--parse-workers", "2", "-o", "stacklit-multi.json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("generate-json --multi with parse worker override returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stacklit-multi.json")); err != nil {
		t.Fatalf("expected generated multi index to exist: %v", err)
	}
}

func TestGenerateJSONAbsentParseWorkersFlagUsesConfigValidation(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	writeGenerateJSONFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, ".stacklitrc.json"), []byte(`{"parse_workers":0}`), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cmd := newGenerateJSONCmd()
	cmd.SetArgs([]string{"-o", "stacklit.json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected invalid configured parse_workers to fail without CLI override")
	}
	if !strings.Contains(err.Error(), "parse_workers") {
		t.Fatalf("expected error to name parse_workers, got %v", err)
	}
	if strings.Contains(err.Error(), "--parse-workers") {
		t.Fatalf("expected config error without CLI flag, got %v", err)
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

func TestAISummaryWritesGeneratedInsights(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "stacklit.json")
	outputPath := filepath.Join(root, "custom insights.json")
	summaryPath := filepath.Join(root, "summary")

	idx := schema.Index{
		Project: schema.Project{Name: "stacklit", Root: ".", Type: "go"},
		Tech: schema.Tech{
			PrimaryLanguage: "go",
			Languages: map[string]schema.LangStats{
				"go": {Files: 1, Lines: 12},
			},
		},
		Structure: schema.Structure{Entrypoints: []string{"cmd/stacklit/main.go"}},
		Modules: map[string]schema.ModuleInfo{
			"internal/cli":     {Purpose: "Old CLI purpose", Files: 1, Lines: 12},
			"internal/summary": {Purpose: "Old summary purpose", Files: 1, Lines: 6},
		},
		Dependencies: schema.Dependencies{},
		Hints:        schema.Hints{TestCmd: "go test ./..."},
	}
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshalling index fixture: %v", err)
	}
	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		t.Fatalf("writing index fixture: %v", err)
	}
	if err := os.WriteFile(outputPath, []byte(`{
  "purpose": {
    "internal/cli": "Existing curated purpose",
    "internal/removed": "Removed module"
  }
}`), 0644); err != nil {
		t.Fatalf("writing existing insights: %v", err)
	}
	if err := os.WriteFile(summaryPath, []byte(`#!/bin/sh
case "$*" in *"$SUMMARY_EXPECTED_INSIGHTS"*) ;; *) exit 1 ;; esac
cat <<'JSON'
{
  "purpose": {
    "internal/cli": "Generated CLI purpose",
    "internal/summary": "AI-generated insight production",
    "internal/generated_removed": "Generated removed module"
  },
  "hints": {
    "add_feature": "Add commands in internal/cli",
    "test_command": "go test ./...",
    "env_vars": ["STACKLIT_SUMMARY_CMD"]
  },
  "architecture": {
    "ai_summary": "A focused CLI around an indexing pipeline."
  }
}
JSON
`), 0755); err != nil {
		t.Fatalf("writing summary command: %v", err)
	}

	t.Setenv("STACKLIT_SUMMARY_CMD", summaryPath)
	t.Setenv("SUMMARY_EXPECTED_INSIGHTS", outputPath)
	cmd := newAISummaryCmd()
	cmd.SetArgs([]string{"-i", indexPath, "-o", outputPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ai-summary returned error: %v", err)
	}

	got, err := insights.Load(outputPath)
	if err != nil {
		t.Fatalf("loading generated insights: %v", err)
	}
	if got.Purpose["internal/cli"] != "Existing curated purpose" {
		t.Fatalf("expected existing curated purpose to be preserved, got %+v", got.Purpose)
	}
	if got.Purpose["internal/summary"] != "AI-generated insight production" {
		t.Fatalf("expected generated purpose for new module, got %+v", got.Purpose)
	}
	if _, ok := got.Purpose["internal/removed"]; ok {
		t.Fatalf("expected removed existing purpose to be pruned, got %+v", got.Purpose)
	}
	if _, ok := got.Purpose["internal/generated_removed"]; ok {
		t.Fatalf("expected off-index generated purpose to be ignored, got %+v", got.Purpose)
	}
	if got.Hints.AddFeature != "Add commands in internal/cli" {
		t.Fatalf("expected generated add_feature hint, got %+v", got.Hints)
	}
	if got.Hints.TestCmd != "go test ./..." {
		t.Fatalf("expected generated test command, got %+v", got.Hints)
	}
	if got.Architecture.Summary != "A focused CLI around an indexing pipeline." {
		t.Fatalf("expected generated architecture summary, got %+v", got.Architecture)
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

func writeGenerateJSONFixture(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("creating fixture root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
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
