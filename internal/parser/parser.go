package parser

import (
	"bytes"
	"os"
	"strings"
	"sync"
)

// FileInfo holds the result of parsing a source file.
type FileInfo struct {
	Path         string            `json:"path"`
	Language     string            `json:"language"`
	Imports      []string          `json:"imports,omitempty"`
	Exports      []string          `json:"exports,omitempty"`
	TypeDefs     map[string]string `json:"type_defs,omitempty"` // type name -> brief definition
	LineCount    int               `json:"line_count"`
	IsEntrypoint bool              `json:"is_entrypoint,omitempty"`
}

// Parser extracts structured information from a source file.
type Parser interface {
	CanParse(filename string) bool
	Parse(path string, content []byte) (*FileInfo, error)
}

// registry holds all registered parsers in priority order.
// Generic must be last so it acts as a fallback.
var registry []Parser

func init() {
	registry = []Parser{
		&GoParser{},
		&TreeSitterParser{},
		&GenericParser{},
	}
}

// ParseFile reads a file from disk and dispatches to the appropriate parser.
func ParseFile(path string) (*FileInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	for _, p := range registry {
		if p.CanParse(path) {
			return p.Parse(path, content)
		}
	}

	// Should never reach here because GenericParser.CanParse always returns true.
	g := &GenericParser{}
	return g.Parse(path, content)
}

type parseFunc func(path string) (*FileInfo, error)

type parseJob struct {
	index int
	path  string
}

type parseResult struct {
	index int
	info  *FileInfo
	err   error
}

// ParseAll parses multiple files with one worker and returns all results.
// Errors from individual files are collected and non-fatal.
func ParseAll(paths []string) ([]*FileInfo, []error) {
	return ParseAllWithWorkers(paths, 1)
}

// ParseAllWithWorkers parses multiple files with up to workerCount concurrent parse calls.
// Callers are expected to pass a positive worker count; non-positive values fall back to one worker.
func ParseAllWithWorkers(paths []string, workerCount int) ([]*FileInfo, []error) {
	return parseAllWithWorkers(paths, workerCount, ParseFile)
}

func parseAllWithWorkers(paths []string, workerCount int, parse parseFunc) ([]*FileInfo, []error) {
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(paths) {
		workerCount = len(paths)
	}

	orderedResults := make([]parseResult, len(paths))
	if len(paths) == 0 {
		return make([]*FileInfo, 0), nil
	}

	jobs := make(chan parseJob, len(paths))
	completions := make(chan parseResult, len(paths))
	for index, path := range paths {
		jobs <- parseJob{index: index, path: path}
	}
	close(jobs)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				info, err := parse(job.path)
				completions <- parseResult{index: job.index, info: info, err: err}
			}
		}()
	}
	wg.Wait()
	close(completions)

	for completion := range completions {
		orderedResults[completion.index] = completion
	}

	results := make([]*FileInfo, 0, len(paths))
	var errs []error
	for _, result := range orderedResults {
		if result.err != nil {
			errs = append(errs, result.err)
			continue
		}
		if result.info != nil {
			results = append(results, result.info)
		}
	}
	return results, errs
}

// countLines counts the number of lines in content.
func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := bytes.Count(content, []byte{'\n'})
	// If the file does not end with a newline, add one for the last line.
	if content[len(content)-1] != '\n' {
		count++
	}
	return count
}

// extIs reports whether filename has one of the given extensions (case-insensitive).
func extIs(filename string, exts ...string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
