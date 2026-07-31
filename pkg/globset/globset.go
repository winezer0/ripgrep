package globset

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Glob 表示一个已编译的 glob 模式。
type Glob struct {
	Original  string
	Regexp    *regexp.Regexp
	IsNegated bool
}

// NewGlob 编译 glob 模式并返回其匹配结构。
func NewGlob(pattern string) (*Glob, error) {
	isNegated := false
	if strings.HasPrefix(pattern, "!") {
		isNegated = true
		pattern = pattern[1:]
	}

	// Normalize windows separators
	pattern = filepath.ToSlash(pattern)

	regexStr, err := GlobToRegex(pattern)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}

	return &Glob{
		Original:  pattern,
		Regexp:    re,
		IsNegated: isNegated,
	}, nil
}

// Match 判断路径是否匹配当前 glob。
func (g *Glob) Match(path string) bool {
	path = filepath.ToSlash(path)
	return g.Regexp.MatchString(path)
}

// GlobToRegex 将 gitignore 风格 glob 转换为 Go 正则表达式。
func GlobToRegex(pattern string) (string, error) {
	var sb strings.Builder

	isAnchored := strings.HasPrefix(pattern, "/")
	trimmed := pattern
	if isAnchored {
		trimmed = pattern[1:]
		pattern = trimmed
	}
	// Check for middle slash (after stripping leading/trailing slashes)
	if strings.Contains(strings.TrimSuffix(trimmed, "/"), "/") {
		isAnchored = true
	}

	if !isAnchored {
		// If not anchored, it matches any component of the path.
		sb.WriteString(`(?:^|/)`)
	} else {
		// If anchored, it starts matching from the root.
		sb.WriteString(`^`)
	}

	runes := []rune(pattern)
	n := len(runes)
	inBracket := false
	for i := 0; i < n; i++ {
		r := runes[i]
		switch r {
		case '*':
			if i+1 < n && runes[i+1] == '*' {
				i++
				if i+1 < n && runes[i+1] == '/' {
					i++
					// '**/': match zero or more directory levels
					sb.WriteString(`(?:.*/)?`)
				} else {
					// '**': match everything
					sb.WriteString(`.*`)
				}
			} else {
				// '*': match anything within a single directory level
				sb.WriteString(`[^/]*`)
			}
		case '?':
			sb.WriteString(`[^/]`)
		case '[':
			inBracket = true
			sb.WriteRune('[')
			if i+1 < n && runes[i+1] == '!' {
				sb.WriteRune('^')
				i++
			}
		case ']':
			inBracket = false
			sb.WriteRune(']')
		case '\\':
			if i+1 < n {
				i++
				sb.WriteString(regexp.QuoteMeta(string(runes[i])))
			} else {
				sb.WriteString(`\\`)
			}
		case '.', '+', '$', '^', '(', ')', '|', '{', '}':
			if inBracket {
				sb.WriteRune(r)
			} else {
				sb.WriteString(`\` + string(r))
			}
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteString(`$`)
	return sb.String(), nil
}

// GlobSet 保存一组已编译的 glob。
type GlobSet struct {
	globs []*Glob
}

// NewGlobSet 编译 glob 列表，空行和注释行会被忽略。
func NewGlobSet(patterns []string) (*GlobSet, error) {
	var globs []*Glob
	for _, pat := range patterns {
		if pat == "" || strings.HasPrefix(pat, "#") {
			// skip empty lines and comments
			continue
		}
		g, err := NewGlob(pat)
		if err != nil {
			return nil, err
		}
		globs = append(globs, g)
	}
	return &GlobSet{globs: globs}, nil
}

// Match 按最后匹配优先规则判断路径，并返回是否命中及是否排除。
func (gs *GlobSet) Match(path string) (matched bool, isIgnored bool) {
	for i := len(gs.globs) - 1; i >= 0; i-- {
		g := gs.globs[i]
		if g.Match(path) {
			if g.IsNegated {
				// Negated match: means do NOT ignore/exclude
				return true, false
			}
			// Regular match: means DO ignore/exclude
			return true, true
		}
	}
	return false, false
}

// MatchPath 检查路径及其父目录，以支持目录级排除规则。
func (gs *GlobSet) MatchPath(path string) (matched bool, isIgnored bool) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	var current string
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		if current == "" {
			current = parts[i]
		} else {
			current = current + "/" + parts[i]
		}

		// Check with trailing slash (e.g. "bin/")
		if m, ig := gs.Match(current + "/"); m {
			if ig {
				return true, true
			}
		}
		// Check without trailing slash (e.g. "bin")
		if m, ig := gs.Match(current); m {
			if ig {
				return true, true
			}
		}
	}

	return gs.Match(path)
}

// MatchGlobFilter 按 ripgrep 的 -g/--glob 规则判断文件是否应排除。
// 1. If there are negated globs (starting with '!') and the path matches one, it is ignored (returns true).
// 2. If there are positive globs:
//   - If the path matches a positive glob, it is NOT ignored (returns false).
//   - If it does not match any positive glob, it IS ignored (returns true).
//
// 3. Otherwise, it is NOT ignored (returns false).
func (gs *GlobSet) MatchGlobFilter(path string) bool {
	if len(gs.globs) == 0 {
		return false
	}

	path = filepath.ToSlash(path)

	// Check if there are any positive globs in the set
	hasPositive := false
	for _, g := range gs.globs {
		if !g.IsNegated {
			hasPositive = true
			break
		}
	}

	// 1. If path matches a negated glob, it is ignored/excluded
	for _, g := range gs.globs {
		if g.IsNegated {
			if g.Match(path) {
				return true
			}
		}
	}

	// 2. If there are positive globs, path must match at least one to be included
	if hasPositive {
		matchedPositive := false
		for _, g := range gs.globs {
			if !g.IsNegated {
				if g.Match(path) {
					matchedPositive = true
					break
				}
			}
		}
		if !matchedPositive {
			return true // ignored because it didn't match any positive glob
		}
	}

	return false
}

// MatchGlobFilterDir 判断遍历时是否应排除目录；正向 glob 不会单独排除父目录。
func (gs *GlobSet) MatchGlobFilterDir(path string) bool {
	if len(gs.globs) == 0 {
		return false
	}

	path = filepath.ToSlash(path)
	for _, g := range gs.globs {
		if g.IsNegated && g.Match(path) {
			return true
		}
	}
	return false
}
