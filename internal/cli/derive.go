package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/glincker/stacklit/internal/derive"
	"github.com/glincker/stacklit/internal/schema"
	"github.com/spf13/cobra"
)

var deriveInput string

func newDeriveCmd() *cobra.Command {
	var includeAISummary bool

	cmd := &cobra.Command{
		Use:   "derive",
		Short: "Generate a compact codebase navigation map (~250 tokens)",
		Long: `Generates a token-efficient navigation map from stacklit.json.

The map contains architecture, modules, dependencies, and hints in ~250 tokens,
replacing 3,000-8,000 tokens of agent exploration per session.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load index file.
			data, err := os.ReadFile(deriveInput)
			if err != nil {
				return fmt.Errorf("could not read %s: %w (run 'stacklit generate-json' first)", deriveInput, err)
			}
			var idx schema.Index
			if err := json.Unmarshal(data, &idx); err != nil {
				return fmt.Errorf("could not parse %s: %w", deriveInput, err)
			}

			output, err := derive.CompactMapWithOptions(&idx, derive.CompactMapOptions{
				IncludeAISummary: includeAISummary,
			})
			if err != nil {
				if errors.Is(err, derive.ErrMissingAISummary) {
					return fmt.Errorf("%s has no architecture.ai_summary; run 'stacklit ai-summary' then 'stacklit generate-json'", deriveInput)
				}
				return err
			}
			fmt.Print(output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&deriveInput, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
	cmd.Flags().BoolVar(&includeAISummary, "ai-summary", false, "Include architecture.ai_summary from the index")
	return cmd
}
