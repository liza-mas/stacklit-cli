package summary

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/glincker/stacklit/internal/insights"
	"github.com/glincker/stacklit/internal/schema"
)

func TestDefaultCommandAppendsSystemPrompt(t *testing.T) {
	command := summaryCommand(defaultCommandPrefix(), freshPrompt(&schema.Index{}, "", ""))

	if !slices.Equal(command[:3], []string{"claude", "-p", "--system-prompt"}) {
		t.Fatalf("expected claude print mode with system prompt flag, got %v", command)
	}
	if command[3] != freshPrompt(&schema.Index{}, "", "") {
		t.Fatal("expected default command to pass summary instructions as system prompt")
	}
}

func TestDefaultSummaryTimeoutIsFiveMinutes(t *testing.T) {
	if defaultTimeoutSec != 5*60 {
		t.Fatalf("expected default summary timeout to be 300 seconds, got %d", defaultTimeoutSec)
	}
}

func TestSummaryCommandAppendsPromptToCustomPrefix(t *testing.T) {
	command := summaryCommand([]string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}, freshPrompt(&schema.Index{}, "", ""))

	if !slices.Equal(command[:3], []string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}) {
		t.Fatalf("expected custom command prefix to be preserved, got %v", command)
	}
	if command[3] != freshPrompt(&schema.Index{}, "", "") {
		t.Fatal("expected summary instructions to be appended after custom command prefix")
	}
}

func TestCodexExecInvocationUsesStdinForPromptAndInput(t *testing.T) {
	prefix := []string{"/usr/local/bin/codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}
	command, input := summaryInvocation(prefix, "Prompt instructions", `{"project":"stacklit"}`)

	if !slices.Equal(command, prefix) {
		t.Fatalf("expected codex command without prompt argument, got %v", command)
	}
	if !strings.Contains(input, "Prompt instructions") || !strings.Contains(input, `{"project":"stacklit"}`) {
		t.Fatalf("expected prompt and JSON on stdin, got %q", input)
	}
}

func TestSummaryWordTargetUsesIndexComplexityWithinBounds(t *testing.T) {
	small := &schema.Index{Modules: map[string]schema.ModuleInfo{"cmd": {}}}
	large := &schema.Index{
		Project:      schema.Project{Workspaces: []string{"a", "b"}},
		Tech:         schema.Tech{Frameworks: []string{"one", "two"}},
		Structure:    schema.Structure{Entrypoints: []string{"a", "b"}},
		Modules:      map[string]schema.ModuleInfo{},
		Dependencies: schema.Dependencies{Edges: make([][2]string, 100)},
	}
	for i := range 100 {
		large.Modules[fmt.Sprintf("module-%d", i)] = schema.ModuleInfo{}
	}

	if got := summaryWordTarget(small); got != minSummaryWords {
		t.Fatalf("expected small repository target %d, got %d", minSummaryWords, got)
	}
	if got := summaryWordTarget(large); got != maxSummaryWords {
		t.Fatalf("expected large repository target %d, got %d", maxSummaryWords, got)
	}
}

func TestReconcileRequestIncludesBothSummariesAndCurrentIndex(t *testing.T) {
	idx := &schema.Index{Modules: map[string]schema.ModuleInfo{"internal/summary": {}}}

	request := newReconcileRequest(idx, "Existing summary", "Fresh summary")

	if request.ExistingSummary != "Existing summary" || request.FreshSummary != "Fresh summary" {
		t.Fatalf("expected both summaries in reconciliation request, got %+v", request)
	}
	if _, ok := request.Index.Modules["internal/summary"]; !ok {
		t.Fatalf("expected current index in reconciliation request, got %+v", request.Index)
	}
	if request.TargetWordCount != minSummaryWords {
		t.Fatalf("expected target %d, got %d", minSummaryWords, request.TargetWordCount)
	}
}

func TestReconcilePromptIncludesResolvedWordBudget(t *testing.T) {
	idx := &schema.Index{Modules: map[string]schema.ModuleInfo{"internal/summary": {}}}

	prompt := reconcilePrompt(idx, "", "")
	if !strings.Contains(prompt, fmt.Sprintf("%d words", summaryWordTarget(idx))) {
		t.Fatalf("expected resolved word budget in reconciliation prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "return the existing summary unchanged") {
		t.Fatalf("expected reconciliation fallback in prompt, got %q", prompt)
	}
}

func TestSummaryPromptsRequestFirstContactOrientation(t *testing.T) {
	for name, prompt := range map[string]string{
		"fresh":     freshPrompt(&schema.Index{}, "", ""),
		"reconcile": reconcilePrompt(&schema.Index{}, "", ""),
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range []string{"first-time reader", "end-to-end flow", "responsibility boundaries", "invariants or constraints", "where new behavior belongs"} {
				if !strings.Contains(prompt, expected) {
					t.Fatalf("expected prompt to request %q, got %q", expected, prompt)
				}
			}
		})
	}
}

