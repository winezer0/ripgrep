package app

import (
	"fmt"
	"strconv"
	"strings"

	ripgrep "github.com/winezer0/ripgrep"
	"github.com/winezer0/ripgrep/pkg/printer"
)

type cliArgs struct {
	Pattern, Replace, Color, SortBy string
	Paths, Globs, Types, TypesNot   []string
	CaseInsensitive, WordRegexp     bool
	FixedStrings, InvertMatch       bool
	HasReplace, NoIgnore, Hidden    bool
	FollowSymlinks, SearchZip       bool
	JSON, Heading, NoHeading        bool
	LineNumber, NoLineNumber        bool
	WithFilename, NoFilename        bool
	Column, OnlyMatching, Count     bool
	Quiet, SortReverse              bool
	Help, Version, TypeList         bool
	BeforeContext, AfterContext     int
	MaxCount, Threads, MaxDepth     int
}

func parseArgs(args []string) (*cliArgs, error) {
	cli := &cliArgs{Color: "auto"}
	positionals := make([]string, 0)
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if strings.HasPrefix(argument, "--") {
			consumed, err := parseLong(cli, argument[2:], args[index+1:])
			if err != nil {
				return nil, err
			}
			index += consumed
			continue
		}
		if strings.HasPrefix(argument, "-") && argument != "-" {
			consumed, err := parseShort(cli, argument[1:], args[index+1:])
			if err != nil {
				return nil, err
			}
			index += consumed
			continue
		}
		positionals = append(positionals, argument)
	}
	if len(positionals) > 0 {
		cli.Pattern = positionals[0]
		cli.Paths = positionals[1:]
	}
	return cli, nil
}

func parseLong(cli *cliArgs, argument string, following []string) (int, error) {
	name, inline, hasInline := strings.Cut(argument, "=")
	if longBoolean(cli, name) {
		if hasInline {
			return 0, fmt.Errorf("option --%s does not take a value", name)
		}
		return 0, nil
	}
	value, consumed, err := optionValue("--"+name, inline, hasInline, following)
	if err != nil {
		return 0, err
	}
	if err := applyValue(cli, name, value); err != nil {
		return 0, err
	}
	return consumed, nil
}

func longBoolean(cli *cliArgs, name string) bool {
	switch name {
	case "ignore-case":
		cli.CaseInsensitive = true
	case "case-sensitive":
		cli.CaseInsensitive = false
	case "word-regexp":
		cli.WordRegexp = true
	case "fixed-strings":
		cli.FixedStrings = true
	case "invert-match":
		cli.InvertMatch = true
	case "hidden":
		cli.Hidden = true
	case "no-ignore":
		cli.NoIgnore = true
	case "follow":
		cli.FollowSymlinks = true
	case "json":
		cli.JSON = true
	case "heading":
		cli.Heading = true
		cli.NoHeading = false
	case "no-heading":
		cli.NoHeading = true
		cli.Heading = false
	case "line-number":
		cli.LineNumber = true
		cli.NoLineNumber = false
	case "no-line-number":
		cli.NoLineNumber = true
		cli.LineNumber = false
	case "with-filename":
		cli.WithFilename = true
		cli.NoFilename = false
	case "no-filename":
		cli.NoFilename = true
		cli.WithFilename = false
	case "column":
		cli.Column = true
	case "only-matching":
		cli.OnlyMatching = true
	case "count":
		cli.Count = true
	case "quiet":
		cli.Quiet = true
	case "search-zip":
		cli.SearchZip = true
	case "type-list":
		cli.TypeList = true
	case "help":
		cli.Help = true
	case "version":
		cli.Version = true
	default:
		return false
	}
	return true
}

func applyValue(cli *cliArgs, name, value string) error {
	switch name {
	case "replace":
		cli.Replace, cli.HasReplace = value, true
	case "glob":
		cli.Globs = append(cli.Globs, value)
	case "type":
		cli.Types = append(cli.Types, value)
	case "type-not":
		cli.TypesNot = append(cli.TypesNot, value)
	case "color":
		if value != "auto" && value != "always" && value != "never" {
			return fmt.Errorf("invalid color mode %q", value)
		}
		cli.Color = value
	case "sort":
		if err := validateSort(value); err != nil {
			return err
		}
		cli.SortBy, cli.SortReverse = value, false
	case "sortr":
		if err := validateSort(value); err != nil {
			return err
		}
		cli.SortBy, cli.SortReverse = value, true
	case "after-context":
		return setNonNegative(&cli.AfterContext, name, value)
	case "before-context":
		return setNonNegative(&cli.BeforeContext, name, value)
	case "context":
		var parsed int
		if err := setNonNegative(&parsed, name, value); err != nil {
			return err
		}
		cli.BeforeContext, cli.AfterContext = parsed, parsed
	case "max-count":
		return setNonNegative(&cli.MaxCount, name, value)
	case "threads":
		return setNonNegative(&cli.Threads, name, value)
	case "max-depth":
		return setNonNegative(&cli.MaxDepth, name, value)
	default:
		return fmt.Errorf("unknown option --%s", name)
	}
	return nil
}

func validateSort(value string) error {
	switch value {
	case "path", "modified", "size", "none":
		return nil
	default:
		return fmt.Errorf("invalid sort mode %q", value)
	}
}

func setNonNegative(target *int, name, value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fmt.Errorf("invalid value for --%s: %q", name, value)
	}
	*target = parsed
	return nil
}

func optionValue(name, inline string, hasInline bool, following []string) (string, int, error) {
	if hasInline {
		return inline, 0, nil
	}
	if len(following) == 0 {
		return "", 0, fmt.Errorf("option %s requires a value", name)
	}
	return following[0], 1, nil
}

func (c *cliArgs) options() ripgrep.Options {
	return ripgrep.Options{
		Pattern: c.Pattern, FixedStrings: c.FixedStrings, CaseInsensitive: c.CaseInsensitive,
		WordRegexp: c.WordRegexp, InvertMatch: c.InvertMatch, Replace: c.Replace,
		HasReplace: c.HasReplace, NoIgnore: c.NoIgnore, Hidden: c.Hidden,
		FollowSymlinks: c.FollowSymlinks, MaxDepth: c.MaxDepth, Globs: c.Globs,
		Types: c.Types, TypesNot: c.TypesNot, SearchZip: c.SearchZip,
		SortBy: c.SortBy, SortReverse: c.SortReverse, BeforeContext: c.BeforeContext,
		AfterContext: c.AfterContext, MaxCount: c.MaxCount, Threads: c.Threads,
	}
}

func (c *cliArgs) printerConfig() printer.Config {
	terminal := printer.IsTerminal()
	return printer.Config{
		Group: c.Heading || (!c.NoHeading && terminal),
		Color: c.Color == "always" || (c.Color == "auto" && terminal && !c.JSON), JSON: c.JSON,
		WithLineNum: !c.NoLineNumber, WithFilename: !c.NoFilename,
		WithColumnNum: c.Column, OnlyMatching: c.OnlyMatching, Count: c.Count,
	}
}
