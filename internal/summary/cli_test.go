package summary

import (
	"strings"
	"testing"
)

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
