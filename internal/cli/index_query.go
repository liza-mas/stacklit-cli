package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/glincker/stacklit/internal/schema"
	"github.com/spf13/cobra"
)

const defaultIndexPath = "stacklit.json"

type indexCommandOptions struct {
	input string
}

func addInputFlag(cmd *cobra.Command, opts *indexCommandOptions) {
	cmd.Flags().StringVarP(&opts.input, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
}

func loadIndexData(path string) ([]byte, *schema.Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var idx schema.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, nil, fmt.Errorf("could not parse %s: %w", path, err)
	}
	return data, &idx, nil
}

func loadIndexFile(path string) (*schema.Index, error) {
	_, idx, err := loadIndexData(path)
	return idx, err
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func newGetModuleCmd() *cobra.Command {
	opts := &indexCommandOptions{}
	cmd := &cobra.Command{
		Use:   "get-module <name>",
		Short: "Get full info for a specific module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexFile(opts.input)
			if err != nil {
				return err
			}
			name := args[0]
			mod, ok := idx.Modules[name]
			if !ok {
				return fmt.Errorf("module %q not found", name)
			}
			return printJSON(map[string]any{
				"name":   name,
				"module": mod,
			})
		},
	}
	addInputFlag(cmd, opts)
	return cmd
}

func newFindModuleCmd() *cobra.Command {
	opts := &indexCommandOptions{}
	cmd := &cobra.Command{
		Use:   "find-module <query>",
		Short: "Search modules by name or purpose",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexFile(opts.input)
			if err != nil {
				return err
			}
			return printJSON(findModules(idx, args[0]))
		},
	}
	addInputFlag(cmd, opts)
	return cmd
}

type moduleMatch struct {
	Name   string            `json:"name"`
	Module schema.ModuleInfo `json:"module"`
}

func findModules(idx *schema.Index, query string) []moduleMatch {
	q := strings.ToLower(query)
	names := make([]string, 0, len(idx.Modules))
	for name := range idx.Modules {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]moduleMatch, 0, 5)
	for _, name := range names {
		mod := idx.Modules[name]
		if strings.Contains(strings.ToLower(name), q) ||
			strings.Contains(strings.ToLower(mod.Purpose), q) {
			results = append(results, moduleMatch{Name: name, Module: mod})
		}
		if len(results) >= 5 {
			break
		}
	}
	return results
}

func newGetDependenciesCmd() *cobra.Command {
	opts := &indexCommandOptions{}
	cmd := &cobra.Command{
		Use:   "get-dependencies <module>",
		Short: "Get dependency edges for a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexFile(opts.input)
			if err != nil {
				return err
			}
			module := args[0]
			edges, err := dependencyEdges(idx, module)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{
				"module": module,
				"edges":  edges,
			})
		},
	}
	addInputFlag(cmd, opts)
	return cmd
}

func dependencyEdges(idx *schema.Index, module string) ([][2]string, error) {
	if _, ok := idx.Modules[module]; !ok {
		return nil, fmt.Errorf("module %q not found", module)
	}

	edges := make([][2]string, 0)
	for _, edge := range idx.Dependencies.Edges {
		if edge[0] == module || edge[1] == module {
			edges = append(edges, edge)
		}
	}
	return edges, nil
}

func newGetHotFilesCmd() *cobra.Command {
	opts := &indexCommandOptions{}
	cmd := &cobra.Command{
		Use:   "get-hot-files",
		Short: "Get the most frequently changed files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexFile(opts.input)
			if err != nil {
				return err
			}
			return printJSON(idx.Git.HotFiles)
		},
	}
	addInputFlag(cmd, opts)
	return cmd
}

func newGetHintsCmd() *cobra.Command {
	opts := &indexCommandOptions{}
	cmd := &cobra.Command{
		Use:   "get-hints",
		Short: "Get workflow hints and commands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexFile(opts.input)
			if err != nil {
				return err
			}
			return printJSON(idx.Hints)
		},
	}
	addInputFlag(cmd, opts)
	return cmd
}
