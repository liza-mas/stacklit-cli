package engine

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glincker/stacklit/internal/config"
	"github.com/glincker/stacklit/internal/git"
	"github.com/glincker/stacklit/internal/graph"
	"github.com/glincker/stacklit/internal/monorepo"
	"github.com/glincker/stacklit/internal/parser"
)

func TestRunJSONOnlyWritesOnlyJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	result, err := Run(Options{
		Root:       root,
		Quiet:      true,
		JSONOnly:   true,
		JSONOutput: "custom-stacklit.json",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.JSONPath != filepath.Join(root, "custom-stacklit.json") {
		t.Fatalf("expected custom JSON path, got %q", result.JSONPath)
	}
	if _, err := os.Stat(result.JSONPath); err != nil {
		t.Fatalf("expected JSON output to exist: %v", err)
	}

	for _, path := range []string{"DEPENDENCIES.md", "stacklit.html"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatalf("expected %s not to be written, stat error: %v", path, err)
		}
	}
}

func TestRunJSONOnlyUsesConfiguredJSONOutputByDefault(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".stacklitrc.json"), []byte(`{"output":{"json":"custom-index.json"}}`), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	result, err := Run(Options{
		Root:     root,
		Quiet:    true,
		JSONOnly: true,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.JSONPath != filepath.Join(root, "custom-index.json") {
		t.Fatalf("expected configured JSON path, got %q", result.JSONPath)
	}
	if _, err := os.Stat(result.JSONPath); err != nil {
		t.Fatalf("expected configured JSON output to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stacklit.json")); !os.IsNotExist(err) {
		t.Fatalf("expected default stacklit.json not to be written, stat error: %v", err)
	}
}

func TestRunRecordsWorkspaceRelativeProjectRoot(t *testing.T) {
	workspace := t.TempDir()
	root := filepath.Join(workspace, "services", "api")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("creating fixture root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	result, err := Run(Options{
		Root:      root,
		Workspace: workspace,
		Quiet:     true,
		JSONOnly:  true,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Index.Project.Root != "services/api" {
		t.Fatalf("expected workspace-relative project root, got %q", result.Index.Project.Root)
	}
}

func TestRunAppliesInsights(t *testing.T) {
	root := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("changing working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})
	moduleDir := filepath.Join(root, "internal", "engine")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("creating module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "engine.go"), []byte("package engine\n\nfunc Run() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "stacklit-insights.json"), []byte(`{
  "purpose": {
    "engine": "Curated indexing pipeline"
  },
  "hints": {
    "test_command": "make test"
  },
  "architecture": {
    "ai_summary": "Curated architecture summary"
  }
}`), 0644); err != nil {
		t.Fatalf("writing insights: %v", err)
	}

	result, err := Run(Options{
		Root:         root,
		Quiet:        true,
		JSONOnly:     true,
		InsightsPath: "stacklit-insights.json",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	foundPurpose := false
	for _, mod := range result.Index.Modules {
		if mod.Purpose == "Curated indexing pipeline" {
			foundPurpose = true
			break
		}
	}
	if !foundPurpose {
		t.Fatalf("expected curated purpose in modules, got %+v", result.Index.Modules)
	}
	if result.Index.Hints.TestCmd != "make test" {
		t.Fatalf("expected curated test command, got %q", result.Index.Hints.TestCmd)
	}
	if result.Index.Architecture.Summary != "Curated architecture summary" {
		t.Fatalf("expected curated summary, got %q", result.Index.Architecture.Summary)
	}
}

func TestRunWarnsButContinuesWhenInsightsMissing(t *testing.T) {
	root := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("changing working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	stderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stderr pipe: %v", err)
	}
	os.Stderr = w

	result, runErr := Run(Options{
		Root:                root,
		Quiet:               true,
		JSONOnly:            true,
		InsightsPath:        "missing-insights.json",
		WarnMissingInsights: true,
	})

	if err := w.Close(); err != nil {
		t.Fatalf("closing stderr pipe: %v", err)
	}
	os.Stderr = stderr
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading stderr: %v", err)
	}

	if runErr != nil {
		t.Fatalf("Run returned error: %v", runErr)
	}
	if result == nil || result.Index == nil {
		t.Fatal("expected index result")
	}
	if !strings.Contains(string(data), "warning: insights file") {
		t.Fatalf("expected missing insights warning, got %q", string(data))
	}
}

func TestRunMultiWritesJSONWithTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	repo := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("creating repo fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("writing repo fixture: %v", err)
	}
	reposFile := filepath.Join(tmpDir, "repos.txt")
	if err := os.WriteFile(reposFile, []byte(repo+"\n"), 0644); err != nil {
		t.Fatalf("writing repos file: %v", err)
	}
	outputPath := filepath.Join(tmpDir, "stacklit-multi.json")

	if _, err := RunMulti(MultiOptions{
		ReposFile:  reposFile,
		OutputPath: outputPath,
		Quiet:      true,
		JSONOnly:   true,
	}); err != nil {
		t.Fatalf("RunMulti returned error: %v", err)
	}

	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading multi output: %v", err)
	}
	if raw[len(raw)-1] != '\n' {
		t.Fatalf("multi output should end with newline, got final byte %q", raw[len(raw)-1])
	}
}

