package insights

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/glincker/stacklit/internal/derive"
	"github.com/glincker/stacklit/internal/git"
	"github.com/glincker/stacklit/internal/hint"
	"github.com/glincker/stacklit/internal/jsonfile"
	"github.com/glincker/stacklit/internal/purpose"
	"github.com/glincker/stacklit/internal/schema"
)

const DefaultPath = "stacklit-insights.json"

// File contains curated semantic knowledge that enriches a generated index.
type File struct {
	Purpose      map[string]string   `json:"purpose,omitempty"`
	Hints        schema.Hints        `json:"hints,omitempty"`
	Architecture schema.Architecture `json:"architecture,omitempty"`
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	ensurePurpose(&file)
	return &file, nil
}

func LoadIfExists(path string) (*File, bool, error) {
	file, err := Load(path)
	if err == nil {
		return file, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return &File{Purpose: map[string]string{}}, false, nil
	}
	return nil, false, err
}

func Write(path string, file *File) error {
	ensurePurpose(file)
	data, err := jsonfile.MarshalIndent(file)
	if err != nil {
		return fmt.Errorf("marshaling insights: %w", err)
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating insights directory: %w", err)
		}
	}
	return os.WriteFile(path, data, 0644)
}

func Apply(idx *schema.Index, file *File) {
	if idx == nil || file == nil {
		return
	}
	for name, mod := range idx.Modules {
		if purpose := purposeFor(file, name); purpose != "" {
			mod.Purpose = purpose
			idx.Modules[name] = mod
		}
	}
	applyHints(&idx.Hints, file.Hints)
	applyArchitecture(&idx.Architecture, file.Architecture)
}

func Merge(target *File, source *File, idx *schema.Index) {
	if target == nil || source == nil {
		return
	}
	ensurePurpose(target)
	prunePurpose(target.Purpose, idx)
	for name, generatedPurpose := range source.Purpose {
		if !purposeMatchesIndex(name, idx) {
			continue
		}
		if shouldMergeGeneratedPurpose(name, target.Purpose[name], generatedPurpose) {
			target.Purpose[name] = generatedPurpose
		}
	}
	mergeHints(&target.Hints, source.Hints, idx)
	mergeArchitecture(&target.Architecture, source.Architecture)
}

func shouldMergeGeneratedPurpose(module, existing, generated string) bool {
	if generated == "" {
		return false
	}
	if existing == "" {
		return true
	}
	return existing == purpose.Infer(module)
}

func SeedFromIndex(file *File, idx *schema.Index, prune bool) {
	ensurePurpose(file)
	if idx == nil {
		return
	}

	if prune {
		prunePurpose(file.Purpose, idx)
	}

	for name, mod := range idx.Modules {
		if _, ok := file.Purpose[name]; !ok && mod.Purpose != "" {
			file.Purpose[name] = mod.Purpose
		}
	}
	seedHints(&file.Hints, idx.Hints)
	seedArchitecture(&file.Architecture, idx.Architecture)
}

func ensurePurpose(file *File) {
	if file.Purpose == nil {
		file.Purpose = map[string]string{}
	}
}

func prunePurpose(purposes map[string]string, idx *schema.Index) {
	if idx == nil {
		return
	}
	for name := range purposes {
		if !purposeMatchesIndex(name, idx) {
			delete(purposes, name)
		}
	}
}

// PruneHints removes do_not_touch paths that no longer exist under root and
// env_vars that no tracked file mentions outside the root-relative exclude
// paths and injected stacklit blocks. It keeps entries whenever existence or
// references cannot be established: glob paths, stat errors other than
// not-found, and any failure to search git or read a matching file.
func PruneHints(file *File, root string, exclude []string) {
	file.Hints.DoNotTouch = slices.DeleteFunc(file.Hints.DoNotTouch, func(p string) bool {
		return pathMissing(root, p)
	})

	referenced, ok := referencedWords(root, file.Hints.EnvVars, exclude)
	if !ok {
		return
	}
	file.Hints.EnvVars = slices.DeleteFunc(file.Hints.EnvVars, func(name string) bool {
		return !referenced[name]
	})
}

