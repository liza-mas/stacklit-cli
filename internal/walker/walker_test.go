package walker_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/glincker/stacklit/internal/walker"
)

// fixtureDir returns the absolute path to the go-project test fixture.
func fixtureDir(t *testing.T) string {
	t.Helper()
	// Walk up from the test file's package directory to the repo root.
	abs, err := filepath.Abs("../../testdata/go-project")
	if err != nil {
		t.Fatalf("resolve fixture dir: %v", err)
	}
	return abs
}

func TestWalkSourceFiles(t *testing.T) {
	root := fixtureDir(t)
	files, err := walker.Walk(root, nil)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	// Must include Go source files.
	wantIncluded := []string{
		"main.go",
		filepath.ToSlash(filepath.Join("internal", "handler.go")),
		filepath.ToSlash(filepath.Join("internal", "handler_test.go")),
	}
	for _, want := range wantIncluded {
		if !fileSet[want] {
			t.Errorf("expected %q to be included, got files: %v", want, sortedKeys(fileSet))
		}
	}

	// Must exclude gitignored directories.
	wantExcluded := []string{
		filepath.ToSlash(filepath.Join("vendor", "lib.go")),
		filepath.ToSlash(filepath.Join("node_modules", "pkg.js")),
	}
	for _, exclude := range wantExcluded {
		if fileSet[exclude] {
			t.Errorf("expected %q to be excluded (gitignored), but it was included", exclude)
		}
	}

	// Must exclude non-source files.
	nonSource := []string{"README.md", "image.png", ".gitignore"}
	for _, ns := range nonSource {
		if fileSet[ns] {
			t.Errorf("expected non-source file %q to be excluded, but it was included", ns)
		}
	}
}

func TestWalkReturnsRelativePaths(t *testing.T) {
	root := fixtureDir(t)
	files, err := walker.Walk(root, nil)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	for _, f := range files {
		if filepath.IsAbs(f) {
			t.Errorf("expected relative path but got absolute: %q", f)
		}
	}
}

func TestWalkEmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := walker.Walk(dir, nil)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d: %v", len(files), files)
	}
}

func TestWalkAlwaysIgnoresDirs(t *testing.T) {
	// Build a temp directory with always-ignore dirs that have no .gitignore.
	dir := t.TempDir()

	ignoredDirs := []string{"node_modules", ".git", "vendor", "dist", "build", "__pycache__"}
	for _, d := range ignoredDirs {
		subdir := filepath.Join(dir, d)
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(subdir, "file.go"), []byte("package foo\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// A legitimate source file at root.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := walker.Walk(dir, nil)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["main.go"] {
		t.Error("expected main.go to be included")
	}

	for _, d := range ignoredDirs {
		p := filepath.ToSlash(filepath.Join(d, "file.go"))
		if fileSet[p] {
			t.Errorf("expected %q inside always-ignore dir to be excluded", p)
		}
	}
}

// sortedKeys returns sorted map keys for readable error messages.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeFile creates dir/rel with minimal content, creating parent directories.
func writeFile(t *testing.T, dir, rel string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package foo\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

// runGit runs git in dir, isolated from the user's global and system config.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{
		"-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false",
	}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// newGitRepo returns an initialized, empty git repository, skipping the test
// when git is unavailable.
func newGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	return dir
}

func walkFiles(t *testing.T, root string, extraIgnore []string) []string {
	t.Helper()
	files, err := walker.Walk(root, extraIgnore)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}
	return files
}

func TestWalkGitIndexesOnlyTrackedAndStaged(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "tracked.go")
	runGit(t, dir, "add", "tracked.go")
	runGit(t, dir, "commit", "-q", "-m", "init")
	writeFile(t, dir, "staged.go")
	runGit(t, dir, "add", "staged.go")
	writeFile(t, dir, "untracked.go")
	writeFile(t, dir, "excluded/local.go")
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte("excluded/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := walkFiles(t, dir, nil)
	want := []string{"staged.go", "tracked.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %v, want %v", got, want)
	}
}

func TestWalkGitSkipsTrackedFilesDeletedFromWorkTree(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "kept.go")
	writeFile(t, dir, "deleted.go")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	if err := os.Remove(filepath.Join(dir, "deleted.go")); err != nil {
		t.Fatal(err)
	}

	got := walkFiles(t, dir, nil)
	want := []string{"kept.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %v, want %v", got, want)
	}
}

func TestWalkGitAppliesAlwaysIgnoreAndExtraIgnore(t *testing.T) {
	dir := newGitRepo(t)
	for _, rel := range []string{"main.go", "vendor/dep.go", "pkg/vendor/dep.go", "gen/out.go", "pkg/skip.go", "README.md"} {
		writeFile(t, dir, rel)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	got := walkFiles(t, dir, []string{"gen/", "pkg/skip.go"})
	want := []string{"main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %v, want %v", got, want)
	}
}

func TestWalkGitSubdirectoryRootReturnsRootRelativePaths(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "outside.go")
	writeFile(t, dir, "sub/inner/a.go")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	got := walkFiles(t, filepath.Join(dir, "sub"), nil)
	want := []string{"inner/a.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %v, want %v", got, want)
	}
}

func TestWalkGitUntrackedRootReturnsNoFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "main.go")
	runGit(t, dir, "add", "main.go")
	runGit(t, dir, "commit", "-q", "-m", "init")
	writeFile(t, dir, "local/untracked.go")
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte("local/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := walkFiles(t, filepath.Join(dir, "local"), nil); len(got) != 0 {
		t.Errorf("Walk = %v, want no files", got)
	}
}

func TestWalkGitListsUnmergedFileOnce(t *testing.T) {
	dir := newGitRepo(t)
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("package a\n")
	runGit(t, dir, "add", "a.go")
	runGit(t, dir, "commit", "-q", "-m", "base")
	runGit(t, dir, "checkout", "-q", "-b", "side")
	write("package a // side\n")
	runGit(t, dir, "commit", "-q", "-am", "side")
	runGit(t, dir, "checkout", "-q", "-")
	write("package a // main\n")
	runGit(t, dir, "commit", "-q", "-am", "main")
	// The merge is expected to fail with a conflict, leaving three index stages for a.go.
	merge := exec.Command("git", "-c", "user.name=test", "-c", "user.email=test@example.com", "merge", "-q", "side")
	merge.Dir = dir
	merge.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if err := merge.Run(); err == nil {
		t.Fatal("expected merge conflict")
	}
	unmerged := exec.Command("git", "ls-files", "--unmerged")
	unmerged.Dir = dir
	if out, err := unmerged.Output(); err != nil || len(bytes.Split(bytes.TrimSpace(out), []byte("\n"))) != 3 {
		t.Fatalf("expected three unmerged index stages, got %q (err %v)", out, err)
	}

	got := walkFiles(t, dir, nil)
	want := []string{"a.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %v, want %v", got, want)
	}
}

func TestWalkOutsideGitHonorsRootGitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go")
	writeFile(t, dir, "generated/x.go")
	writeFile(t, dir, "skip.go")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("generated/\nskip.go\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := walkFiles(t, dir, nil)
	want := []string{"main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %v, want %v", got, want)
	}
}
