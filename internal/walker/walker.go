// Package walker returns all source files worth indexing from a given root
// directory: the git-tracked files inside a git work tree, or a
// gitignore-aware filesystem walk otherwise.
package walker

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// sourceExtensions is the set of file extensions considered indexable source.
var sourceExtensions = map[string]bool{
	".go":      true,
	".ts":      true,
	".tsx":     true,
	".js":      true,
	".jsx":     true,
	".mjs":     true,
	".cjs":     true,
	".py":      true,
	".rs":      true,
	".java":    true,
	".cs":      true,
	".rb":      true,
	".php":     true,
	".swift":   true,
	".kt":      true,
	".scala":   true,
	".c":       true,
	".cpp":     true,
	".h":       true,
	".hpp":     true,
	".vue":     true,
	".svelte":  true,
	".dart":    true,
	".ex":      true,
	".exs":     true,
	".zig":     true,
	".nim":     true,
	".lua":     true,
	".sh":      true,
	".bash":    true,
	".sql":     true,
	".graphql": true,
	".gql":     true,
	".proto":   true,
}

// alwaysIgnoreDirs contains directory names that are always skipped regardless
// of .gitignore configuration.
var alwaysIgnoreDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".tox":         true,
	"target":       true,
	"coverage":     true,
	".cache":       true,
	".turbo":       true,
}

// Walk returns a sorted slice of root-relative paths for all source files under
// root. Inside a git work tree the candidates are the files in the git index
// (tracked plus staged), so untracked and git-ignored files are never indexed.
// When git is unavailable or root is not in a work tree, the filesystem is
// walked instead, honouring the root .gitignore. alwaysIgnoreDirs and source extensions apply in both cases.
// extraIgnore is an optional list of additional gitignore-style patterns to
// skip; it may be nil or empty.
func Walk(root string, extraIgnore []string) ([]string, error) {
	// Compile extra ignore patterns into a separate matcher when provided.
	var extraGi *gitignore.GitIgnore
	if len(extraIgnore) > 0 {
		extraGi = gitignore.CompileIgnoreLines(extraIgnore...)
	}

	if tracked, err := gitTrackedFiles(root); err == nil {
		return filterTracked(root, tracked, extraGi), nil
	}
	return walkFilesystem(root, extraGi)
}

// gitTrackedFiles lists the git index entries under root, relative to root,
// recursing into submodules. It errors when git is missing or root is not
// inside a git work tree.
func gitTrackedFiles(root string) ([]string, error) {
	out, err := exec.Command(
		"git", "-C", root,
		"ls-files", "-z", "--cached", "--recurse-submodules",
	).Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range bytes.Split(out, []byte{0}) {
		if len(entry) > 0 {
			files = append(files, string(entry))
		}
	}
	return files, nil
}

// filterTracked applies the same exclusions as walkFilesystem to git index
// entries, and drops entries missing from the working tree.
func filterTracked(root string, tracked []string, extraGi *gitignore.GitIgnore) []string {
	var results []string
	for _, rel := range tracked {
		if ignoredAncestor(rel, extraGi) {
			continue
		}
		if extraGi != nil && extraGi.MatchesPath(rel) {
			continue
		}
		if !sourceExtensions[filepath.Ext(rel)] {
			continue
		}
		// Tracked files deleted in the working tree have no content to index.
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		results = append(results, rel)
	}
	slices.Sort(results)
	// Unmerged paths appear once per conflict stage in the git index.
	return slices.Compact(results)
}

// ignoredAncestor reports whether any parent directory of rel would be skipped
// by walkFilesystem, matching directories in the same "dir/" form.
func ignoredAncestor(rel string, extraGi *gitignore.GitIgnore) bool {
	parts := strings.Split(rel, "/")
	for i := 0; i < len(parts)-1; i++ {
		if alwaysIgnoreDirs[parts[i]] {
			return true
		}
		if extraGi != nil && extraGi.MatchesPath(strings.Join(parts[:i+1], "/")+"/") {
			return true
		}
	}
	return false
}

// walkFilesystem traverses root and returns relative paths for all source
// files that are not excluded by the root .gitignore, alwaysIgnoreDirs, or
// extraGi.
func walkFilesystem(root string, extraGi *gitignore.GitIgnore) ([]string, error) {
	// Load .gitignore from root if present.
	var gi *gitignore.GitIgnore
	gitignorePath := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gi, _ = gitignore.CompileIgnoreFile(gitignorePath)
	}

	var results []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute path relative to root for all comparisons and results.
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		// Normalize path separators so generated output and ignore matching stay stable across OSes.
		rel = filepath.ToSlash(rel)

		// Skip the root itself.
		if rel == "." {
			return nil
		}

		name := d.Name()

		if d.IsDir() {
			// Always skip known noise directories.
			if alwaysIgnoreDirs[name] {
				return filepath.SkipDir
			}
			// Skip directories matched by .gitignore.
			if gi != nil && gi.MatchesPath(rel+"/") {
				return filepath.SkipDir
			}
			// Skip directories matched by extra ignore patterns.
			if extraGi != nil && extraGi.MatchesPath(rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files matched by .gitignore.
		if gi != nil && gi.MatchesPath(rel) {
			return nil
		}

		// Skip files matched by extra ignore patterns.
		if extraGi != nil && extraGi.MatchesPath(rel) {
			return nil
		}

		// Only keep recognised source extensions.
		ext := filepath.Ext(name)
		if !sourceExtensions[ext] {
			return nil
		}

		results = append(results, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}