func pathMissing(root, p string) bool {
	if p == "" || strings.ContainsAny(p, "*?[") {
		return false
	}
	path := filepath.FromSlash(p)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}

// referencedWords reports which words appear as whole words in tracked files
// under root once injected stacklit blocks are removed. ok is false when the
// references cannot be established.
func referencedWords(root string, words, exclude []string) (map[string]bool, bool) {
	if len(words) == 0 {
		return nil, false
	}
	candidates, err := git.FilesContainingWords(root, words, exclude)
	if err != nil {
		return nil, false
	}

	referenced := make(map[string]bool, len(words))
	for _, rel := range candidates {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, false
		}
		text := derive.StripBlock(string(data))
		for _, w := range words {
			if !referenced[w] && regexp.MustCompile(`\b`+regexp.QuoteMeta(w)+`\b`).MatchString(text) {
				referenced[w] = true
			}
		}
	}
	return referenced, true
}

func purposeMatchesIndex(name string, idx *schema.Index) bool {
	if idx == nil {
		return true
	}
	if _, ok := idx.Modules[name]; ok {
		return true
	}
	for module := range idx.Modules {
		if filepath.Base(module) == name {
			return true
		}
	}
	return false
}

func purposeFor(file *File, module string) string {
	if file.Purpose == nil {
		return ""
	}
	if purpose := file.Purpose[module]; purpose != "" {
		return purpose
	}
	if purpose := file.Purpose[filepath.Base(module)]; purpose != "" {
		return purpose
	}
	return ""
}

func applyHints(target *schema.Hints, source schema.Hints) {
	if source.AddFeature != "" {
		target.AddFeature = source.AddFeature
	}
	if source.TestCmd != "" {
		target.TestCmd = source.TestCmd
	}
	target.EnvVars = unionStrings(target.EnvVars, source.EnvVars)
	target.DoNotTouch = unionStrings(target.DoNotTouch, source.DoNotTouch)
}

func mergeHints(target *schema.Hints, source schema.Hints, idx *schema.Index) {
	if shouldMergeGeneratedAddFeature(target.AddFeature, source.AddFeature, idx) {
		target.AddFeature = source.AddFeature
	}
	if target.TestCmd == "" && source.TestCmd != "" {
		target.TestCmd = source.TestCmd
	}
	target.EnvVars = unionStrings(target.EnvVars, source.EnvVars)
	target.DoNotTouch = unionStrings(target.DoNotTouch, source.DoNotTouch)
}

func shouldMergeGeneratedAddFeature(existing, generated string, idx *schema.Index) bool {
	if generated == "" {
		return false
	}
	if existing == "" {
		return true
	}
	if idx == nil {
		return false
	}
	return existing == hint.AddFeature(idx.Modules, idx.Structure.Entrypoints)
}

func seedHints(target *schema.Hints, source schema.Hints) {
	if target.AddFeature == "" {
		target.AddFeature = source.AddFeature
	}
	if target.TestCmd == "" {
		target.TestCmd = source.TestCmd
	}
	if len(target.EnvVars) == 0 {
		target.EnvVars = source.EnvVars
	}
	if len(target.DoNotTouch) == 0 {
		target.DoNotTouch = source.DoNotTouch
	}
}

func applyArchitecture(target *schema.Architecture, source schema.Architecture) {
	if source.Pattern != "" {
		target.Pattern = source.Pattern
	}
	if source.Summary != "" {
		target.Summary = source.Summary
	}
}

func mergeArchitecture(target *schema.Architecture, source schema.Architecture) {
	if target.Pattern == "" && source.Pattern != "" {
		target.Pattern = source.Pattern
	}
	if source.Summary != "" {
		target.Summary = source.Summary
	}
}

func seedArchitecture(target *schema.Architecture, source schema.Architecture) {
	if target.Pattern == "" {
		target.Pattern = source.Pattern
	}
	if target.Summary == "" {
		target.Summary = source.Summary
	}
}

func unionStrings(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	out := slices.Clone(base)
	for _, item := range extra {
		if item != "" && !slices.Contains(out, item) {
			out = append(out, item)
		}
	}
	return out
}
