package insights

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/glincker/stacklit/internal/derive"
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
			"engine":           "Curated engine basename purpose",
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
	if got := file.Purpose["engine"]; got != "Curated engine basename purpose" {
		t.Fatalf("expected basename purpose to be preserved, got %q", got)
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
			"internal/cli":     "Curated CLI purpose",
			"internal/engine":  "Core orchestration engine",
			"handlers":         "Curated handler basename purpose",
			"internal/removed": "Removed module",
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
			"internal/removed": "Generated removed module",
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
			"internal/cli":      {Purpose: "Command-line interface"},
			"internal/engine":   {Purpose: "Core orchestration engine"},
			"internal/summary":  {Purpose: "AI-powered codebase summaries"},
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
	if got := target.Purpose["handlers"]; got != "Curated handler basename purpose" {
		t.Fatalf("expected basename purpose to be preserved, got %q", got)
	}
	if _, ok := target.Purpose["internal/removed"]; ok {
		t.Fatal("expected removed purpose to be pruned")
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

func writeTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func staleHintsFile() *File {
	return &File{Hints: schema.Hints{
		EnvVars:    []string{"CODE_VAR", "DOC_VAR", "INSIGHTS_ONLY_VAR", "UNTRACKED_VAR", "BLOCK_ONLY_VAR"},
		DoNotTouch: []string{"vendor/", "migrations/", "gen/*.pb.go"},
	}}
}

func TestPruneHintsDropsUnreferencedEnvVarsAndMissingPaths(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH")
	}
	dir := t.TempDir()
	block := derive.InjectableBlock(&schema.Index{Hints: schema.Hints{EnvVars: []string{"BLOCK_ONLY_VAR"}}})
	writeTestFiles(t, dir, map[string]string{
		"main.go":                `os.Getenv("CODE_VAR")`,
		"README.md":              "Set DOC_VAR before running.",
		"CLAUDE.md":              "# Agent notes\n\n" + block,
		"stacklit-insights.json": `{"hints":{"env_vars":["INSIGHTS_ONLY_VAR"]}}`,
		"vendor/lib.go":          "package lib",
	})
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	writeTestFiles(t, dir, map[string]string{"scratch.go": `os.Getenv("UNTRACKED_VAR")`})

	file := staleHintsFile()
	PruneHints(file, dir, []string{"stacklit-insights.json"})

	if want := []string{"CODE_VAR", "DOC_VAR"}; !reflect.DeepEqual(file.Hints.EnvVars, want) {
		t.Errorf("EnvVars = %v, want %v", file.Hints.EnvVars, want)
	}
	if want := []string{"vendor/", "gen/*.pb.go"}; !reflect.DeepEqual(file.Hints.DoNotTouch, want) {
		t.Errorf("DoNotTouch = %v, want %v", file.Hints.DoNotTouch, want)
	}
}

func TestPruneHintsKeepsEnvVarsReferencedInSubmodules(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH")
	}
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{
			"-c", "user.name=test", "-c", "user.email=test@example.com",
			"-c", "commit.gpgsign=false", "-c", "protocol.file.allow=always",
		}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	child := t.TempDir()
	writeTestFiles(t, child, map[string]string{"lib.go": `os.Getenv("CHILD_VAR")`})
	run(child, "init", "-q")
	run(child, "add", ".")
	run(child, "commit", "-q", "-m", "child")

	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{"main.go": "package main"})
	run(dir, "init", "-q")
	run(dir, "submodule", "add", "-q", child, "components/lib")

	file := &File{Hints: schema.Hints{EnvVars: []string{"CHILD_VAR", "GONE_VAR"}}}
	PruneHints(file, dir, nil)

	if want := []string{"CHILD_VAR"}; !reflect.DeepEqual(file.Hints.EnvVars, want) {
		t.Errorf("EnvVars = %v, want %v", file.Hints.EnvVars, want)
	}
}

func TestPruneHintsOutsideGitKeepsEnvVars(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{"vendor/lib.go": "package lib"})

	file := staleHintsFile()
	want := append([]string(nil), file.Hints.EnvVars...)
	PruneHints(file, dir, nil)

	if !reflect.DeepEqual(file.Hints.EnvVars, want) {
		t.Errorf("EnvVars = %v, want unchanged %v", file.Hints.EnvVars, want)
	}
	if want := []string{"vendor/", "gen/*.pb.go"}; !reflect.DeepEqual(file.Hints.DoNotTouch, want) {
		t.Errorf("DoNotTouch = %v, want %v", file.Hints.DoNotTouch, want)
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
