package matcher

import (
	"errors"
	"testing"

	pcre2 "github.com/VillanCh/go-pcre2-lite"
)

func TestBuildAndMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		opts    Options
		input   string
		want    bool
	}{
		{name: "lookbehind", pattern: `(?<=user:)\w+`, input: "user:alice", want: true},
		{name: "backreference", pattern: `(go)\1`, input: "gogo", want: true},
		{name: "fixed metacharacters", pattern: "a.b", opts: Options{Fixed: true}, input: "a.b", want: true},
		{name: "fixed does not expand", pattern: "a.b", opts: Options{Fixed: true}, input: "axb", want: false},
		{name: "case insensitive", pattern: "hello", opts: Options{CaseInsensitive: true}, input: "HELLO", want: true},
		{name: "whole word rejects suffix", pattern: "go", opts: Options{WordRegexp: true}, input: "gopher", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := buildTestMatcher(t, test.pattern, test.opts)
			got, err := m.Match([]byte(test.input))
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Match() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestBuildRejectsInvalidPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{name: "unclosed group", pattern: "("},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Build(test.pattern, Options{}); err == nil {
				t.Fatal("Build() error = nil, want compilation error")
			}
		})
	}
}

func TestFindSpans(t *testing.T) {
	m := buildTestMatcher(t, `\p{Han}+`, Options{})
	spans, err := m.FindSpans([]byte("a中文b汉"))
	if err != nil {
		t.Fatalf("FindSpans() error = %v", err)
	}
	want := []Span{{Start: 1, End: 7}, {Start: 8, End: 11}}
	if len(spans) != len(want) || spans[0] != want[0] || spans[1] != want[1] {
		t.Fatalf("FindSpans() = %#v, want %#v", spans, want)
	}
}

func TestReplace(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		replacement string
		wantLine    string
		wantSpans   []Span
	}{
		{name: "numbered groups", pattern: `(\w+)-(\w+)`, replacement: "$2/$1", wantLine: "bar/foo", wantSpans: []Span{{Start: 0, End: 7}}},
		{name: "named group", pattern: `(?<word>\w+)`, replacement: "${word}!", wantLine: "foo!", wantSpans: []Span{{Start: 0, End: 4}}},
		{name: "literal dollar", pattern: `foo`, replacement: "$$$&", wantLine: "$foo", wantSpans: []Span{{Start: 0, End: 4}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := buildTestMatcher(t, test.pattern, Options{})
			line, spans, err := m.Replace([]byte("foo-bar"[:len(test.wantLine)-len(test.wantLine)+3]), test.replacement)
			if test.name == "numbered groups" {
				line, spans, err = m.Replace([]byte("foo-bar"), test.replacement)
			}
			if err != nil {
				t.Fatalf("Replace() error = %v", err)
			}
			if string(line) != test.wantLine {
				t.Fatalf("Replace() line = %q, want %q", line, test.wantLine)
			}
			if len(spans) != 1 || spans[0] != test.wantSpans[0] {
				t.Fatalf("Replace() spans = %#v, want %#v", spans, test.wantSpans)
			}
		})
	}
}

func TestClosedMatcherReturnsError(t *testing.T) {
	m := buildTestMatcher(t, "x", Options{})
	if err := m.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	_, err := m.Match([]byte("x"))
	if !errors.Is(err, pcre2.ErrClosed) {
		t.Fatalf("Match() error = %v, want %v", err, pcre2.ErrClosed)
	}
}

func buildTestMatcher(t *testing.T, pattern string, opts Options) Matcher {
	t.Helper()
	m, err := Build(pattern, opts)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return m
}
