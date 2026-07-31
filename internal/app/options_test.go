package app

import (
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "long flags", args: []string{"--ignore-case", "--glob=*.go", "--context", "2", "pattern", "path"}, want: "pattern|path|true|2|2|*.go"},
		{name: "short cluster", args: []string{"-in", "pattern"}, want: "pattern||true|0|0|"},
		{name: "attached short value", args: []string{"-C3", "pattern"}, want: "pattern||false|3|3|"},
		{name: "option terminator", args: []string{"--", "-pattern", "-path"}, want: "-pattern|-path|false|0|0|"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cli, err := parseArgs(test.args)
			if err != nil {
				t.Fatalf("parseArgs() error = %v", err)
			}
			path := ""
			if len(cli.Paths) > 0 {
				path = cli.Paths[0]
			}
			glob := ""
			if len(cli.Globs) > 0 {
				glob = cli.Globs[0]
			}
			got := cli.Pattern + "|" + path + "|" + boolString(cli.CaseInsensitive) + "|" + intString(cli.BeforeContext) + "|" + intString(cli.AfterContext) + "|" + glob
			if got != test.want {
				t.Fatalf("parseArgs() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseAllLongOptions(t *testing.T) {
	args := []string{
		"--ignore-case", "--case-sensitive", "--word-regexp", "--fixed-strings",
		"--invert-match", "--hidden", "--no-ignore", "--follow", "--json",
		"--heading", "--no-heading", "--line-number", "--no-line-number",
		"--with-filename", "--no-filename", "--column", "--only-matching",
		"--count", "--quiet", "--search-zip", "--replace", "value", "--glob", "*.go",
		"--type", "go", "--type-not", "text", "--color", "always", "--sort", "path",
		"--sortr", "size", "--after-context", "1", "--before-context", "2",
		"--context", "3", "--max-count", "4", "--threads", "5", "--max-depth", "6",
		"pattern", "path",
	}
	cli, err := parseArgs(args)
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	got := []string{
		cli.Pattern, cli.Paths[0], cli.Replace, cli.Globs[0], cli.Types[0], cli.TypesNot[0],
		cli.Color, cli.SortBy, boolString(cli.SortReverse), boolString(cli.WordRegexp),
		boolString(cli.FixedStrings), boolString(cli.InvertMatch), boolString(cli.Hidden),
		boolString(cli.NoIgnore), boolString(cli.FollowSymlinks), boolString(cli.JSON),
		boolString(cli.NoHeading), boolString(cli.NoLineNumber), boolString(cli.NoFilename),
		boolString(cli.Column), boolString(cli.OnlyMatching), boolString(cli.Count),
		boolString(cli.Quiet), boolString(cli.SearchZip),
	}
	want := "pattern|path|value|*.go|go|text|always|size|true|true|true|true|true|true|true|true|true|true|true|true|true|true|true|true"
	if strings.Join(got, "|") != want {
		t.Fatalf("parseArgs() = %q, want %q", strings.Join(got, "|"), want)
	}
	_ = cli.options()
	_ = cli.printerConfig()
}

func TestParseAllShortOptions(t *testing.T) {
	args := []string{
		"-iswFvLnNHIocqz", "-r", "value", "-g*.go", "-tgo", "-T", "text",
		"-A1", "-B", "2", "-C3", "-m4", "-j5", "pattern",
	}
	cli, err := parseArgs(args)
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if cli.Pattern != "pattern" || cli.Replace != "value" || cli.MaxCount != 4 || cli.Threads != 5 {
		t.Fatalf("parseArgs() = %#v", cli)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "boolean with value", args: []string{"--hidden=true"}},
		{name: "missing value", args: []string{"--glob"}},
		{name: "unknown long", args: []string{"--unknown", "value"}},
		{name: "unknown short", args: []string{"-x"}},
		{name: "negative number", args: []string{"--threads", "-1"}},
		{name: "invalid number", args: []string{"--max-count", "many"}},
		{name: "invalid color", args: []string{"--color", "sometimes"}},
		{name: "invalid sort", args: []string{"--sort", "random"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseArgs(test.args); err == nil {
				t.Fatal("parseArgs() error = nil, want error")
			}
		})
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	if value == 2 {
		return "2"
	}
	if value == 3 {
		return "3"
	}
	return "other"
}
