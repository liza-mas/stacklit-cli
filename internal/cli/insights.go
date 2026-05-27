package cli

import (
	"errors"
	"fmt"
	"os"

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
			if err := insights.Write(output, file); err != nil {
				return err
			}
			fmt.Printf("Wrote %s\n", output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
	cmd.Flags().StringVarP(&output, "output", "o", insights.DefaultPath, "Path to write the insights JSON")
	cmd.Flags().BoolVar(&prune, "prune", false, "Remove purpose entries for modules no longer in the index")
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
			generated, err := summary.Run(idx)
			if err != nil {
				return err
			}
			insights.Merge(file, generated)
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
