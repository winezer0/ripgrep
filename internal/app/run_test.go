package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithStdin(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		input      string
		wantCode   int
		wantOutput string
	}{
		{name: "PCRE lookbehind", args: []string{`(?<=user:)\w+`, "-"}, input: "user:alice\n", wantCode: 0, wantOutput: "-:1:user:alice\n"},
		{name: "no match", args: []string{"missing", "-"}, input: "user:alice\n", wantCode: 1},
		{name: "only matching", args: []string{"-o", `(?<=user:)\w+`, "-"}, input: "user:alice\n", wantCode: 0, wantOutput: "-:1:alice\n"},
		{name: "count", args: []string{"-c", "user", "-"}, input: "user:a\nuser:b\n", wantCode: 0, wantOutput: "-:2\n"},
		{name: "quiet", args: []string{"-q", "user", "-"}, input: "user:a\n", wantCode: 0},
		{name: "invalid pattern", args: []string{"(", "-"}, input: "text", wantCode: 2},
		{name: "JSON summary", args: []string{"--json", "user", "-"}, input: "user:a\n", wantCode: 0, wantOutput: "{\"type\":\"begin\",\"data\":{\"path\":{\"text\":\"-\"}}}\n{\"type\":\"match\",\"data\":{\"path\":{\"text\":\"-\"},\"lines\":{\"text\":\"user:a\"},\"line_number\":1,\"submatches\":[{\"match\":{\"text\":\"user\"},\"start\":0,\"end\":4}]}}\n{\"type\":\"end\",\"data\":{\"path\":{\"text\":\"-\"},\"binary\":false,\"stats\":{\"searched_lines\":1,\"matches\":1}}}\n{\"type\":\"summary\",\"data\":{\"stats\":{\"elapsed\":{\"secs\":0,\"nanos\":0},\"searched_lines\":1,\"matches\":1}}}\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, strings.NewReader(test.input), &stdout, &stderr)
			if code != test.wantCode {
				t.Fatalf("Run() code = %d, want %d; stderr=%q", code, test.wantCode, stderr.String())
			}
			if stdout.String() != test.wantOutput {
				t.Fatalf("Run() stdout = %q, want %q", stdout.String(), test.wantOutput)
			}
		})
	}
}

func TestRunGeneralOptions(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{name: "help", args: []string{"--help"}, wantCode: 0, wantOutput: "ripgrep recursively"},
		{name: "version", args: []string{"--version"}, wantCode: 0, wantOutput: "ripgrep 0.1.0\n"},
		{name: "missing pattern", wantCode: 2},
		{name: "unknown option", args: []string{"--unknown"}, wantCode: 2},
		{name: "type list", args: []string{"--type-list"}, wantCode: 0, wantOutput: "go:"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, strings.NewReader(""), &stdout, &stderr)
			if code != test.wantCode {
				t.Fatalf("Run() code = %d, want %d", code, test.wantCode)
			}
			if test.wantOutput != "" && !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("Run() stdout = %q, want substring %q", stdout.String(), test.wantOutput)
			}
		})
	}
}

func TestRunWithPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte("user:alice\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"alice", path}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "alice") {
		t.Fatalf("Run() code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunPathAndReaderErrors(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		reader interface{ Read([]byte) (int, error) }
	}{
		{name: "missing path", args: []string{"x", filepath.Join(t.TempDir(), "missing")}, reader: strings.NewReader("")},
		{name: "reader failure", args: []string{"x", "-"}, reader: errorReader{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(test.args, test.reader, &stdout, &stderr); code != 2 {
				t.Fatalf("Run() code = %d, want 2", code)
			}
		})
	}
}

func TestRunWriterError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "help writer", args: []string{"--help"}},
		{name: "result writer", args: []string{"user", "-"}},
		{name: "type list writer", args: []string{"--type-list"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code := Run(test.args, strings.NewReader("user:a\n"), failingWriter{}, failingWriter{})
			if code != 2 {
				t.Fatalf("Run() code = %d, want 2", code)
			}
		})
	}
}

var errWrite = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
