package insights

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/glincker/stacklit/internal/jsonfile"
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

func SeedFromIndex(file *File, idx *schema.Index, prune bool) {
	ensurePurpose(file)
	if idx == nil {
		return
	}

	if prune {
		for name := range file.Purpose {
			if _, ok := idx.Modules[name]; !ok {
				delete(file.Purpose, name)
			}
		}
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
