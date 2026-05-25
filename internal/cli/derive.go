package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/glincker/stacklit/internal/derive"
	"github.com/glincker/stacklit/internal/schema"
	"github.com/spf13/cobra"
)

var deriveInput string

func newDeriveCmd() *cobra.Command {
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

			fmt.Print(derive.CompactMap(&idx))
			return nil
		},
	}
	cmd.Flags().StringVarP(&deriveInput, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
	return cmd
}
