package printer

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPrinterGrouped(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Group:        true,
		Color:        false,
		WithLineNum:  true,
		WithFilename: true,
	}
	p := NewPrinter(&buf, cfg)

	res := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{
				Line:      "hello world\n",
				LineNum:   1,
				IsContext: false,
				Submatches: []Submatch{
					{Start: 0, End: 5, Text: "hello"},
				},
			},
		},
		Stats: FileStats{SearchedLines: 10, Matches: 1},
	}

	if err := p.PrintFileResult(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "test.txt\n1:hello world\n\n"
	if buf.String() != expected {
		t.Errorf("expected output:\n%q\ngot:\n%q", expected, buf.String())
	}
}

func TestPrinterModes(t *testing.T) {
	result := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{Line: "hello world", LineNum: 2, Submatches: []Submatch{{Start: 6, End: 11, Text: "world"}}},
			{Line: "context", LineNum: 3, IsContext: true},
		},
		Stats: FileStats{SearchedLines: 3, Matches: 1},
	}
	tests := []struct {
		name   string
		config Config
		result FileResult
		want   string
	}{
		{name: "empty result", result: FileResult{Path: "empty"}, want: ""},
		{name: "count with filename", config: Config{Count: true, WithFilename: true}, result: result, want: "test.txt:1\n"},
		{name: "count only", config: Config{Count: true}, result: result, want: "1\n"},
		{name: "colored count", config: Config{Count: true, WithFilename: true, Color: true}, result: result, want: "\x1b[35mtest.txt\x1b[0m:1\n"},
		{name: "context separators", config: Config{WithFilename: true, WithLineNum: true}, result: result, want: "test.txt:2:hello world\ntest.txt-3-context\n"},
		{name: "only match with column", config: Config{WithFilename: true, WithLineNum: true, WithColumnNum: true, OnlyMatching: true}, result: result, want: "test.txt:2:7:world\ntest.txt-3-context\n"},
		{name: "colored grouped", config: Config{Group: true, Color: true, WithFilename: true, WithLineNum: true}, result: result, want: "\x1b[35mtest.txt\x1b[0m\n\x1b[32m2\x1b[0m:hello \x1b[1;31mworld\x1b[0m\n\x1b[32m3\x1b[0m-context\n\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := NewPrinter(&output, test.config).PrintFileResult(test.result); err != nil {
				t.Fatalf("PrintFileResult() error = %v", err)
			}
			if output.String() != test.want {
				t.Fatalf("PrintFileResult() = %q, want %q", output.String(), test.want)
			}
		})
	}
}

func TestPrintSummary(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{name: "text has no summary", config: Config{}, want: ""},
		{name: "JSON summary", config: Config{JSON: true}, want: `"type":"summary"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := NewPrinter(&output, test.config).PrintSummary(1, 2, 3, time.Second); err != nil {
				t.Fatalf("PrintSummary() error = %v", err)
			}
			if !strings.Contains(output.String(), test.want) {
				t.Fatalf("PrintSummary() = %q, want substring %q", output.String(), test.want)
			}
		})
	}
}

func TestPrinterWriteErrors(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{name: "text", config: Config{WithFilename: true}},
		{name: "JSON", config: Config{JSON: true}},
		{name: "summary", config: Config{JSON: true}},
	}
	result := FileResult{Path: "test", Matches: []SearchMatch{{Line: "x", LineNum: 1}}, Stats: FileStats{Matches: 1}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			printer := NewPrinter(errorWriter{}, test.config)
			var err error
			if test.name == "summary" {
				err = printer.PrintSummary(1, 1, 1, time.Second)
			} else {
				err = printer.PrintFileResult(result)
			}
			if !errors.Is(err, errOutput) {
				t.Fatalf("write error = %v, want %v", err, errOutput)
			}
		})
	}
	_ = IsTerminal()
}

var errOutput = errors.New("output failed")

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errOutput }

func TestPrinterNonGrouped(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Group:        false,
		Color:        false,
		WithLineNum:  true,
		WithFilename: true,
	}
	p := NewPrinter(&buf, cfg)

	res := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{
				Line:      "hello world\n",
				LineNum:   1,
				IsContext: false,
				Submatches: []Submatch{
					{Start: 0, End: 5, Text: "hello"},
				},
			},
		},
		Stats: FileStats{SearchedLines: 10, Matches: 1},
	}

	if err := p.PrintFileResult(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "test.txt:1:hello world\n"
	if buf.String() != expected {
		t.Errorf("expected output:\n%q\ngot:\n%q", expected, buf.String())
	}
}

func TestPrinterJSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{JSON: true}
	p := NewPrinter(&buf, cfg)

	res := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{
				Line:      "hello world\n",
				LineNum:   1,
				IsContext: false,
				Submatches: []Submatch{
					{Start: 0, End: 5, Text: "hello"},
				},
			},
		},
		Stats: FileStats{SearchedLines: 10, Matches: 1},
	}

	if err := p.PrintFileResult(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 JSON messages (begin, match, end), got %d lines: %v", len(lines), lines)
	}

	// Verify "begin"
	var beginMsg jsonMessage
	if err := json.Unmarshal([]byte(lines[0]), &beginMsg); err != nil {
		t.Fatalf("failed to decode begin: %v", err)
	}
	if beginMsg.Type != "begin" {
		t.Errorf("expected first message type 'begin', got %q", beginMsg.Type)
	}

	// Verify "match"
	var matchMsg jsonMessage
	if err := json.Unmarshal([]byte(lines[1]), &matchMsg); err != nil {
		t.Fatalf("failed to decode match: %v", err)
	}
	if matchMsg.Type != "match" {
		t.Errorf("expected second message type 'match', got %q", matchMsg.Type)
	}

	// Verify "end"
	var endMsg jsonMessage
	if err := json.Unmarshal([]byte(lines[2]), &endMsg); err != nil {
		t.Fatalf("failed to decode end: %v", err)
	}
	if endMsg.Type != "end" {
		t.Errorf("expected third message type 'end', got %q", endMsg.Type)
	}
}
