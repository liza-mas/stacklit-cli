package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/glincker/stacklit/internal/archexport"
	"github.com/glincker/stacklit/internal/jsonfile"
	"github.com/spf13/cobra"
)

type exportArchitectureOptions struct {
	input  string
	output string
}

func newExportArchitectureCmd() *cobra.Command {
	opts := &exportArchitectureOptions{}
	cmd := &cobra.Command{
		Use:   "export-architecture",
		Short: "Export architecture structure from a Stacklit index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportArchitecture(opts, cmd.OutOrStdout(), time.Now().UTC())
		},
	}
	cmd.Flags().StringVarP(&opts.input, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Write architecture export JSON to a file instead of stdout")
	return cmd
}

func runExportArchitecture(opts *exportArchitectureOptions, stdout io.Writer, generatedAt time.Time) error {
	data, idx, err := loadIndexData(opts.input)
	if err != nil {
		return err
	}

	export := archexport.Build(opts.input, data, idx, generatedAt, Version)
	if opts.output != "" {
		return jsonfile.WriteIndent(opts.output, export, 0644)
	}

	out, err := jsonfile.MarshalIndent(export)
	if err != nil {
		return fmt.Errorf("marshal architecture export: %w", err)
	}
	_, err = stdout.Write(out)
	return err
}
