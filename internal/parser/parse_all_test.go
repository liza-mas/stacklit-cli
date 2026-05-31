package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseAllUsesOneWorkerCompatibility(t *testing.T) {
	paths := writeParseAllFixtures(t)

	gotInfos, gotErrs := ParseAll(paths)
	wantInfos, wantErrs := parseAllWithWorkers(paths, 1, ParseFile)

	if len(gotErrs) != 0 || len(wantErrs) != 0 {
		t.Fatalf("ParseAll errors = %v, ParseAllWithWorkers errors = %v, want none", gotErrs, wantErrs)
	}

	if !reflect.DeepEqual(gotInfos, wantInfos) {
		t.Fatalf("ParseAll results = %#v, want one-worker worker-pool results %#v", gotInfos, wantInfos)
	}
}

func TestParseAllWithWorkersPreservesInputOrder(t *testing.T) {
	paths := []string{"slow-ok", "first-error", "fast-ok", "second-error"}
	delays := map[string]time.Duration{
		"slow-ok":      30 * time.Millisecond,
		"first-error":  10 * time.Millisecond,
		"fast-ok":      1 * time.Millisecond,
		"second-error": 5 * time.Millisecond,
	}

	infos, errs := parseAllWithWorkers(paths, 3, func(path string) (*FileInfo, error) {
		time.Sleep(delays[path])
		if path == "first-error" || path == "second-error" {
			return nil, fmt.Errorf("parse %s", path)
		}
		return &FileInfo{Path: path, Language: "test"}, nil
	})

	gotInfoPaths := fileInfoPaths(infos)
	wantInfoPaths := []string{"slow-ok", "fast-ok"}
	if !reflect.DeepEqual(gotInfoPaths, wantInfoPaths) {
		t.Fatalf("result paths = %v, want %v", gotInfoPaths, wantInfoPaths)
	}

	gotErrs := errorStrings(errs)
	wantErrs := []string{"parse first-error", "parse second-error"}
	if !reflect.DeepEqual(gotErrs, wantErrs) {
		t.Fatalf("errors = %v, want %v", gotErrs, wantErrs)
	}
}

func TestParseAllWithWorkersCollectsParseErrorsNonFatally(t *testing.T) {
	paths := []string{"ok-before", "bad-file", "ok-after"}

	infos, errs := parseAllWithWorkers(paths, 2, func(path string) (*FileInfo, error) {
		if path == "bad-file" {
			return nil, errors.New("bad-file failed")
		}
		return &FileInfo{Path: path, Language: "test"}, nil
	})

	gotInfoPaths := fileInfoPaths(infos)
	wantInfoPaths := []string{"ok-before", "ok-after"}
	if !reflect.DeepEqual(gotInfoPaths, wantInfoPaths) {
		t.Fatalf("result paths = %v, want %v", gotInfoPaths, wantInfoPaths)
	}

	gotErrs := errorStrings(errs)
	wantErrs := []string{"bad-file failed"}
	if !reflect.DeepEqual(gotErrs, wantErrs) {
		t.Fatalf("errors = %v, want %v", gotErrs, wantErrs)
	}
}

func TestParseAllWithWorkersBoundsConcurrentParseCalls(t *testing.T) {
	const workerCount int32 = 2

	var active int32
	var maxActive int32
	release := make(chan struct{})
	var releaseOnce sync.Once

	paths := []string{"one", "two", "three", "four"}
	infos, errs := parseAllWithWorkers(paths, int(workerCount), func(path string) (*FileInfo, error) {
		current := atomic.AddInt32(&active, 1)
		recordMaxActive(&maxActive, current)
		if current == workerCount {
			releaseOnce.Do(func() { close(release) })
		}
		defer atomic.AddInt32(&active, -1)

		select {
		case <-release:
		case <-time.After(2 * time.Second):
			return nil, errors.New("timed out waiting for worker overlap")
		}

		time.Sleep(10 * time.Millisecond)
		return &FileInfo{Path: path, Language: "test"}, nil
	})

	if len(errs) != 0 {
		t.Fatalf("errors = %v, want none", errs)
	}
	if got := atomic.LoadInt32(&maxActive); got != workerCount {
		t.Fatalf("max active parse calls = %d, want %d", got, workerCount)
	}
	if gotInfoPaths := fileInfoPaths(infos); !reflect.DeepEqual(gotInfoPaths, paths) {
		t.Fatalf("result paths = %v, want %v", gotInfoPaths, paths)
	}
}

func TestParseAllWithWorkersParsesRealFilesConcurrently(t *testing.T) {
	paths := writeParseAllFixtures(t)

	infos, errs := ParseAllWithWorkers(paths, 3)

	if len(errs) != 0 {
		t.Fatalf("errors = %v, want none", errs)
	}
	if gotInfoPaths := fileInfoPaths(infos); !reflect.DeepEqual(gotInfoPaths, paths) {
		t.Fatalf("result paths = %v, want %v", gotInfoPaths, paths)
	}
}

func writeParseAllFixtures(t *testing.T) []string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"one.go":   "package one\n\nfunc One() {}\n",
		"two.txt":  "hello\nworld\n",
		"three.go": "package three\n\nfunc Three() {}\n",
	}

	paths := make([]string, 0, len(files))
	for _, name := range []string{"one.go", "two.txt", "three.go"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(files[name]), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
		paths = append(paths, path)
	}
	return paths
}

func fileInfoPaths(infos []*FileInfo) []string {
	paths := make([]string, 0, len(infos))
	for _, info := range infos {
		paths = append(paths, info.Path)
	}
	return paths
}

func errorStrings(errs []error) []string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return messages
}

func recordMaxActive(maxActive *int32, current int32) {
	for {
		previous := atomic.LoadInt32(maxActive)
		if current <= previous {
			return
		}
		if atomic.CompareAndSwapInt32(maxActive, previous, current) {
			return
		}
	}
}
