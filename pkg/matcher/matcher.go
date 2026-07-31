// Package matcher 提供基于 PCRE2 的文本匹配能力。
package matcher

import (
	"bytes"
	"regexp"

	pcre2 "github.com/VillanCh/go-pcre2-lite"
)

// Matcher 定义搜索器所需的匹配、替换和资源释放行为。
type Matcher interface {
	Match(line []byte) (bool, error)
	FindSpans(line []byte) ([]Span, error)
	Replace(line []byte, replacement string) ([]byte, []Span, error)
	Close() error
}

// Span 表示左闭右开的 UTF-8 字节范围。
type Span struct {
	Start int
	End   int
}

// Options 配置 PCRE2 模式的编译行为和执行限制。
type Options struct {
	Fixed           bool
	CaseInsensitive bool
	WordRegexp      bool
	MatchLimit      uint32
	DepthLimit      uint32
}

// PCRE2Matcher 封装一个可并发使用的 PCRE2 正则表达式。
type PCRE2Matcher struct {
	re *pcre2.Regexp
}

// Build 根据模式与选项构建 PCRE2 匹配器；返回值由调用方负责关闭。
func Build(pattern string, opts Options) (Matcher, error) {
	compiledPattern := pattern
	if opts.Fixed {
		compiledPattern = regexp.QuoteMeta(compiledPattern)
	}
	if opts.WordRegexp {
		compiledPattern = `\b(?:` + compiledPattern + `)\b`
	}

	re, err := pcre2.Compile(compiledPattern, pcre2.CompileOptions{
		UTF:        true,
		UCP:        true,
		Caseless:   opts.CaseInsensitive,
		MatchLimit: opts.MatchLimit,
		DepthLimit: opts.DepthLimit,
	})
	if err != nil {
		return nil, err
	}
	return &PCRE2Matcher{re: re}, nil
}

// Match 判断输入行是否包含匹配，并返回 PCRE2 执行错误。
func (m *PCRE2Matcher) Match(line []byte) (bool, error) {
	return m.re.Match(line)
}

// FindSpans 返回输入行内全部非重叠匹配的 UTF-8 字节范围。
func (m *PCRE2Matcher) FindSpans(line []byte) ([]Span, error) {
	matches, err := m.re.FindAll(line, -1)
	if err != nil {
		return nil, err
	}
	spans := make([]Span, 0, len(matches))
	for _, match := range matches {
		group := match.Groups[0]
		spans = append(spans, Span{Start: group.Start, End: group.End})
	}
	return spans, nil
}

// Replace 替换全部匹配，并返回替换文本在结果中的字节范围。
func (m *PCRE2Matcher) Replace(line []byte, replacement string) ([]byte, []Span, error) {
	matches, err := m.re.FindAll(line, -1)
	if err != nil {
		return nil, nil, err
	}
	if len(matches) == 0 {
		return line, nil, nil
	}

	var result bytes.Buffer
	spans := make([]Span, 0, len(matches))
	lastEnd := 0
	for index := range matches {
		match := &matches[index]
		whole := match.Groups[0]
		result.Write(line[lastEnd:whole.Start])
		start := result.Len()
		result.Write(expandReplacement(m.re, match, replacement))
		spans = append(spans, Span{Start: start, End: result.Len()})
		lastEnd = whole.End
	}
	result.Write(line[lastEnd:])
	return result.Bytes(), spans, nil
}

// Close 释放匹配器持有的 PCRE2 C 资源。
func (m *PCRE2Matcher) Close() error {
	return m.re.Close()
}

func expandReplacement(re *pcre2.Regexp, match *pcre2.Match, replacement string) []byte {
	var result bytes.Buffer
	for index := 0; index < len(replacement); {
		if replacement[index] != '$' || index+1 >= len(replacement) {
			result.WriteByte(replacement[index])
			index++
			continue
		}
		consumed, group, literal := replacementGroup(re, replacement[index:])
		if consumed == 0 {
			result.WriteByte(replacement[index])
			index++
			continue
		}
		if literal {
			result.WriteByte('$')
		} else {
			result.Write(match.Group(group))
		}
		index += consumed
	}
	return result.Bytes()
}

func replacementGroup(re *pcre2.Regexp, value string) (int, int, bool) {
	if len(value) >= 2 && value[1] == '$' {
		return 2, 0, true
	}
	if len(value) >= 2 && value[1] == '&' {
		return 2, 0, false
	}
	if len(value) >= 3 && value[1] == '{' {
		return bracedGroup(re, value)
	}
	end := 1
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 1 {
		return 0, 0, false
	}
	return end, parseGroupNumber(value[1:end]), false
}

func bracedGroup(re *pcre2.Regexp, value string) (int, int, bool) {
	end := bytes.IndexByte([]byte(value[2:]), '}')
	if end < 0 {
		return 0, 0, false
	}
	end += 2
	name := value[2:end]
	if number, ok := re.NamedGroupNumber(name); ok {
		return end + 1, number, false
	}
	if name == "" {
		return 0, 0, false
	}
	return end + 1, parseGroupNumber(name), false
}

func parseGroupNumber(value string) int {
	number := 0
	for _, char := range value {
		if char < '0' || char > '9' {
			return -1
		}
		number = number*10 + int(char-'0')
	}
	return number
}
