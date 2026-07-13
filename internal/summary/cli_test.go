package summary

import (
	"slices"
	"strings"
	"testing"
)

func TestDefaultCommandAppendsSystemPrompt(t *testing.T) {
	command := summaryCommand(defaultCommandPrefix())

	if !slices.Equal(command[:3], []string{"claude", "-p", "--system-prompt"}) {
		t.Fatalf("expected claude print mode with system prompt flag, got %v", command)
	}
	if command[3] != systemPrompt {
		t.Fatal("expected default command to pass summary instructions as system prompt")
	}
}

func TestDefaultSummaryTimeoutIsFiveMinutes(t *testing.T) {
	if defaultTimeoutSec != 5*60 {
		t.Fatalf("expected default summary timeout to be 300 seconds, got %d", defaultTimeoutSec)
	}
}

func TestSummaryCommandAppendsPromptToCustomPrefix(t *testing.T) {
	command := summaryCommand([]string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"})

	if !slices.Equal(command[:3], []string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}) {
		t.Fatalf("expected custom command prefix to be preserved, got %v", command)
	}
	if command[3] != systemPrompt {
		t.Fatal("expected summary instructions to be appended after custom command prefix")
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