func TestAssembleIndexFiltersTrimmedModuleReferences(t *testing.T) {
	files := []*parser.FileInfo{
		{Path: "src/api/index.ts", Language: "TypeScript", Imports: []string{"src/auth", "src/db"}, LineCount: 50},
		{Path: "src/auth/service.ts", Language: "TypeScript", Imports: []string{"src/db"}, LineCount: 40},
		{Path: "src/db/pool.ts", Language: "TypeScript", Exports: []string{"Pool"}, LineCount: 30},
	}

	g := graph.Build(files, graph.BuildOptions{MaxDepth: 4})
	cfg := config.DefaultConfig()
	cfg.MaxModules = 2

	idx := assembleIndex(
		".",
		"",
		&monorepo.Result{Type: "single"},
		[]string{"src/api/index.ts", "src/auth/service.ts", "src/db/pool.ts"},
		files,
		g,
		&git.Activity{},
		map[string][]byte{},
		cfg,
	)

	if len(idx.Modules) != 2 {
		t.Fatalf("expected 2 retained modules, got %d: %v", len(idx.Modules), idx.Modules)
	}
	if _, ok := idx.Modules["src/db"]; ok {
		t.Fatalf("expected trimmed module src/db to be omitted from modules, got %v", idx.Modules)
	}

	api := idx.Modules["src/api"]
	if len(api.DependsOn) != 1 || api.DependsOn[0] != "src/auth" {
		t.Fatalf("expected src/api depends_on to retain only src/auth, got %v", api.DependsOn)
	}

	auth := idx.Modules["src/auth"]
	if len(auth.DependsOn) != 0 {
		t.Fatalf("expected src/auth depends_on to drop trimmed src/db reference, got %v", auth.DependsOn)
	}
	if len(auth.DependedBy) != 1 || auth.DependedBy[0] != "src/api" {
		t.Fatalf("expected src/auth depended_by to contain only src/api, got %v", auth.DependedBy)
	}

	if len(idx.Dependencies.Edges) != 1 || idx.Dependencies.Edges[0] != ([2]string{"src/api", "src/auth"}) {
		t.Fatalf("expected only retained edge src/api -> src/auth, got %v", idx.Dependencies.Edges)
	}

	if len(idx.Dependencies.MostDepended) < 2 {
		t.Fatalf("expected ranked retained modules, got %v", idx.Dependencies.MostDepended)
	}
	if idx.Dependencies.MostDepended[0] != "src/auth" {
		t.Fatalf("expected src/auth to be most depended after trimming, got %v", idx.Dependencies.MostDepended)
	}
	for _, name := range idx.Dependencies.MostDepended {
		if name == "src/db" {
			t.Fatalf("expected trimmed module src/db to be absent from most_depended, got %v", idx.Dependencies.MostDepended)
		}
	}

	if len(idx.Dependencies.Isolated) != 0 {
		t.Fatalf("expected no isolated retained modules, got %v", idx.Dependencies.Isolated)
	}
}
