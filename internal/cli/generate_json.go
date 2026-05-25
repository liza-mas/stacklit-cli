package cli

import (
	"github.com/glincker/stacklit/internal/engine"
	"github.com/spf13/cobra"
)

func newGenerateJSONCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "generate-json",
		Short: "Generate only the stacklit.json codebase index",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput := ""
			if cmd.Flags().Changed("output") {
				jsonOutput = output
			}
			_, err := engine.Run(engine.Options{
				Root:       ".",
				Quiet:      true,
				JSONOnly:   true,
				JSONOutput: jsonOutput,
			})
			return err
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Path to write the JSON index (default: configured output.json)")
	return cmd
}