func TestRunReconcilesExistingSummary(t *testing.T) {
	script := filepath.Join(t.TempDir(), "summary")
	if err := os.WriteFile(script, []byte(`#!/bin/sh
case "$*" in
  *"reconciling an existing"*) echo '{"architecture":{"ai_summary":"Reconciled summary."}}' ;;
  *)
    input="$(cat)"
    case "$input" in *"target_word_count"*) exit 1 ;; esac
    echo '{"purpose":{"internal/summary":"AI summary generation"},"architecture":{"ai_summary":"Fresh summary."}}'
    ;;
esac
`), 0755); err != nil {
		t.Fatalf("writing summary command: %v", err)
	}
	t.Setenv(envCmd, script)

	got, err := Run(&schema.Index{Modules: map[string]schema.ModuleInfo{"internal/summary": {}}}, &insights.File{Architecture: schema.Architecture{Summary: "Existing summary."}}, t.TempDir())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got.Architecture.Summary != "Reconciled summary." {
		t.Fatalf("expected reconciled summary, got %q", got.Architecture.Summary)
	}
}

func TestExistingPurposeContextIsSortedAndPromptOnly(t *testing.T) {
	purposes := existingPurposeContext(&insights.File{Purpose: map[string]string{"b": "Second", "a": "First"}})
	if !strings.HasPrefix(purposes, "a: First\nb: Second\n") {
		t.Fatalf("expected sorted purpose context, got %q", purposes)
	}
	if prompt := freshPrompt(&schema.Index{}, "", purposes); !strings.Contains(prompt, "do not output or enumerate them") {
		t.Fatalf("expected background-only instruction, got %q", prompt)
	}
}

func TestParseInsightsOutput(t *testing.T) {
	got, err := parseInsightsOutput(`{
  "purpose": {
    "internal/cli": "CLI commands and wiring"
  },
  "hints": {
    "add_feature": "Add commands in internal/cli",
    "test_command": "go test ./..."
  },
  "architecture": {
    "ai_summary": "A focused CLI around an indexing pipeline."
  }
}`)
	if err != nil {
		t.Fatalf("parseInsightsOutput returned error: %v", err)
	}

	if got.Purpose["internal/cli"] != "CLI commands and wiring" {
		t.Fatalf("expected generated purpose, got %+v", got.Purpose)
	}
	if got.Hints.TestCmd != "go test ./..." {
		t.Fatalf("expected generated test hint, got %+v", got.Hints)
	}
	if got.Architecture.Summary != "A focused CLI around an indexing pipeline." {
		t.Fatalf("expected generated architecture summary, got %+v", got.Architecture)
	}
}

func TestParseInsightsOutputExtractsWrappedJSON(t *testing.T) {
	got, err := parseInsightsOutput("Generated insights:\n```json\n{\n  \"purpose\": {\n    \"internal/summary\": \"AI insight generation with {braces} in prose\"\n  }\n}\n```\nDone.")
	if err != nil {
		t.Fatalf("parseInsightsOutput returned error: %v", err)
	}

	if got.Purpose["internal/summary"] != "AI insight generation with {braces} in prose" {
		t.Fatalf("expected generated purpose from wrapped JSON, got %+v", got.Purpose)
	}
}

func TestParseInsightsOutputRejectsProse(t *testing.T) {
	_, err := parseInsightsOutput("This repository is a CLI around an indexing pipeline.")
	if err == nil {
		t.Fatal("expected prose output to be rejected")
	}
	if !strings.Contains(err.Error(), "expected stacklit-insights JSON") {
		t.Fatalf("expected JSON error, got %v", err)
	}
}

func TestParseInsightsOutputRejectsEmptyObject(t *testing.T) {
	_, err := parseInsightsOutput(`{}`)
	if err == nil {
		t.Fatal("expected empty JSON object to be rejected")
	}
	if !strings.Contains(err.Error(), "expected at least one generated insight") {
		t.Fatalf("expected empty insight error, got %v", err)
	}
}
