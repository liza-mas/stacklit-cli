package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// newGrepRepo creates a git repository holding files and adds all of them to
// the index, skipping the test when git is unavailable.
func newGrepRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH")
	}
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestFilesContainingWordsMatchesWholeWordsAndHonorsExclude(t *testing.T) {
	dir := newGrepRepo(t, map[string]string{
		"main.go":        `os.Getenv("API_KEY")`,
		"docs/usage.md":  "Set API_KEY first.",
		"prefix.go":      `os.Getenv("API_KEY_ID")`,
		"insights.json":  `["API_KEY"]`,
		"unrelated.go":   "package main",
		"nested/also.sh": "echo $API_KEY",
	})

	got, err := FilesContainingWords(dir, []string{"API_KEY"}, []string{"insights.json"})
	if err != nil {
		t.Fatalf("FilesContainingWords returned error: %v", err)
	}
	want := []string{"docs/usage.md", "main.go", "nested/also.sh"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilesContainingWords = %v, want %v", got, want)
	}
}

func TestFilesContainingWordsWithoutMatchReturnsNil(t *testing.T) {
	dir := newGrepRepo(t, map[string]string{"main.go": "package main"})

	got, err := FilesContainingWords(dir, []string{"MISSING_VAR"}, nil)
	if err != nil || got != nil {
		t.Errorf("FilesContainingWords = %v, %v; want nil, nil", got, err)
	}
}

func TestFilesContainingWordsOutsideGitReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`os.Getenv("API_KEY")`), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := FilesContainingWords(dir, []string{"API_KEY"}, nil); err == nil {
		t.Error("expected an error outside a git work tree")
	}
}
