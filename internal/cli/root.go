package cli

import (
	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:               "stacklit",
	Short:             "Generate a token-efficient codebase index for AI agents",
	Long:              "Generate a token-efficient codebase index for AI agents.\n\nUsage details: https://github.com/liza-mas/stacklit-cli/blob/main/USAGE.md",
	Version:           Version,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(newGenerateJSONCmd())
	rootCmd.AddCommand(newInitInsightsCmd())
	rootCmd.AddCommand(newAISummaryCmd())
	rootCmd.AddCommand(newFindModuleCmd())
	rootCmd.AddCommand(newGetDependenciesCmd())
	rootCmd.AddCommand(newGetHintsCmd())
	rootCmd.AddCommand(newGetHotFilesCmd())
	rootCmd.AddCommand(newGetModuleCmd())
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newDeriveCmd())
}
