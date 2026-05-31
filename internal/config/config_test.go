package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	cfg := Load(t.TempDir())
	if cfg.MaxDepth != 4 {
		t.Errorf("expected default max_depth=4, got %d", cfg.MaxDepth)
	}
	if cfg.MaxModules != 200 {
		t.Errorf("expected default max_modules=200, got %d", cfg.MaxModules)
	}
	if cfg.MaxExports != 10 {
		t.Errorf("expected default max_exports=10, got %d", cfg.MaxExports)
	}
	if cfg.Output.JSON != "stacklit.json" {
		t.Errorf("expected default output.json=stacklit.json, got %q", cfg.Output.JSON)
	}
	if cfg.Output.Mermaid != "DEPENDENCIES.md" {
		t.Errorf("expected default output.mermaid=DEPENDENCIES.md, got %q", cfg.Output.Mermaid)
	}
	if cfg.Output.HTML != "stacklit.html" {
		t.Errorf("expected default output.html=stacklit.html, got %q", cfg.Output.HTML)
	}
	if cfg.ParseWorkers != 1 {
		t.Errorf("expected default parse_workers=1, got %d", cfg.ParseWorkers)
	}
}

func TestLoadCustom(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"max_depth": 6, "ignore": ["custom/"]}`)
	cfg := Load(dir)
	if cfg.MaxDepth != 6 {
		t.Errorf("expected max_depth=6, got %d", cfg.MaxDepth)
	}
	if len(cfg.Ignore) != 1 {
		t.Errorf("expected 1 ignore pattern, got %d: %v", len(cfg.Ignore), cfg.Ignore)
	}
	// Unset fields should still use defaults.
	if cfg.MaxModules != 200 {
		t.Errorf("expected default max_modules=200, got %d", cfg.MaxModules)
	}
	if cfg.ParseWorkers != 1 {
		t.Errorf("expected omitted parse_workers to default to 1, got %d", cfg.ParseWorkers)
	}
}

func TestLoadValidatedParseWorkersOmitted(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"max_depth": 6, "ignore": ["custom/"]}`)

	cfg, err := LoadValidated(dir)
	if err != nil {
		t.Fatalf("expected omitted parse_workers to load, got error: %v", err)
	}
	if cfg.ParseWorkers != 1 {
		t.Fatalf("expected omitted parse_workers to default to 1, got %d", cfg.ParseWorkers)
	}
}

func TestLoadValidatedParseWorkersConfigured(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"parse_workers": 4}`)

	cfg, err := LoadValidated(dir)
	if err != nil {
		t.Fatalf("expected configured parse_workers to load, got error: %v", err)
	}
	if cfg.ParseWorkers != 4 {
		t.Fatalf("expected parse_workers=4, got %d", cfg.ParseWorkers)
	}
}

func TestLoadValidatedRejectsInvalidParseWorkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "zero", content: `{"parse_workers": 0}`},
		{name: "negative", content: `{"parse_workers": -2}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.content)

			_, err := LoadValidated(dir)
			if err == nil {
				t.Fatal("expected invalid parse_workers to fail validation")
			}
			if !strings.Contains(err.Error(), "parse_workers") {
				t.Fatalf("expected error to name parse_workers, got %q", err.Error())
			}
		})
	}
}

func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `not json`)
	cfg := Load(dir)
	// Should fall back to defaults without panicking.
	if cfg.MaxDepth != 4 {
		t.Errorf("expected default max_depth=4 after malformed file, got %d", cfg.MaxDepth)
	}
}

func TestScanIgnoreIncludesOutputs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Ignore = []string{"custom/", "vendor\\libs"}
	cfg.Output.JSON = "out\\stacklit.json"
	cfg.Output.Mermaid = "docs\\DEPENDENCIES.md"
	cfg.Output.HTML = "stacklit.html"

	got := cfg.ScanIgnore()
	want := []string{"custom/", "vendor/libs", "out/stacklit.json", "docs/DEPENDENCIES.md", "stacklit.html"}
	if len(got) != len(want) {
		t.Fatalf("expected %d ignore patterns, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected ignore[%d]=%q, got %q (all=%v)", i, want[i], got[i], got)
		}
	}
}

func writeConfig(t *testing.T, dir string, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, ".stacklitrc.json"), []byte(contents), 0644); err != nil {
		t.Fatalf("write .stacklitrc.json: %v", err)
	}
}
