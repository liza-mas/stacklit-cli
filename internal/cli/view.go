package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/glincker/stacklit/internal/renderer"
	"github.com/glincker/stacklit/internal/schema"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View the generated stacklit.json index",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(viewInput)
		if err != nil {
			return fmt.Errorf("reading %s: %w (run `stacklit generate-json` first)", viewInput, err)
		}

		var idx schema.Index
		if err := json.Unmarshal(data, &idx); err != nil {
			return fmt.Errorf("parsing %s: %w", viewInput, err)
		}

		htmlPath := "stacklit.html"
		if err := renderer.WriteHTML(&idx, htmlPath); err != nil {
			return fmt.Errorf("writing HTML: %w", err)
		}

		fmt.Println("Opening visual map...")
		openBrowser(htmlPath)
		return nil
	},
}

var viewInput string

func init() {
	viewCmd.Flags().StringVarP(&viewInput, "input", "i", defaultIndexPath, "Path to the stacklit JSON index")
}

// openBrowser opens path in the default system browser.
func openBrowser(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		fmt.Printf("Open %s in your browser\n", path)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Open %s in your browser\n", path)
	}
}
