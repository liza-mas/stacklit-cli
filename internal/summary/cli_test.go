package summary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/glincker/stacklit/internal/schema"
)

func TestDefaultCommandAppendsSystemPrompt(t *testing.T) {
	command := summaryCommand(defaultCommandPrefix(), freshPrompt(&schema.Index{}, "", true))

	if !slices.Equal(command[:3], []string{"claude", "-p", "--system-prompt"}) {
		t.Fatalf("expected claude print mode with system prompt flag, got %v", command)
	}
	if command[3] != freshPrompt(&schema.Index{}, "", true) {
		t.Fatal("expected default command to pass summary instructions as system prompt")
	}
}

func TestDefaultSummaryTimeoutIsFiveMinutes(t *testing.T) {
	if defaultTimeoutSec != 5*60 {
		t.Fatalf("expected default summary timeout to be 300 seconds, got %d", defaultTimeoutSec)
	}
}

func TestSummaryCommandAppendsPromptToCustomPrefix(t *testing.T) {
	command := summaryCommand([]string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}, freshPrompt(&schema.Index{}, "", true))

	if !slices.Equal(command[:3], []string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}) {
		t.Fatalf("expected custom command prefix to be preserved, got %v", command)
	}
	if command[3] != freshPrompt(&schema.Index{}, "", true) {
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

func TestSnapshotForExcludesInsightsWithoutMutatingIndex(t *testing.T) {
	idx := &schema.Index{
		Project:      schema.Project{Name: "project"},
		Modules:      map[string]schema.ModuleInfo{"core": {Purpose: "OLD_PURPOSE", Files: 2, Exports: []string{"Run"}}},
		Hints:        schema.Hints{AddFeature: "OLD_HINT", TestCmd: "OLD_TEST_COMMAND"},
		Architecture: schema.Architecture{Summary: "OLD_SUMMARY"},
	}
	before, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := snapshotFor(idx)
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"OLD_PURPOSE", "OLD_HINT", "OLD_TEST_COMMAND", "OLD_SUMMARY", `"hints"`, `"architecture"`} {
		if strings.Contains(string(payload), unwanted) {
			t.Fatalf("initial snapshot exposes previous insights: %s", payload)
		}
	}
	if snapshot.Project.Name != "project" || snapshot.Modules["core"].Files != 2 || !slices.Equal(snapshot.Modules["core"].Exports, []string{"Run"}) {
		t.Fatalf("snapshot lost structural evidence: %+v", snapshot)
	}
	after, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("snapshot sanitization mutated the index used for merging")
	}
}

func TestSessionPromptOrdersDraftReadAndReconciliation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "custom insights.json")
	prompt := sessionPrompt(&schema.Index{}, "", root, path)
	previous := -1
	for _, step := range []string{"1. Draft fresh", "2. Only after drafting", "3. Reconcile", "4. Return only the final insights JSON"} {
		position := strings.Index(prompt, step)
		if position <= previous {
			t.Fatalf("workflow must order drafting, deferred reading, reconciliation, and final output: missing or misplaced %q", step)
		}
		previous = position
	}
	if !strings.Contains(prompt, fmt.Sprintf("%q", path)) || !strings.Contains(prompt, fmt.Sprintf("%q", root)) {
		t.Fatal("workflow must identify the actual repository and previous insights file")
	}
	if !strings.Contains(prompt, "If it is absent, keep the fresh draft") || !strings.Contains(prompt, "Do not emit intermediate drafts or edit repository files") {
		t.Fatal("workflow must handle absent insights and leave persistence to Stacklit")
	}
}

func TestSummaryPromptsRequestSupportedDynamicsWithoutStructuralRepetition(t *testing.T) {
	for name, prompt := range map[string]string{
		"fresh":             freshPrompt(&schema.Index{}, "", true),
		"session":           sessionPrompt(&schema.Index{}, "", "/repo", "/repo/insights.json"),
		"fresh with docs":   freshPrompt(&schema.Index{}, "\n--- design-notes.md ---\nReference evidence.", true),
		"session with docs": sessionPrompt(&schema.Index{}, "\n--- design-notes.md ---\nReference evidence.", "/repo", "/repo/insights.json"),
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range []string{
				"\nSCOPE:\n", "\nCONTENT:\n", "\nEVIDENCE RULES:\n", "\nSTYLE:\n",
				"2-5 concise paragraphs", "first-time reader",
				"Ground behavioral claims in available evidence",
				"Within architecture.ai_summary, do not repeat module inventories or placement hints",
				"Connect responsibilities where needed",
				"Establish the main end-to-end flow before specialized details",
				"State project-level constraints that bound acceptable implementations",
				"and stacklit derive as the reader's next step",
				fmt.Sprintf("Use a soft word budget of %d words (+/-10%%)", summaryWordTarget(&schema.Index{})),
			} {
				if !strings.Contains(prompt, expected) {
					t.Fatalf("expected prompt to request %q, got %q", expected, prompt)
				}
			}
			for _, obsolete := range []string{"where new behavior belongs", "Focus on purpose"} {
				if strings.Contains(prompt, obsolete) {
					t.Fatalf("obsolete instruction %q conflicts with the summary contract", obsolete)
				}
			}
		})
	}
}

