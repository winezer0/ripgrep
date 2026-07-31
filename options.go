package ripgrep

import "github.com/winezer0/ripgrep/pkg/printer"

// Options 配置一次目录或文件搜索。
type Options struct {
	Pattern         string
	FixedStrings    bool
	CaseInsensitive bool
	WordRegexp      bool
	InvertMatch     bool
	Replace         string
	HasReplace      bool
	NoIgnore        bool
	Hidden          bool
	FollowSymlinks  bool
	MaxDepth        int
	Globs           []string
	Types           []string
	TypesNot        []string
	SearchZip       bool
	SortBy          string
	SortReverse     bool
	BeforeContext   int
	AfterContext    int
	MaxCount        int
	Threads         int
	MatchLimit      uint32
	DepthLimit      uint32
}

// FileResult 是单个文件的搜索结果。
type FileResult = printer.FileResult

// SearchMatch 是匹配行或上下文行。
type SearchMatch = printer.SearchMatch

// Submatch 是匹配行中的 UTF-8 字节范围。
type Submatch = printer.Submatch

// FileStats 是单个文件的搜索统计。
type FileStats = printer.FileStats

// Stream 保存流式搜索产生的结果和错误通道。
type Stream struct {
	Results <-chan FileResult
	Errors  <-chan error
}
