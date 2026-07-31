package searcher

import (
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/winezer0/gogrep/pkg/printer"
)

// SearchFile 搜索普通文件；启用 SearchZip 时也支持 gzip、bzip2 和 zip。
func (s *Searcher) SearchFile(path string) ([]*printer.FileResult, error) {
	if err := s.checkCancelled(); err != nil {
		return nil, err
	}
	if s.config.SearchZip {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".gz":
			return s.searchGzip(path)
		case ".bz2":
			return s.searchBzip2(path)
		case ".zip":
			return s.searchZip(path)
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	result, searchErr := s.SearchReader(file, path)
	closeErr := file.Close()
	return resultSlice(result), errors.Join(searchErr, closeErr)
}

func (s *Searcher) searchGzip(path string) ([]*printer.FileResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	result, searchErr := s.SearchReader(reader, path)
	return resultSlice(result), errors.Join(searchErr, reader.Close(), file.Close())
}

func (s *Searcher) searchBzip2(path string) ([]*printer.FileResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	result, searchErr := s.SearchReader(bzip2.NewReader(file), path)
	return resultSlice(result), errors.Join(searchErr, file.Close())
}

func (s *Searcher) searchZip(path string) ([]*printer.FileResult, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	results := make([]*printer.FileResult, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		entry, openErr := file.Open()
		if openErr != nil {
			return nil, errors.Join(openErr, reader.Close())
		}
		result, searchErr := s.SearchReader(entry, path+"//"+file.Name)
		if err := errors.Join(searchErr, entry.Close()); err != nil {
			return nil, errors.Join(err, reader.Close())
		}
		results = append(results, result)
	}
	return results, reader.Close()
}

func resultSlice(result *printer.FileResult) []*printer.FileResult {
	if result == nil {
		return nil
	}
	return []*printer.FileResult{result}
}
