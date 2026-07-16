package summary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentationDigestRanksAndBoundsExcerpts(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"README.md":                     "README evidence",
		"adr/001.md":                    "Root ADR evidence",
		"docs/vision.md":                "Vision evidence",
		"specs/architecture/ADR/001.md": "ADR evidence",
		"INVARIANTS.md":                 "Invariant evidence",
		"docs/large.md":                 strings.Repeat("x", maxDocumentationBytes+100),
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	digest, err := documentationDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(digest, "ADR evidence") || !strings.Contains(digest, "Root ADR evidence") || !strings.Contains(digest, "Invariant evidence") || !strings.Contains(digest, "Vision evidence") || !strings.Contains(digest, "README evidence") {
		t.Fatalf("expected ranked documentation, got %q", digest)
	}
	if strings.Index(digest, "ADR evidence") > strings.Index(digest, "Invariant evidence") || strings.Index(digest, "Invariant evidence") > strings.Index(digest, "Vision evidence") {
		t.Fatalf("expected ADR before invariants before vision, got %q", digest)
	}
	if len(digest) > maxDocumentationBytes*2 {
		t.Fatalf("expected per-document bound, got %d bytes", len(digest))
	}
}

func TestDocumentationRankAvoidsIncidentalContractTerms(t *testing.T) {
	if documentationRank("adr/001.md") != 0 {
		t.Fatalf("expected root ADR directory to rank first")
	}
	if documentationRank("contractor-notes.md") <= documentationRank("INVARIANTS.md") {
		t.Fatal("expected incidental contractor name to rank below invariant documentation")
	}
}
