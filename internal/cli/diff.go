package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/glincker/stacklit/internal/config"
	"github.com/glincker/stacklit/internal/git"
	"github.com/glincker/stacklit/internal/schema"
	"github.com/glincker/stacklit/internal/walker"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var input string

	cmd := &cobra.Command{
		Use:           "diff",
		Short:         "Show changes since last index generation",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Load config and read the index file.
			cfg := config.Load(".")
			indexPath := cfg.Output.JSON
			if indexPath == "" {
				indexPath = defaultIndexPath
			}
			if cmd.Flags().Changed("input") {
				indexPath = input
			}

			data, err := os.ReadFile(indexPath)
			if err != nil {
				return newExitError(ExitFailure, fmt.Errorf("could not read %s: %w (run 'stacklit generate-json' first)", indexPath, err))
			}

			var index schema.Index
			if err := json.Unmarshal(data, &index); err != nil {
				return newExitError(ExitFailure, fmt.Errorf("could not parse %s: %w", indexPath, err))
			}

			storedHash := index.MerkleHash
			if storedHash == "" {
				return newExitError(ExitFailure, fmt.Errorf("%s has no merkle_hash; run 'stacklit generate-json' to rebuild", indexPath))
			}

			// 2. Walk current source files, excluding Stacklit's own generated outputs.
			ignore := append(cfg.ScanIgnore(), indexPath)
			files, err := walker.Walk(".", ignore)
			if err != nil {
				return newExitError(ExitFailure, fmt.Errorf("failed to walk source files: %w", err))
			}

			// 3. Read file contents and compute fresh Merkle hash
			contents := make(map[string][]byte, len(files))
			for _, f := range files {
				b, err := os.ReadFile(f)
				if err != nil {
					return newExitError(ExitFailure, fmt.Errorf("could not read %s: %w", f, err))
				}
				contents[f] = b
			}

			currentHash := git.ComputeMerkle(files, contents)

			// 4. Compare hashes
			if currentHash == storedHash {
				fmt.Println("Index is up to date. No source changes detected.")
				return nil
			}

			// 5. Hashes differ — report and suggest regeneration
			fmt.Println("Source files changed since last generation. Run 'stacklit generate-json' to update.")
			return newExitError(ExitStale, nil)
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", "", "Path to the stacklit JSON index (default: configured output.json)")
	return cmd
}

// comment