func TestSummaryPromptInspectionAndDocumentation(t *testing.T) {
	const docs = "\n--- docs/decisions/README.md ---\nDecision index evidence."
	for _, canInspect := range []bool{false, true} {
		for _, suppliedDocs := range []string{"", docs} {
			t.Run(fmt.Sprintf("inspect=%t/docs=%t", canInspect, suppliedDocs != ""), func(t *testing.T) {
				prompt := freshPrompt(&schema.Index{}, suppliedDocs, canInspect)
				if strings.Contains(prompt, "or verified repository inspection") != canInspect ||
					strings.Contains(prompt, "do not request repository inspection") == canInspect {
					t.Fatal("inspection guidance must follow invocation capability, not document presence")
				}
				if strings.Contains(prompt, "\nDOCUMENTATION:\n") != (suppliedDocs != "") {
					t.Fatal("documentation instructions must follow supplied document presence")
				}
				if suppliedDocs != "" {
					if !strings.HasSuffix(prompt, docs+"\n--- end documentation excerpts ---\n") {
						t.Fatal("documentation excerpts must have an explicit closing marker")
					}
					if !strings.Contains(prompt, "Review and cite README.md when present") ||
						!strings.Contains(prompt, "Link the decision index when one is identified, otherwise the ADR directory") {
						t.Fatal("documentation routing must prefer README and a decision index")
					}
				}
			})
		}
	}
}

func TestRunUsesOneSessionWithoutExposingPreviousInsights(t *testing.T) {
	for _, hasExisting := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%t", hasExisting), func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "custom insights.json")
			existing := []byte(`{"purpose":{"core":"OLD_PURPOSE"},"hints":{"add_feature":"OLD_HINT"},"architecture":{"ai_summary":"OLD_SUMMARY"}}`)
			if hasExisting {
				if err := os.WriteFile(path, existing, 0644); err != nil {
					t.Fatal(err)
				}
			}
			promptPath := filepath.Join(root, "prompt.txt")
			inputPath := filepath.Join(root, "input.json")
			callsPath := filepath.Join(root, "calls.txt")
			t.Setenv("SUMMARY_TEST_PROMPT", promptPath)
			t.Setenv("SUMMARY_TEST_INPUT", inputPath)
			t.Setenv("SUMMARY_TEST_CALLS", callsPath)
			script := filepath.Join(root, "summary")
			if err := os.WriteFile(script, []byte(`#!/bin/sh
printf '%s' "$*" > "$SUMMARY_TEST_PROMPT"
cat > "$SUMMARY_TEST_INPUT"
printf 'call\n' >> "$SUMMARY_TEST_CALLS"
printf '%s\n' '{"purpose":{"core":"Final purpose"},"hints":{"add_feature":"Final hint"},"architecture":{"ai_summary":"Final summary."}}'
`), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv(envCmd, script)
			idx := &schema.Index{
				Modules:      map[string]schema.ModuleInfo{"core": {Purpose: "OLD_PURPOSE", Files: 1}},
				Hints:        schema.Hints{AddFeature: "OLD_HINT"},
				Architecture: schema.Architecture{Summary: "OLD_SUMMARY"},
			}
			got, err := Run(idx, path, root)
			if err != nil {
				t.Fatal(err)
			}
			if got.Purpose["core"] != "Final purpose" || got.Hints.AddFeature != "Final hint" || got.Architecture.Summary != "Final summary." {
				t.Fatalf("expected final insights from the single session, got %+v", got)
			}
			calls, err := os.ReadFile(callsPath)
			if err != nil || string(calls) != "call\n" {
				t.Fatalf("expected exactly one invocation, got %q (%v)", calls, err)
			}
			prompt, err := os.ReadFile(promptPath)
			if err != nil {
				t.Fatal(err)
			}
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, old := range []string{"OLD_PURPOSE", "OLD_HINT", "OLD_SUMMARY"} {
				if strings.Contains(string(prompt)+string(input), old) {
					t.Fatalf("previous insights leaked into initial agent input: %s", old)
				}
			}
			if !strings.Contains(string(prompt), fmt.Sprintf("%q", path)) ||
				!strings.Contains(string(prompt), "1. Draft fresh") ||
				!strings.Contains(string(prompt), "or verified repository inspection") {
				t.Fatal("single-session invocation omitted the deferred-read path, draft phase, or source access")
			}
			stored, err := os.ReadFile(path)
			if hasExisting {
				if err != nil || string(stored) != string(existing) {
					t.Fatal("Run modified the existing insights file before the caller's merge")
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("Run created the output file before the caller's merge: %v", err)
			}
		})
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
