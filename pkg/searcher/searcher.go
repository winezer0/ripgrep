// Package searcher 提供面向文件和数据流的逐行搜索能力。
package searcher

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/winezer0/ripgrep/pkg/matcher"
	"github.com/winezer0/ripgrep/pkg/printer"
)

const maxLineLength = 10 * 1024 * 1024

// Config 配置单文件搜索行为。
type Config struct {
	Matcher       matcher.Matcher
	BeforeContext int
	AfterContext  int
	MaxCount      int
	InvertMatch   bool
	Replace       string
	HasReplace    bool
	SearchZip     bool
}

// Searcher 执行单个文件或数据流的搜索。
type Searcher struct {
	config Config
	ctx    context.Context
}

// New 使用配置创建搜索器；Matcher 不能为空。
func New(config Config) *Searcher {
	return &Searcher{config: config, ctx: context.Background()}
}

// SetContext 设置取消上下文；传入 nil 时恢复为 Background。
func (s *Searcher) SetContext(ctx context.Context) {
	if ctx == nil {
		s.ctx = context.Background()
		return
	}
	s.ctx = ctx
}

// SearchReader 搜索输入流，并返回路径对应的完整结果。
func (s *Searcher) SearchReader(reader io.Reader, path string) (*printer.FileResult, error) {
	if err := s.checkCancelled(); err != nil {
		return nil, err
	}
	buffered := bufio.NewReader(reader)
	prefix, err := buffered.Peek(1024)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return nil, err
	}
	if bytes.IndexByte(prefix, 0) >= 0 {
		return &printer.FileResult{Path: path}, nil
	}

	state := newLineState(s.config.BeforeContext)
	scanner := bufio.NewScanner(buffered)
	scanner.Buffer(make([]byte, 64*1024), maxLineLength)
	for scanner.Scan() {
		if err := s.checkCancelled(); err != nil {
			return nil, err
		}
		state.lines++
		if err := s.processLine(scanner.Bytes(), state.lines, state); err != nil {
			return nil, err
		}
		if s.config.MaxCount > 0 && state.matchCount >= s.config.MaxCount {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return state.result(path), nil
}

func (s *Searcher) checkCancelled() error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return nil
	}
}
