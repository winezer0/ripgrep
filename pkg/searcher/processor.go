package searcher

import (
	"github.com/winezer0/gogrep/pkg/matcher"
	"github.com/winezer0/gogrep/pkg/printer"
)

type bufferedLine struct {
	number int
	text   string
}

type lineState struct {
	matches        []printer.SearchMatch
	before         []bufferedLine
	afterRemaining int
	lastPrinted    int
	lines          int
	matchCount     int
}

func newLineState(before int) *lineState {
	return &lineState{before: make([]bufferedLine, 0, before)}
}

func (s *Searcher) processLine(line []byte, number int, state *lineState) error {
	hasMatch, err := s.config.Matcher.Match(line)
	if err != nil {
		return err
	}
	reported := hasMatch != s.config.InvertMatch
	if !reported {
		s.processContext(string(line), number, state)
		return nil
	}

	state.matchCount++
	s.flushBefore(state)
	text, submatches, err := s.buildMatch(line)
	if err != nil {
		return err
	}
	if number > state.lastPrinted {
		state.matches = append(state.matches, printer.SearchMatch{
			Line:       text,
			LineNum:    number,
			Submatches: submatches,
		})
		state.lastPrinted = number
	}
	state.afterRemaining = s.config.AfterContext
	return nil
}

func (s *Searcher) buildMatch(line []byte) (string, []printer.Submatch, error) {
	if s.config.InvertMatch {
		return string(line), nil, nil
	}
	if s.config.HasReplace {
		replaced, spans, err := s.config.Matcher.Replace(line, s.config.Replace)
		return string(replaced), buildSubmatches(replaced, spans), err
	}
	spans, err := s.config.Matcher.FindSpans(line)
	return string(line), buildSubmatches(line, spans), err
}

func buildSubmatches(line []byte, spans []matcher.Span) []printer.Submatch {
	submatches := make([]printer.Submatch, 0, len(spans))
	for _, span := range spans {
		submatches = append(submatches, printer.Submatch{
			Start: span.Start,
			End:   span.End,
			Text:  string(line[span.Start:span.End]),
		})
	}
	return submatches
}

func (s *Searcher) flushBefore(state *lineState) {
	for _, line := range state.before {
		if line.number <= state.lastPrinted {
			continue
		}
		state.matches = append(state.matches, printer.SearchMatch{
			Line: line.text, LineNum: line.number, IsContext: true,
		})
		state.lastPrinted = line.number
	}
	state.before = state.before[:0]
}

func (s *Searcher) processContext(text string, number int, state *lineState) {
	if state.afterRemaining > 0 {
		if number > state.lastPrinted {
			state.matches = append(state.matches, printer.SearchMatch{
				Line: text, LineNum: number, IsContext: true,
			})
			state.lastPrinted = number
		}
		state.afterRemaining--
		return
	}
	if s.config.BeforeContext == 0 {
		return
	}
	state.before = append(state.before, bufferedLine{number: number, text: text})
	if len(state.before) > s.config.BeforeContext {
		state.before = state.before[1:]
	}
}

func (s *lineState) result(path string) *printer.FileResult {
	return &printer.FileResult{
		Path:    path,
		Matches: s.matches,
		Stats:   printer.FileStats{SearchedLines: s.lines, Matches: s.matchCount},
	}
}
