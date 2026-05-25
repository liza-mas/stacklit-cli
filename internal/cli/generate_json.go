package cli

import (
	"github.com/glincker/stacklit/internal/engine"
	"github.com/glincker/stacklit/internal/insights"
	"github.com/spf13/cobra"
)

func newGenerateJSONCmd() *cobra.Command {
	var (
		output       string
		workspace    string
		multiFile    string
		insightsPath string
	)

	cmd := &cobra.Command{
		Use:   "generate-json",
		Short: "Generate only the stacklit.json codebase index",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput := ""
			if cmd.Flags().Changed("output") {
				jsonOutput = output
			}
			warnMissingInsights := cmd.Flags().Changed("insights")
			if multiFile != "" {
				_, err := engine.RunMulti(engine.MultiOptions{
					ReposFile:           multiFile,
					Quiet:               true,
					Workspace:           workspace,
					JSONOnly:            true,
					OutputPath:          jsonOutput,
					InsightsPath:        insightsPath,
					WarnMissingInsights: warnMissingInsights,
				})
				return err
			}
			_, err := engine.Run(engine.Options{
				Root:                ".",
				Workspace:           workspace,
				Quiet:               true,
				JSONOnly:            true,
				JSONOutput:          jsonOutput,
				InsightsPath:        insightsPath,
				WarnMissingInsights: warnMissingInsights,
			})
			return err
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Path to write the JSON index (default: configured output.json)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Path to workspace root (default: current directory)")
	cmd.Flags().StringVar(&multiFile, "multi", "", "Path to file listing repos for polyrepo scanning")
	cmd.Flags().StringVar(&insightsPath, "insights", insights.DefaultPath, "Path to stacklit insights JSON")
	return cmd
}
