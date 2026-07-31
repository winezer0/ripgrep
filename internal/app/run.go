// Package app 提供 gogrep 命令行程序实现。
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/winezer0/gogrep"
	"github.com/winezer0/gogrep/pkg/ignore"
	"github.com/winezer0/gogrep/pkg/matcher"
	"github.com/winezer0/gogrep/pkg/printer"
	"github.com/winezer0/gogrep/pkg/searcher"
)

const version = "0.1.0"

// Run 执行 CLI；参数为命令行参数及标准流，返回 gogrep 兼容退出码。
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cli, err := parseArgs(args)
	if err != nil {
		return errorExit(stderr, err)
	}
	if cli.Help {
		return writeText(stdout, helpMessage)
	}
	if cli.Version {
		return writeText(stdout, "gogrep "+version+"\n")
	}
	if cli.TypeList {
		return printTypes(stdout)
	}
	if cli.Pattern == "" {
		return errorExit(stderr, fmt.Errorf("PATTERN is required; see 'gogrep --help'"))
	}
	if useStdin(cli.Paths, stdin) {
		return runStdin(cli, stdin, stdout, stderr)
	}
	if len(cli.Paths) == 0 {
		cli.Paths = []string{"."}
	}
	return runPaths(cli, stdout, stderr)
}

func runPaths(cli *cliArgs, stdout, stderr io.Writer) int {
	results, err := gogrep.Search(context.Background(), cli.Paths, cli.options())
	if err != nil {
		return errorExit(stderr, err)
	}
	return printResults(cli, results, stdout, stderr)
}

func runStdin(cli *cliArgs, stdin io.Reader, stdout, stderr io.Writer) int {
	compiled, err := matcher.Build(cli.Pattern, matcher.Options{
		Fixed: cli.FixedStrings, CaseInsensitive: cli.CaseInsensitive,
		WordRegexp: cli.WordRegexp,
	})
	if err != nil {
		return errorExit(stderr, err)
	}
	fileSearcher := searcher.New(searcher.Config{
		Matcher: compiled, BeforeContext: cli.BeforeContext, AfterContext: cli.AfterContext,
		MaxCount: cli.MaxCount, InvertMatch: cli.InvertMatch,
		Replace: cli.Replace, HasReplace: cli.HasReplace,
	})
	result, searchErr := fileSearcher.SearchReader(stdin, "-")
	closeErr := compiled.Close()
	if err := joinErrors(searchErr, closeErr); err != nil {
		return errorExit(stderr, err)
	}
	return printResults(cli, []gogrep.FileResult{*result}, stdout, stderr)
}

func printResults(cli *cliArgs, results []gogrep.FileResult, stdout, stderr io.Writer) int {
	totalMatches := 0
	totalLines := 0
	var elapsed time.Duration
	output := printer.NewPrinter(stdout, cli.printerConfig())
	for _, result := range results {
		totalMatches += result.Stats.Matches
		totalLines += result.Stats.SearchedLines
		elapsed += result.Elapsed
		if cli.Quiet {
			continue
		}
		if err := output.PrintFileResult(result); err != nil {
			return errorExit(stderr, err)
		}
	}
	if cli.JSON && !cli.Quiet {
		if err := output.PrintSummary(len(results), totalMatches, totalLines, elapsed); err != nil {
			return errorExit(stderr, err)
		}
	}
	if totalMatches == 0 {
		return 1
	}
	return 0
}

func printTypes(writer io.Writer) int {
	names := make([]string, 0, len(ignore.BuiltInTypes))
	for name := range ignore.BuiltInTypes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", name, strings.Join(ignore.BuiltInTypes[name], ", ")); err != nil {
			return 2
		}
	}
	return 0
}

func useStdin(paths []string, reader io.Reader) bool {
	if len(paths) == 1 && paths[0] == "-" {
		return true
	}
	if len(paths) != 0 {
		return false
	}
	file, ok := reader.(*os.File)
	if !ok {
		return true
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice == 0
}

func writeText(writer io.Writer, value string) int {
	if _, err := io.WriteString(writer, value); err != nil {
		return 2
	}
	return 0
}

func errorExit(writer io.Writer, err error) int {
	if _, writeErr := fmt.Fprintf(writer, "error: %v\n", err); writeErr != nil {
		return 2
	}
	return 2
}

func joinErrors(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
