package ripgrep

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	ignore "github.com/winezer0/gitignore"
	typefilter "github.com/winezer0/ripgrep/pkg/ignore"
)

type walker struct {
	pipeline *pipeline
	visited  map[string]struct{}
}

func newWalker(pipeline *pipeline) *walker {
	return &walker{pipeline: pipeline, visited: make(map[string]struct{})}
}

func (w *walker) walkPaths(paths []string) error {
	for _, path := range paths {
		if err := w.walkPath(path); err != nil {
			return err
		}
	}
	return nil
}

func (w *walker) walkPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	resolvedPath, isDirectory, err := w.resolve(path, info.Mode())
	if err != nil {
		return err
	}
	if !isDirectory {
		include, includeErr := w.shouldIncludeFile(resolvedPath)
		if includeErr != nil {
			return includeErr
		}
		if include {
			return w.sendFile(resolvedPath)
		}
		return nil
	}
	stack := ignore.NewIgnoreStack(w.pipeline.options.NoIgnore, w.pipeline.options.Hidden, w.pipeline.options.MaxDepth)
	if err := stack.LoadBaseRules(resolvedPath); err != nil {
		return err
	}
	return w.walkDirectory(resolvedPath, stack, 1)
}

func (w *walker) walkDirectory(path string, stack *ignore.IgnoreStack, depth int) error {
	if err := w.pipeline.ctx.Err(); err != nil {
		return err
	}
	if w.pipeline.options.MaxDepth > 0 && depth > w.pipeline.options.MaxDepth {
		return nil
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if _, exists := w.visited[realPath]; exists {
		return nil
	}
	w.visited[realPath] = struct{}{}
	if err := stack.Push(path); err != nil {
		return err
	}
	defer stack.Pop()
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	w.sortEntries(entries)
	for _, entry := range entries {
		if err := w.walkEntry(path, entry, stack, depth); err != nil {
			return err
		}
	}
	return nil
}

func (w *walker) walkEntry(parent string, entry os.DirEntry, stack *ignore.IgnoreStack, depth int) error {
	path := filepath.Join(parent, entry.Name())
	if entry.Type()&os.ModeSymlink != 0 && !w.pipeline.options.FollowSymlinks {
		return nil
	}
	resolved, isDirectory, err := w.resolve(path, entry.Type())
	if err != nil {
		return err
	}
	ignored, err := stack.IsIgnored(path, isDirectory)
	if err != nil {
		return err
	}
	if ignored || w.shouldSkipGlob(path, isDirectory) {
		return nil
	}
	if isDirectory {
		return w.walkDirectory(resolved, stack.Clone(), depth+1)
	}
	include, err := w.shouldIncludeFile(path)
	if err != nil {
		return err
	}
	if include {
		return w.sendFile(path)
	}
	return nil
}

func (w *walker) resolve(path string, mode os.FileMode) (string, bool, error) {
	isSymlink := mode&os.ModeSymlink != 0
	if !isSymlink || !w.pipeline.options.FollowSymlinks {
		entry, err := os.Stat(path)
		if err != nil {
			return "", false, err
		}
		return path, entry.IsDir(), nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false, err
	}
	entry, err := os.Stat(resolved)
	if err != nil {
		return "", false, err
	}
	return resolved, entry.IsDir(), nil
}

func (w *walker) shouldIncludeFile(path string) (bool, error) {
	ignored, err := typefilter.ShouldIgnoreByType(filepath.Base(path), w.pipeline.options.Types, w.pipeline.options.TypesNot)
	if err != nil {
		return false, err
	}
	if ignored {
		return false, nil
	}
	return !w.shouldSkipGlob(path, false), nil
}

func (w *walker) shouldSkipGlob(path string, directory bool) bool {
	if w.pipeline.globs == nil {
		return false
	}
	if directory {
		return w.pipeline.globs.MatchGlobFilterDir(path)
	}
	return w.pipeline.globs.MatchGlobFilter(path)
}

func (w *walker) sendFile(path string) error {
	select {
	case w.pipeline.files <- path:
		return nil
	case <-w.pipeline.ctx.Done():
		return w.pipeline.ctx.Err()
	}
}

func (w *walker) sortEntries(entries []os.DirEntry) {
	sortBy := w.pipeline.options.SortBy
	if sortBy == "" || sortBy == "none" {
		return
	}
	sort.SliceStable(entries, func(left, right int) bool {
		comparison := w.compareEntries(entries[left], entries[right], sortBy)
		if w.pipeline.options.SortReverse {
			return comparison > 0
		}
		return comparison < 0
	})
}

func (w *walker) compareEntries(left, right os.DirEntry, sortBy string) int {
	if sortBy == "path" {
		return strings.Compare(left.Name(), right.Name())
	}
	leftInfo, leftErr := left.Info()
	rightInfo, rightErr := right.Info()
	if leftErr != nil || rightErr != nil {
		return strings.Compare(left.Name(), right.Name())
	}
	if sortBy == "modified" {
		if leftInfo.ModTime().Before(rightInfo.ModTime()) {
			return -1
		}
		if leftInfo.ModTime().After(rightInfo.ModTime()) {
			return 1
		}
	} else if leftInfo.Size() != rightInfo.Size() {
		if leftInfo.Size() < rightInfo.Size() {
			return -1
		}
		return 1
	}
	return strings.Compare(left.Name(), right.Name())
}
