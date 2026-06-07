package archexport

import (
	"crypto/sha256"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/glincker/stacklit/internal/schema"
)

const (
	SchemaVersion = "stacklit.architecture-export.v1"

	generatorName       = "stacklit"
	parserSource        = "parser"
	frameworkSource     = "framework_pattern"
	parserConfidence    = 0.8
	frameworkConfidence = 0.7
)

type Export struct {
	SchemaVersion string         `json:"schema_version"`
	Generator     Generator      `json:"generator"`
	GeneratedAt   string         `json:"generated_at"`
	Inputs        Inputs         `json:"inputs"`
	Packages      []Unit         `json:"packages"`
	Components    []Unit         `json:"components"`
	Membership    []Membership   `json:"membership"`
	Relationships []Relationship `json:"relationships"`
	EntryPoints   []EntryPoint   `json:"entry_points"`
}

type Generator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Inputs struct {
	StacklitIndex StacklitIndexInput `json:"stacklit_index"`
}

type StacklitIndexInput struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

type Unit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Membership struct {
	Path      string `json:"path"`
	Package   string `json:"package"`
	Component string `json:"component"`
}

type Relationship struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type EntryPoint struct {
	Path       string  `json:"path"`
	Kind       string  `json:"kind"`
	Source     string  `json:"entry_point_source"`
	Confidence float64 `json:"entry_point_confidence"`
}

func Build(indexPath string, indexBytes []byte, idx *schema.Index, generatedAt time.Time, generatorVersion string) Export {
	packages, components, membership := unitsAndMembership(idx.Modules)

	return Export{
		SchemaVersion: SchemaVersion,
		Generator: Generator{
			Name:    generatorName,
			Version: generatorVersion,
		},
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Inputs: Inputs{
			StacklitIndex: StacklitIndexInput{
				Path:        indexPath,
				Fingerprint: fingerprint(indexBytes),
			},
		},
		Packages:      packages,
		Components:    components,
		Membership:    membership,
		Relationships: relationships(idx.Dependencies.Edges),
		EntryPoints:   entryPoints(idx),
	}
}

func fingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}

func unitsAndMembership(modules map[string]schema.ModuleInfo) ([]Unit, []Unit, []Membership) {
	names := sortedModuleNames(modules)
	packages := make([]Unit, 0, len(names))
	components := make([]Unit, 0, len(names))
	membership := make([]Membership, 0)

	for _, name := range names {
		mod := modules[name]
		unit := Unit{
			ID:          name,
			Name:        name,
			Description: mod.Purpose,
		}
		packages = append(packages, unit)
		components = append(components, unit)

		files := append([]string(nil), mod.FileList...)
		sort.Strings(files)
		for _, file := range files {
			memberPath := moduleFilePath(name, file)
			if memberPath == "" {
				continue
			}
			membership = append(membership, Membership{
				Path:      memberPath,
				Package:   name,
				Component: name,
			})
		}
	}

	return packages, components, membership
}

func sortedModuleNames(modules map[string]schema.ModuleInfo) []string {
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func moduleFilePath(module, file string) string {
	cleanFile := strings.TrimPrefix(path.Clean(file), "./")
	if cleanFile == "." || cleanFile == "" {
		return ""
	}
	if module == "" || module == "." || module == "root" {
		return cleanFile
	}
	return path.Join(module, cleanFile)
}

func relationships(edges [][2]string) []Relationship {
	out := make([]Relationship, 0, len(edges))
	for _, edge := range edges {
		out = append(out, Relationship{
			Source: edge[0],
			Target: edge[1],
			Type:   "package_dependency",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func entryPoints(idx *schema.Index) []EntryPoint {
	seen := make(map[string]EntryPoint)

	for _, entry := range idx.Structure.Entrypoints {
		addEntryPoint(seen, entry, parserSource, parserConfidence)
	}
	for _, entry := range idx.Dependencies.Entrypoints {
		addEntryPoint(seen, entry, parserSource, parserConfidence)
	}
	for _, pattern := range idx.Tech.FrameworkPatterns {
		addEntryPoint(seen, pattern.Entry, frameworkSource, frameworkConfidence)
	}

	out := make([]EntryPoint, 0, len(seen))
	for _, entry := range seen {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func addEntryPoint(seen map[string]EntryPoint, entryPath, source string, confidence float64) {
	cleanPath := strings.TrimPrefix(path.Clean(entryPath), "./")
	if cleanPath == "." || cleanPath == "" {
		return
	}
	kind := entryPointKind(cleanPath, source)
	key := cleanPath + "\x00" + source + "\x00" + kind
	seen[key] = EntryPoint{
		Path:       cleanPath,
		Kind:       kind,
		Source:     source,
		Confidence: confidence,
	}
}

func entryPointKind(entryPath, source string) string {
	if source == frameworkSource {
		return "service"
	}
	lower := strings.ToLower(entryPath)
	if strings.Contains(lower, "controller") {
		return "controller"
	}
	if strings.Contains(lower, "/api/") || strings.Contains(lower, "router") || strings.Contains(lower, "routes") {
		return "public_api"
	}
	return "command"
}
