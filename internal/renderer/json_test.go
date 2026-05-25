package renderer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/glincker/stacklit/internal/schema"
)

func TestWriteJSON(t *testing.T) {
	idx := &schema.Index{
		Version: "1",
		Project: schema.Project{
			Name: "my-project",
			Root: "/home/user/my-project",
			Type: "application",
		},
		Tech: schema.Tech{
			PrimaryLanguage: "Go",
			Languages: map[string]schema.LangStats{
				"Go": {Files: 10, Lines: 500},
			},
		},
		Structure: schema.Structure{
			Entrypoints: []string{"cmd/main.go"},
			TotalFiles:  10,
			TotalLines:  500,
		},
		Modules: map[string]schema.ModuleInfo{
			"cmd": {Purpose: "entry point", Files: 1},
		},
		Dependencies: schema.Dependencies{
			Edges:       [][2]string{{"cmd", "internal/renderer"}},
			Entrypoints: []string{"cmd"},
		},
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "stacklit.json")

	if err := WriteJSON(idx, outPath); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Valid JSON
	var decoded schema.Index
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if raw[len(raw)-1] != '\n' {
		t.Fatalf("output should end with newline, got final byte %q", raw[len(raw)-1])
	}

	// $schema is set
	if decoded.Schema != schemaURL {
		t.Errorf("expected $schema %q, got %q", schemaURL, decoded.Schema)
	}

	// project name matches
	if decoded.Project.Name != "my-project" {
		t.Errorf("expected project name %q, got %q", "my-project", decoded.Project.Name)
	}

	// generated_at is populated
	if decoded.GeneratedAt == "" {
		t.Error("expected generated_at to be set, got empty string")
	}

	// stacklit_version is set
	if decoded.StacklitVersion != version {
		t.Errorf("expected stacklit_version %q, got %q", version, decoded.StacklitVersion)
	}
}
