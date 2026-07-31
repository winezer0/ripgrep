package searcher

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/winezer0/gogrep/pkg/matcher"
	"github.com/winezer0/gogrep/pkg/printer"
)

func TestSearchReader(t *testing.T) {
	tests := []struct {
		name    string
		content string
		config  Config
		want    string
	}{
		{name: "basic PCRE", content: "no\nuser:alice\n", want: "2|user:alice|0:10|2|1"},
		{name: "before and after context", content: "before\nuser:alice\nafter\nlast\n", config: Config{BeforeContext: 1, AfterContext: 1}, want: "1|before|context;2|user:alice|0:10;3|after|context|4|1"},
		{name: "invert match", content: "user:alice\nplain\n", config: Config{InvertMatch: true}, want: "2|plain|2|1"},
		{name: "replace named capture", content: "user:alice\n", config: Config{HasReplace: true, Replace: "${name}!"}, want: "1|alice!|0:6|1|1"},
		{name: "max count", content: "user:a\nuser:b\n", config: Config{MaxCount: 1}, want: "1|user:a|0:6|1|1"},
		{name: "binary input", content: "user:\x00alice\n", want: "0|0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := buildMatcher(t)
			test.config.Matcher = m
			result, err := New(test.config).SearchReader(bytes.NewBufferString(test.content), "input")
			if err != nil {
				t.Fatalf("SearchReader() error = %v", err)
			}
			if got := resultString(result); got != test.want {
				t.Fatalf("SearchReader() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSearchReaderCancellationAndMatcherError(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*Searcher)
		reader string
		want   error
	}{
		{name: "cancelled", setup: cancelSearcher, reader: "text", want: context.Canceled},
		{name: "matcher failure", setup: func(*Searcher) {}, reader: "text", want: errMatcher},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			searcher := New(Config{Matcher: failingMatcher{}})
			if test.name == "cancelled" {
				searcher = New(Config{Matcher: buildMatcher(t)})
			}
			test.setup(searcher)
			_, err := searcher.SearchReader(bytes.NewBufferString(test.reader), "input")
			if !errors.Is(err, test.want) {
				t.Fatalf("SearchReader() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestSearchFile(t *testing.T) {
	tests := []struct {
		name   string
		ext    string
		create func(*testing.T, string) string
		want   string
	}{
		{name: "plain", ext: ".txt", create: createPlain, want: "1|user:alice|0:10|1|1"},
		{name: "gzip", ext: ".gz", create: createGzip, want: "1|user:alice|0:10|1|1"},
		{name: "zip", ext: ".zip", create: createZip, want: "1|user:alice|0:10|1|1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.create(t, filepath.Join(t.TempDir(), "input"+test.ext))
			searcher := New(Config{Matcher: buildMatcher(t), SearchZip: true})
			results, err := searcher.SearchFile(path)
			if err != nil {
				t.Fatalf("SearchFile() error = %v", err)
			}
			if len(results) != 1 || resultString(results[0]) != test.want {
				t.Fatalf("SearchFile() = %#v, want %q", results, test.want)
			}
		})
	}
}

func TestSearchFileErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "missing file", path: filepath.Join(t.TempDir(), "missing.txt")},
		{name: "invalid gzip", path: createPlain(t, filepath.Join(t.TempDir(), "invalid.gz"))},
		{name: "invalid zip", path: createPlain(t, filepath.Join(t.TempDir(), "invalid.zip"))},
		{name: "invalid bzip2", path: createPlain(t, filepath.Join(t.TempDir(), "invalid.bz2"))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			searcher := New(Config{Matcher: buildMatcher(t), SearchZip: true})
			if _, err := searcher.SearchFile(test.path); err == nil {
				t.Fatal("SearchFile() error = nil, want error")
			}
		})
	}
}

var errMatcher = errors.New("matcher failed")

type failingMatcher struct{}

func (failingMatcher) Match([]byte) (bool, error)               { return false, errMatcher }
func (failingMatcher) FindSpans([]byte) ([]matcher.Span, error) { return nil, errMatcher }
func (failingMatcher) Replace([]byte, string) ([]byte, []matcher.Span, error) {
	return nil, nil, errMatcher
}
func (failingMatcher) Close() error { return nil }

func buildMatcher(t *testing.T) matcher.Matcher {
	t.Helper()
	m, err := matcher.Build(`user:(?<name>\w+)`, matcher.Options{})
	if err != nil {
		t.Fatalf("matcher.Build() error = %v", err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return m
}

func cancelSearcher(searcher *Searcher) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	searcher.SetContext(ctx)
}

func resultString(result *printer.FileResult) string {
	items := make([]string, 0, len(result.Matches))
	for _, match := range result.Matches {
		item := fmt.Sprintf("%d|%s", match.LineNum, match.Line)
		if match.IsContext {
			item += "|context"
		}
		for _, submatch := range match.Submatches {
			item += fmt.Sprintf("|%d:%d", submatch.Start, submatch.End)
		}
		items = append(items, item)
	}
	stats := fmt.Sprintf("%d|%d", result.Stats.SearchedLines, result.Stats.Matches)
	if len(items) == 0 {
		return stats
	}
	return strings.Join(items, ";") + "|" + stats
}

func createPlain(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("user:alice\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func createGzip(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte("user:alice\n")); err != nil {
		t.Fatalf("gzip Write() error = %v", err)
	}
	if err := errors.Join(writer.Close(), file.Close()); err != nil {
		t.Fatalf("gzip Close() error = %v", err)
	}
	return path
}

func createZip(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("inner.txt")
	if err != nil {
		t.Fatalf("zip Create() error = %v", err)
	}
	if _, err := entry.Write([]byte("user:alice\n")); err != nil {
		t.Fatalf("zip Write() error = %v", err)
	}
	if err := errors.Join(writer.Close(), file.Close()); err != nil {
		t.Fatalf("zip Close() error = %v", err)
	}
	return path
}
