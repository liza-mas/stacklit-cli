package summary

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxDocumentationFiles = 12
	maxDocumentationBytes = 16 * 1024
	maxDigestBytes        = 192 * 1024
)

type documentationCandidate struct {
	path string
	size int64
	rank int
}

// documentationDigest discovers and ranks documentation before reading bounded excerpts.
func documentationDigest(root string) (string, error) {
	candidates, err := documentationCandidates(root)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	used := 0
	for i, candidate := range candidates {
		if i >= maxDocumentationFiles || used >= maxDigestBytes {
			break
		}
		data, err := os.ReadFile(filepath.Join(root, candidate.path))
		if err != nil {
			continue
		}
		header := fmt.Sprintf("\n--- %s ---\n", candidate.path)
		available := maxDigestBytes - used - len(header) - 1
		if available <= 0 {
			break
		}
		limit := min(len(data), maxDocumentationBytes, available)
		content := string(data[:limit])
		if limit < len(data) {
			if newline := strings.LastIndex(content, "\n"); newline > 0 {
				content = content[:newline]
			}
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		out.WriteString(header)
		out.WriteString(content)
		out.WriteByte('\n')
		used += len(header) + len(content) + 1
	}
	return out.String(), nil
}

func documentationCandidates(root string) ([]documentationCandidate, error) {
	var candidates []documentationCandidate
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skippedDocumentationDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".mdx" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		candidates = append(candidates, documentationCandidate{path: filepath.ToSlash(rel), size: info.Size(), rank: documentationRank(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		return candidates[i].path < candidates[j].path
	})
	return candidates, nil
}

func skippedDocumentationDir(name string) bool {
	switch name {
	case ".git", ".stacklit", "node_modules", "vendor", "build", "dist", "target":
		return true
	default:
		return false
	}
}

func documentationRank(path string) int {
	path = strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(path)
	parts := strings.Split(path, "/")
	switch {
	case hasDocumentationSegment(parts, "adr", "adrs") || strings.HasPrefix(base, "adr"):
		return 0
	case hasDocumentationSegment(parts, "invariant", "invariants", "constraint", "constraints", "guardrail", "guardrails", "contract", "contracts") || hasDocumentationStem(base, "invariant", "invariants", "constraint", "constraints", "guardrail", "guardrails", "contract", "contracts"):
		return 1
	case strings.Contains(base, "vision") || strings.Contains(base, "mission") || strings.Contains(base, "strategy"):
		return 2
	case strings.HasPrefix(base, "readme"):
		return 3
	case strings.Contains(base, "architecture") || strings.Contains(base, "design"):
		return 4
	case strings.Contains(base, "prd") || strings.Contains(base, "spec"):
		return 5
	default:
		return 6
	}
}

func hasDocumentationSegment(parts []string, names ...string) bool {
	for _, part := range parts {
		for _, name := range names {
			if part == name {
				return true
			}
		}
	}
	return false
}

func hasDocumentationStem(path string, names ...string) bool {
	stem := strings.TrimSuffix(path, filepath.Ext(path))
	for _, name := range names {
		if stem == name {
			return true
		}
	}
	return false
}
