package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFrameworks_ConfigAndImports(t *testing.T) {
	// Create a temp dir with next.config.js.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}

	imports := []string{"react", "express", "net/http"}
	got := DetectFrameworks(dir, imports)

	want := map[string]bool{
		"Next.js":  true,
		"React":    true,
		"Express":  true,
		"net/http": true,
	}

	if len(got) != len(want) {
		t.Errorf("got %d frameworks, want %d: %v", len(got), len(want), got)
	}
	for _, fw := range got {
		if !want[fw] {
			t.Errorf("unexpected framework: %q", fw)
		}
	}
	for name := range want {
		found := false
		for _, fw := range got {
			if fw == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing framework: %q", name)
		}
	}
}

func TestDetectFrameworks_PackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{
		"dependencies": {"react": "^18.0.0", "express": "^4.0.0"},
		"devDependencies": {"jest": "^29.0.0", "tailwindcss": "^3.0.0"}
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	got := DetectFrameworks(dir, nil)

	want := map[string]bool{
		"React":        true,
		"Express":      true,
		"Jest":         true,
		"Tailwind CSS": true,
	}

	for name := range want {
		found := false
		for _, fw := range got {
			if fw == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing framework from package.json: %q, got: %v", name, got)
		}
	}
}

func TestDetectFrameworks_Sorted(t *testing.T) {
	dir := t.TempDir()
	imports := []string{"express", "react", "net/http"}
	got := DetectFrameworks(dir, imports)

	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Errorf("result not sorted: %v", got)
			break
		}
	}
}

func TestDetectFrameworks_Deduplication(t *testing.T) {
	// mongo and mongodb should both map to MongoDB but appear only once.
	dir := t.TempDir()
	imports := []string{"mongodb", "mongo"}
	got := DetectFrameworks(dir, imports)

	count := 0
	for _, fw := range got {
		if fw == "MongoDB" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected MongoDB exactly once, got %d times in %v", count, got)
	}
}

func TestDetectFrameworks_GitHubActions(t *testing.T) {
	dir := t.TempDir()
	// Create .github/workflows directory.
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0755); err != nil {
		t.Fatal(err)
	}

	got := DetectFrameworks(dir, nil)

	found := false
	for _, fw := range got {
		if fw == "GitHub Actions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected GitHub Actions to be detected, got: %v", got)
	}
}

func TestDetectFrameworks_Empty(t *testing.T) {
	dir := t.TempDir()
	got := DetectFrameworks(dir, nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for empty dir, got: %v", got)
	}
}
