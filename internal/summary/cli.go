package summary

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/glincker/stacklit/internal/insights"
	"github.com/glincker/stacklit/internal/schema"
)

const (
	envCmd     = "STACKLIT_SUMMARY_CMD"
	envTimeout = "STACKLIT_SUMMARY_TIMEOUT"

	defaultTimeoutSec = 120
	stderrTail        = 500
)

// Run invokes the configured agent CLI and parses its stacklit-insights JSON.
func Run(idx *schema.Index) (*insights.File, error) {
	command := []string{"claude", "-p"}
	if parts := strings.Fields(os.Getenv(envCmd)); len(parts) > 0 {
		command = parts
	}

	n, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(envTimeout)))
	timeout := time.Duration(cmp.Or(max(n, 0), defaultTimeoutSec)) * time.Second

	snapshot := indexSnapshot{
		Project:      idx.Project,
		Tech:         idx.Tech,
		Modules:      idx.Modules,
		Dependencies: idx.Dependencies,
		Entrypoints:  idx.Structure.Entrypoints,
		Hints:        idx.Hints,
	}

	userJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshalling index snapshot: %w", err)
	}

	// Most agent CLIs lack a stdin system-prompt channel, so prepend inline.
	prompt := systemPrompt + "\n\n" + string(userJSON)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("summary CLI %q timed out after %s", command[0], timeout)
	}

	if err != nil {
		return nil, fmt.Errorf("summary CLI %q failed: %w: %.*s",
			command[0], err, stderrTail, strings.TrimSpace(stderr.String()))
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return nil, fmt.Errorf("summary CLI %q produced empty output (stderr: %.*s)",
			command[0], stderrTail, strings.TrimSpace(stderr.String()))
	}

	generated, err := parseInsightsOutput(text)
	if err != nil {
		return nil, fmt.Errorf("parsing summary CLI output: %w", err)
	}
	return generated, nil
}

func parseInsightsOutput(text string) (*insights.File, error) {
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(text)))
	var generated insights.File
	if err := dec.Decode(&generated); err != nil {
		return nil, fmt.Errorf("expected stacklit-insights JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("expected a single JSON object")
	}
	if isEmpty(generated) {
		return nil, fmt.Errorf("expected at least one generated insight")
	}
	return &generated, nil
}

func isEmpty(file insights.File) bool {
	return len(file.Purpose) == 0 &&
		file.Hints.AddFeature == "" &&
		file.Hints.TestCmd == "" &&
		len(file.Hints.EnvVars) == 0 &&
		len(file.Hints.DoNotTouch) == 0 &&
		file.Architecture.Pattern == "" &&
		file.Architecture.Summary == ""
}
