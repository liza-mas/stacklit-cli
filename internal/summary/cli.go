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

	defaultTimeoutSec = 5 * 60
	stderrTail        = 500
)

// Run invokes the configured agent CLI. When a prior AI summary exists, a second
// invocation reconciles it with the fresh summary against the current index.
func Run(idx *schema.Index, existing *insights.File, root string) (*insights.File, error) {
	docs, err := documentationDigest(root)
	if err != nil {
		return nil, fmt.Errorf("collecting documentation: %w", err)
	}
	commandPrefix := defaultCommandPrefix()
	if parts := strings.Fields(os.Getenv(envCmd)); len(parts) > 0 {
		commandPrefix = parts
	}
	purposes := existingPurposeContext(existing)
	command := summaryCommand(commandPrefix, freshPrompt(idx, docs, purposes))

	n, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(envTimeout)))
	timeout := time.Duration(cmp.Or(max(n, 0), defaultTimeoutSec)) * time.Second

	userJSON, err := json.Marshal(snapshotFor(idx))
	if err != nil {
		return nil, fmt.Errorf("marshalling index snapshot: %w", err)
	}

	generated, err := runSummaryCommand(timeout, command, string(userJSON))
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.Architecture.Summary == "" || generated.Architecture.Summary == "" {
		return generated, nil
	}

	reconcileJSON, err := json.Marshal(newReconcileRequest(idx, existing.Architecture.Summary, generated.Architecture.Summary))
	if err != nil {
		return nil, fmt.Errorf("marshalling summary reconciliation: %w", err)
	}
	reconciled, err := runSummaryCommand(timeout, summaryCommand(commandPrefix, reconcilePrompt(idx, docs, purposes)), string(reconcileJSON))
	if err != nil {
		return nil, err
	}
	if reconciled.Architecture.Summary == "" {
		return nil, fmt.Errorf("summary CLI %q produced no reconciled architecture.ai_summary", command[0])
	}
	generated.Architecture.Summary = reconciled.Architecture.Summary
	return generated, nil
}

func runSummaryCommand(timeout time.Duration, command []string, userJSON string) (*insights.File, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = strings.NewReader(userJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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

func defaultCommandPrefix() []string {
	return []string{"claude", "-p", "--system-prompt"}
}

func summaryCommand(prefix []string, prompt string) []string {
	command := make([]string, 0, len(prefix)+1)
	command = append(command, prefix...)
	command = append(command, prompt)
	return command
}

func parseInsightsOutput(text string) (*insights.File, error) {
	generated, err := parseInsightsJSON(strings.TrimSpace(text))
	if err == nil {
		return generated, nil
	}

	jsonText, ok := firstJSONObject(text)
	if !ok {
		return nil, err
	}
	return parseInsightsJSON(jsonText)
}

func parseInsightsJSON(text string) (*insights.File, error) {
	dec := json.NewDecoder(strings.NewReader(text))
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

func firstJSONObject(text string) (string, bool) {
	start := strings.IndexByte(text, '{')
	if start == -1 {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '\\':
			if inString {
				escaped = !escaped
			}
		case '"':
			if !escaped {
				inString = !inString
			}
			escaped = false
		case '{':
			if !inString {
				depth++
			}
			escaped = false
		case '}':
			if !inString {
				depth--
				if depth == 0 {
					return text[start : i+1], true
				}
			}
			escaped = false
		default:
			escaped = false
		}
	}

	return "", false
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
