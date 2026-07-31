package globset

import (
	"regexp"
	"testing"
)

func TestGlobSetMatch(t *testing.T) {
	tests := []struct {
		patterns  []string
		path      string
		matched   bool
		isIgnored bool
	}{
		// Simple suffix match
		{[]string{"*.log"}, "error.log", true, true},
		{[]string{"*.log"}, "dir/error.log", true, true},
		{[]string{"*.log"}, "dir/subdir/error.log", true, true},
		{[]string{"*.log"}, "error.txt", false, false},

		// Slash pattern (anchored to root)
		{[]string{"/bin/"}, "bin/exe", true, true},
		{[]string{"bin/"}, "dir/bin/exe", true, true}, // trailing slash matches directories or nested directories

		// Double star pattern
		{[]string{"src/**/*.go"}, "src/main.go", true, true},
		{[]string{"src/**/*.go"}, "src/pkg/helper.go", true, true},
		{[]string{"src/**/*.go"}, "pkg/helper.go", false, false},

		// Negated patterns (override ignore)
		{[]string{"*.go", "!main.go"}, "main.go", true, false},
		{[]string{"*.go", "!main.go"}, "helper.go", true, true},
		{[]string{"!main.go", "*.go"}, "main.go", true, true}, // order matters: last match wins
	}

	for i, tc := range tests {
		gs, err := NewGlobSet(tc.patterns)
		if err != nil {
			t.Fatalf("test case %d: unexpected error: %v", i, err)
		}
		matched, isIgnored := gs.MatchPath(tc.path)
		if matched != tc.matched || isIgnored != tc.isIgnored {
			t.Errorf("test case %d: patterns %v on path %q: expected (matched=%v, ignored=%v), got (matched=%v, ignored=%v)",
				i, tc.patterns, tc.path, tc.matched, tc.isIgnored, matched, isIgnored)
		}
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "single character", pattern: "file?.go", path: "dir/file1.go", want: true},
		{name: "negative class", pattern: "[!a].txt", path: "b.txt", want: true},
		{name: "regex metacharacters", pattern: "a+(b).txt", path: "a+(b).txt", want: true},
		{name: "anchored path", pattern: "/src/*.go", path: "src/main.go", want: true},
	}
	regexPattern, err := GlobToRegex(`file\*.txt`)
	if err != nil {
		t.Fatalf("GlobToRegex() error = %v", err)
	}
	if !regexp.MustCompile(regexPattern).MatchString("file*.txt") {
		t.Fatalf("GlobToRegex() = %q, want escaped star match", regexPattern)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			glob, err := NewGlob(test.pattern)
			if err != nil {
				t.Fatalf("NewGlob() error = %v", err)
			}
			if got := glob.Match(test.path); got != test.want {
				t.Fatalf("Match() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestMatchGlobFilter(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		wantFile bool
		wantDir  bool
	}{
		{name: "empty set", path: "file.txt"},
		{name: "positive match", patterns: []string{"*.go"}, path: "src/main.go"},
		{name: "positive miss", patterns: []string{"*.go"}, path: "src/main.txt", wantFile: true},
		{name: "negative match", patterns: []string{"!vendor/**"}, path: "vendor/pkg/file.go", wantFile: true, wantDir: true},
		{name: "negative directory", patterns: []string{"!vendor/**"}, path: "vendor/pkg", wantFile: true, wantDir: true},
		{name: "comments ignored", patterns: []string{"", "# comment"}, path: "file.txt"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set, err := NewGlobSet(test.patterns)
			if err != nil {
				t.Fatalf("NewGlobSet() error = %v", err)
			}
			if got := set.MatchGlobFilter(test.path); got != test.wantFile {
				t.Fatalf("MatchGlobFilter() = %t, want %t", got, test.wantFile)
			}
			if got := set.MatchGlobFilterDir(test.path); got != test.wantDir {
				t.Fatalf("MatchGlobFilterDir() = %t, want %t", got, test.wantDir)
			}
		})
	}
}
