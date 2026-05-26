package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/glincker/stacklit/internal/git"
	"github.com/glincker/stacklit/internal/schema"
)

func TestDiffReturnsNilWhenIndexIsCurrent(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	writeDiffFixture(t, root, "package main\n\nfunc main() {}\n")
	writeDiffIndex(t, root, "stacklit.json", "main.go")

	cmd := newDiffCmd()
	cmd.SetArgs([]string{"-i", "stacklit.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("diff returned error for current index: %v", err)
	}
}

func TestDiffReturnsStaleExitWhenIndexChanged(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	writeDiffFixture(t, root, "package main\n\nfunc main() {}\n")
	writeDiffIndex(t, root, "stacklit.json", "main.go")
	writeDiffFixture(t, root, "package main\n\nfunc main() { println(\"changed\") }\n")

	cmd := newDiffCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"-i", "stacklit.json"})

	code, reportErr := ErrorToExit(cmd.Execute())
	if code != ExitStale {
		t.Fatalf("expected stale exit %d, got %d", ExitStale, code)
	}
	if reportErr != nil {
		t.Fatalf("stale diff should not report an operational error: %v", reportErr)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stale diff should not write to stderr, got %q", stderr.String())
	}
}

func TestDiffReturnsFailureExitWhenIndexCannotBeRead(t *testing.T) {
	root := t.TempDir()
	restoreWorkingDirectory(t, root)
	writeDiffFixture(t, root, "package main\n\nfunc main() {}\n")

	cmd := newDiffCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"-i", "missing.json"})

	code, reportErr := ErrorToExit(cmd.Execute())
	if code != ExitFailure {
		t.Fatalf("expected failure exit %d, got %d", ExitFailure, code)
	}
	if reportErr == nil {
		t.Fatal("expected missing index to report an operational error")
	}
	if stderr.Len() != 0 {
		t.Fatalf("diff command should leave error reporting to main, got %q", stderr.String())
	}
}

func writeDiffFixture(t *testing.T, root, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(contents), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
}

func writeDiffIndex(t *testing.T, root, name string, files ...string) {
	t.Helper()

	contents := make(map[string][]byte, len(files))
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatalf("reading fixture %s: %v", file, err)
		}
		contents[file] = data
	}

	index := schema.Index{MerkleHash: git.ComputeMerkle(files, contents)}
	data, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshaling index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
		t.Fatalf("writing index: %v", err)
	}
}
