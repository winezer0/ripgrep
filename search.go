package gogrep

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/winezer0/gogrep/pkg/globset"
	"github.com/winezer0/gogrep/pkg/ignore"
	"github.com/winezer0/gogrep/pkg/matcher"
	"github.com/winezer0/gogrep/pkg/searcher"
)

type pipeline struct {
	ctx      context.Context
	cancel   context.CancelFunc
	options  Options
	matcher  matcher.Matcher
	globs    *globset.GlobSet
	files    chan string
	results  chan FileResult
	errors   chan error
	walkDone chan struct{}
	workerWg sync.WaitGroup
}

// Search 执行阻塞式搜索，返回全部匹配结果和搜索期间发生的聚合错误。
func Search(ctx context.Context, paths []string, options Options) ([]FileResult, error) {
	stream, err := SearchStream(ctx, paths, options)
	if err != nil {
		return nil, err
	}
	results := make([]FileResult, 0)
	for result := range stream.Results {
		results = append(results, result)
	}
	var searchErrors []error
	for searchErr := range stream.Errors {
		searchErrors = append(searchErrors, searchErr)
	}
	return results, errors.Join(searchErrors...)
}

// SearchStream 启动流式搜索；调用方必须同时消费 Results 和 Errors。
func SearchStream(ctx context.Context, paths []string, options Options) (*Stream, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	compiled, err := matcher.Build(options.Pattern, matcher.Options{
		Fixed: options.FixedStrings, CaseInsensitive: options.CaseInsensitive,
		WordRegexp: options.WordRegexp, MatchLimit: options.MatchLimit,
		DepthLimit: options.DepthLimit,
	})
	if err != nil {
		return nil, err
	}
	globs, err := buildGlobSet(options.Globs)
	if err != nil {
		if closeErr := compiled.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	threads := workerCount(options)
	pipelineCtx, cancel := context.WithCancel(ctx)
	p := &pipeline{
		ctx: pipelineCtx, cancel: cancel, options: options, matcher: compiled, globs: globs,
		files: make(chan string, threads*4), results: make(chan FileResult, threads*2),
		errors: make(chan error, threads+2), walkDone: make(chan struct{}),
	}
	p.start(paths, threads)
	return &Stream{Results: p.results, Errors: p.errors}, nil
}

func validateOptions(options Options) error {
	switch options.SortBy {
	case "", "none", "path", "modified", "size":
	default:
		return fmt.Errorf("unsupported sort mode %q", options.SortBy)
	}
	if options.BeforeContext < 0 || options.AfterContext < 0 || options.MaxCount < 0 || options.MaxDepth < 0 {
		return errors.New("context, count, and depth values must be non-negative")
	}
	for _, typeName := range append(append([]string(nil), options.Types...), options.TypesNot...) {
		if _, exists := ignore.BuiltInTypes[strings.ToLower(typeName)]; !exists {
			return fmt.Errorf("unknown file type %q", typeName)
		}
	}
	return nil
}

func buildGlobSet(patterns []string) (*globset.GlobSet, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	return globset.NewGlobSet(patterns)
}

func workerCount(options Options) int {
	if options.SortBy != "" && options.SortBy != "none" {
		return 1
	}
	if options.Threads > 0 {
		return options.Threads
	}
	return runtime.NumCPU()
}

func (p *pipeline) start(paths []string, threads int) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	go func() {
		defer close(p.files)
		defer close(p.walkDone)
		if err := newWalker(p).walkPaths(paths); err != nil {
			p.report(err)
		}
	}()
	for index := 0; index < threads; index++ {
		p.workerWg.Add(1)
		go p.runWorker()
	}
	go p.finish()
}

func (p *pipeline) runWorker() {
	defer p.workerWg.Done()
	fileSearcher := searcher.New(searcher.Config{
		Matcher: p.matcher, BeforeContext: p.options.BeforeContext,
		AfterContext: p.options.AfterContext, MaxCount: p.options.MaxCount,
		InvertMatch: p.options.InvertMatch, Replace: p.options.Replace,
		HasReplace: p.options.HasReplace, SearchZip: p.options.SearchZip,
	})
	fileSearcher.SetContext(p.ctx)
	for path := range p.files {
		started := time.Now()
		results, err := fileSearcher.SearchFile(path)
		if err != nil {
			p.report(err)
			return
		}
		for _, result := range results {
			if result == nil || len(result.Matches) == 0 {
				continue
			}
			result.Elapsed = time.Since(started)
			select {
			case p.results <- *result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

func (p *pipeline) report(err error) {
	if err == nil {
		return
	}
	p.errors <- err
	p.cancel()
}

func (p *pipeline) finish() {
	<-p.walkDone
	p.workerWg.Wait()
	if err := p.matcher.Close(); err != nil {
		p.errors <- err
	}
	p.cancel()
	close(p.results)
	close(p.errors)
}
