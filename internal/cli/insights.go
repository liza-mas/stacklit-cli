package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glincker/stacklit/internal/config"
	"github.com/glincker/stacklit/internal/engine"
	"github.com/glincker/stacklit/internal/insights"
	"github.com/glincker/stacklit/internal/schema"
	"github.com/glincker/stacklit/internal/summary"
	"github.com/spf13/cobra"
)

func newInitInsightsCmd() *cobra.Command {
	var input string
	var output string
	var prune bool

	cmd := &cobra.Command{
		Use:   "init-insights",
		Short: "Create or update stacklit-insights.json",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexOrGenerate(input)
			if err != nil {
				return err
			}

			file, _, err := insights.LoadIfExists(output)
			if err != nil {
				return err
			}
			insights.SeedFromIndex(file, idx, prune)
			if prune {
				insights.PruneHints(file, projectRoot, stacklitOutputs(projectRoot, input, output))
			}
			if err := insights.Write(output, file); err != nil {
				return err
			}
			fmt.Printf("Wrote %s\n", output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
	cmd.Flags().StringVarP(&output, "output", "o", insights.DefaultPath, "Path to write the insights JSON")
	cmd.Flags().BoolVar(&prune, "prune", false, "Remove stale purpose entries, missing do_not_touch paths and unreferenced env_vars")
	return cmd
}

func newAISummaryCmd() *cobra.Command {
	var input string
	var output string

	cmd := &cobra.Command{
		Use:   "ai-summary",
		Short: "Generate an AI architecture summary into stacklit-insights.json",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := loadIndexFile(input)
			if err != nil {
				return fmt.Errorf("%w (run 'stacklit generate-json' first)", err)
			}

			file, _, err := insights.LoadIfExists(output)
			if err != nil {
				return err
			}
			generated, err := summary.Run(idx, output, projectRoot)
			if err != nil {
				return err
			}
			insights.Merge(file, generated, idx)
			insights.PruneHints(file, projectRoot, stacklitOutputs(projectRoot, input, output))
			if err := insights.Write(output, file); err != nil {
				return err
			}
			fmt.Printf("Wrote %s\n", output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
	cmd.Flags().StringVarP(&output, "output", "o", insights.DefaultPath, "Path to write the insights JSON")
	return cmd
}

// projectRoot is the directory the insights commands describe: the current
// directory, which generate-json scans, wherever the index file is stored.
const projectRoot = "."

// stacklitOutputs returns the root-relative paths of files Stacklit writes, so
// generated copies of insights never count as references to them.
// input and output are relative to the working directory; configured outputs
// are relative to root.
func stacklitOutputs(root, input, output string) []string {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	candidates := []string{input, output}
	cfg := config.Load(root)
	for _, p := range []string{cfg.Output.JSON, cfg.Output.Mermaid, cfg.Output.HTML} {
		if p != "" && !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		candidates = append(candidates, p)
	}

	var paths []string
	for _, p := range candidates {
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(absRoot, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		paths = append(paths, filepath.ToSlash(rel))
	}
	return paths
}

func loadIndexOrGenerate(path string) (*schema.Index, error) {
	idx, err := loadIndexFile(path)
	if err == nil {
		return idx, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	result, err := engine.Run(engine.Options{
		Root:      ".",
		Quiet:     true,
		JSONOnly:  true,
		SkipWrite: true,
	})
	if err != nil {
		return nil, err
	}
	return result.Index, nil
}
