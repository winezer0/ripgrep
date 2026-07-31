package app

import "fmt"

func parseShort(cli *cliArgs, flags string, following []string) (int, error) {
	for index, flag := range flags {
		name, takesValue, ok := shortName(flag)
		if !ok {
			return 0, fmt.Errorf("unknown option -%c", flag)
		}
		if !takesValue {
			if !longBoolean(cli, name) {
				return 0, fmt.Errorf("unknown option -%c", flag)
			}
			continue
		}
		remainder := flags[index+1:]
		value, consumed, err := optionValue("-"+string(flag), remainder, remainder != "", following)
		if err != nil {
			return 0, err
		}
		if err := applyValue(cli, name, value); err != nil {
			return 0, err
		}
		return consumed, nil
	}
	return 0, nil
}

func shortName(flag rune) (string, bool, bool) {
	switch flag {
	case 'i':
		return "ignore-case", false, true
	case 's':
		return "case-sensitive", false, true
	case 'w':
		return "word-regexp", false, true
	case 'F':
		return "fixed-strings", false, true
	case 'v':
		return "invert-match", false, true
	case 'L':
		return "follow", false, true
	case 'n':
		return "line-number", false, true
	case 'N':
		return "no-line-number", false, true
	case 'H':
		return "with-filename", false, true
	case 'I':
		return "no-filename", false, true
	case 'o':
		return "only-matching", false, true
	case 'c':
		return "count", false, true
	case 'q':
		return "quiet", false, true
	case 'z':
		return "search-zip", false, true
	case 'h':
		return "help", false, true
	case 'V':
		return "version", false, true
	case 'r':
		return "replace", true, true
	case 'g':
		return "glob", true, true
	case 't':
		return "type", true, true
	case 'T':
		return "type-not", true, true
	case 'A':
		return "after-context", true, true
	case 'B':
		return "before-context", true, true
	case 'C':
		return "context", true, true
	case 'm':
		return "max-count", true, true
	case 'j':
		return "threads", true, true
	default:
		return "", false, false
	}
}
