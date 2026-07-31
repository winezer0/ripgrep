package gogrep

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	root := createSearchTree(t)
	tests := []struct {
		name    string
		paths   []string
		options Options
		want    string
	}{
		{
			name: "respects ignore and hidden rules", paths: []string{root},
			options: Options{Pattern: `(?<=user:)\w+`, Threads: 2, SortBy: "path"},
			want:    "visible.go=alice",
		},
		{
			name: "includes hidden and ignored files", paths: []string{root},
			options: Options{Pattern: `user:(\w+)`, Hidden: true, NoIgnore: true, SortBy: "path"},
			want:    ".hidden=user:hidden,skip.txt=user:skip,visible.go=user:alice",
		},
		{
			name: "glob and type filters", paths: []string{root},
			options: Options{Pattern: "user", NoIgnore: true, Hidden: true, Globs: []string{"*.go"}, Types: []string{"go"}},
			want:    "visible.go=user",
		},
		{
			name: "explicit file", paths: []string{filepath.Join(root, "skip.txt")},
			options: Options{Pattern: "skip", FixedStrings: true}, want: "skip.txt=skip",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results, err := Search(context.Background(), test.paths, test.options)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if got := summarizeResults(results); got != test.want {
				t.Fatalf("Search() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSearchStream(t *testing.T) {
	path := filepath.Join(createSearchTree(t), "visible.go")
	stream, err := SearchStream(nil, []string{path}, Options{Pattern: "alice"})
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}
	var results []FileResult
	for result := range stream.Results {
		results = append(results, result)
	}
	for streamErr := range stream.Errors {
		t.Fatalf("SearchStream() async error = %v", streamErr)
	}
	if got := summarizeResults(results); got != "visible.go=alice" {
		t.Fatalf("SearchStream() = %q, want %q", got, "visible.go=alice")
	}
}

func TestSearchErrors(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		options Options
	}{
		{name: "invalid PCRE", options: Options{Pattern: "("}},
		{name: "invalid glob", options: Options{Pattern: "x", Globs: []string{"["}}},
		{name: "missing path", paths: []string{filepath.Join(t.TempDir(), "missing")}, options: Options{Pattern: "x"}},
		{name: "invalid sort", options: Options{Pattern: "x", SortBy: "random"}},
		{name: "unknown type", options: Options{Pattern: "x", Types: []string{"unknown"}}},
		{name: "negative context", options: Options{Pattern: "x", BeforeContext: -1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Search(context.Background(), test.paths, test.options)
			if err == nil {
				t.Fatal("Search() error = nil, want error")
			}
		})
	}
}

func TestSearchCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Search(ctx, []string{createSearchTree(t)}, Options{Pattern: "user"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Search() error = %v, want %v", err, context.Canceled)
	}
}

func createSearchTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"visible.go":      "user:alice\nplain\n",
		"skip.txt":        "user:skip\n",
		".hidden":         "user:hidden\n",
		"nested/empty.go": "plain\n",
		".gitignore":      "skip.txt\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	return root
}

func summarizeResults(results []FileResult) string {
	items := make([]string, 0, len(results))
	for _, result := range results {
		for _, match := range result.Matches {
			text := match.Line
			if len(match.Submatches) > 0 {
				text = match.Submatches[0].Text
			}
			items = append(items, filepath.Base(result.Path)+"="+text)
		}
	}
	sort.Strings(items)
	return strings.Join(items, ",")
}
