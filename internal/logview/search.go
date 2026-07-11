package logview

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Query struct {
	FileIDs         []string  `json:"fileIds"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	Include         []string  `json:"include"`
	Exclude         []string  `json:"exclude"`
	Regex           bool      `json:"regex"`
	CaseSensitive   bool      `json:"caseSensitive"`
	IncludeUnparsed bool      `json:"includeUnparsed"`
	Levels          []string  `json:"levels"`
	SearchIDs       []string  `json:"searchIds"`
	UserIDs         []string  `json:"userIds"`
	Sources         []string  `json:"sources"`
}

type Progress struct {
	CandidateFiles int   `json:"candidateFiles"`
	ScannedFiles   int   `json:"scannedFiles"`
	TotalBytes     int64 `json:"totalBytes"`
	ScannedBytes   int64 `json:"scannedBytes"`
	Results        int   `json:"results"`
}

type Event struct {
	Type     string    `json:"type"`
	Record   *Record   `json:"record,omitempty"`
	Warning  string    `json:"warning,omitempty"`
	Progress *Progress `json:"progress,omitempty"`
	Reason   string    `json:"reason,omitempty"`
}

type Searcher struct {
	Catalog      *Catalog
	Workers      int
	MaxResults   int
	MaxDuration  time.Duration
	MaxLineBytes int
}

func NewSearcher(catalog *Catalog) *Searcher {
	workers := runtime.GOMAXPROCS(0)
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}
	return &Searcher{
		Catalog:      catalog,
		Workers:      workers,
		MaxResults:   10_000,
		MaxDuration:  30 * time.Second,
		MaxLineBytes: 256 * 1024,
	}
}

func (s *Searcher) Search(ctx context.Context, query Query, emit func(Event) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	matcher, err := newQueryMatcher(query)
	if err != nil {
		return err
	}

	searchCtx := ctx
	cancel := func() {}
	if s.MaxDuration > 0 {
		searchCtx, cancel = context.WithTimeout(ctx, s.MaxDuration)
	} else {
		searchCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	selected := make(map[string]struct{}, len(query.FileIDs))
	for _, id := range query.FileIDs {
		selected[id] = struct{}{}
	}
	candidates := make([]File, 0)
	var totalBytes int64
	for _, catalogFile := range s.Catalog.Files() {
		if len(selected) > 0 {
			if _, ok := selected[catalogFile.ID]; !ok {
				continue
			}
		}
		file, err := s.Catalog.Resolve(catalogFile.ID)
		if err != nil {
			if emitErr := emit(Event{Type: "warning", Warning: err.Error()}); emitErr != nil {
				return emitErr
			}
			continue
		}
		if !query.Start.IsZero() && file.ModTime.Before(query.Start) {
			continue
		}
		candidates = append(candidates, file)
		totalBytes += file.Size
	}
	for id := range selected {
		if _, err := s.Catalog.Resolve(id); err != nil && strings.Contains(err.Error(), "unknown file ID") {
			return err
		}
	}

	progress := Progress{CandidateFiles: len(candidates), TotalBytes: totalBytes}
	if err := emit(Event{Type: "progress", Progress: cloneProgress(progress)}); err != nil {
		return err
	}
	if len(candidates) == 0 {
		return emit(Event{Type: "done", Progress: cloneProgress(progress), Reason: "complete"})
	}

	workerCount := s.Workers
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(candidates) {
		workerCount = len(candidates)
	}
	maxLineBytes := s.MaxLineBytes
	if maxLineBytes < 1 {
		maxLineBytes = 256 * 1024
	}
	maxResults := s.MaxResults
	if maxResults < 1 {
		maxResults = 10_000
	}

	type scanOutput struct {
		record  *Record
		warning string
		done    bool
		bytes   int64
	}
	jobs := make(chan File)
	outputs := make(chan scanOutput, workerCount*2)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for file := range jobs {
				bytesRead, err := scanSearchFile(searchCtx, s.Catalog, file, maxLineBytes, matcher, func(record Record) error {
					copy := record
					select {
					case outputs <- scanOutput{record: &copy}:
						return nil
					case <-searchCtx.Done():
						return searchCtx.Err()
					}
				})
				if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					select {
					case outputs <- scanOutput{warning: fmt.Sprintf("scan %q: %v", file.RelativePath, err)}:
					case <-searchCtx.Done():
					}
				}
				select {
				case outputs <- scanOutput{done: true, bytes: bytesRead}:
				case <-searchCtx.Done():
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, file := range candidates {
			select {
			case jobs <- file:
			case <-searchCtx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(outputs)
	}()

	reason := "complete"
	var callbackErr error
	for output := range outputs {
		if output.record != nil && progress.Results < maxResults {
			progress.Results++
			if err := emit(Event{Type: "result", Record: output.record}); err != nil {
				callbackErr = err
				cancel()
			}
			if progress.Results >= maxResults {
				reason = "limit"
				cancel()
			}
		}
		if output.warning != "" && callbackErr == nil {
			if err := emit(Event{Type: "warning", Warning: output.warning}); err != nil {
				callbackErr = err
				cancel()
			}
		}
		if output.done {
			progress.ScannedFiles++
			progress.ScannedBytes += output.bytes
			if callbackErr == nil && reason == "complete" {
				if err := emit(Event{Type: "progress", Progress: cloneProgress(progress)}); err != nil {
					callbackErr = err
					cancel()
				}
			}
		}
	}
	if callbackErr != nil {
		return callbackErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if reason == "complete" && errors.Is(searchCtx.Err(), context.DeadlineExceeded) {
		reason = "timeout"
	}
	return emit(Event{Type: "done", Progress: cloneProgress(progress), Reason: reason})
}

type queryMatcher struct {
	query          Query
	includeRegexps []*regexp.Regexp
	excludeRegexps []*regexp.Regexp
	includeTerms   []string
	excludeTerms   []string
}

func newQueryMatcher(query Query) (*queryMatcher, error) {
	matcher := &queryMatcher{query: query}
	if query.Regex {
		prefix := "(?i)"
		if query.CaseSensitive {
			prefix = ""
		}
		for _, expression := range query.Include {
			compiled, err := regexp.Compile(prefix + expression)
			if err != nil {
				return nil, fmt.Errorf("invalid include regex %q: %w", expression, err)
			}
			matcher.includeRegexps = append(matcher.includeRegexps, compiled)
		}
		for _, expression := range query.Exclude {
			compiled, err := regexp.Compile(prefix + expression)
			if err != nil {
				return nil, fmt.Errorf("invalid exclude regex %q: %w", expression, err)
			}
			matcher.excludeRegexps = append(matcher.excludeRegexps, compiled)
		}
		return matcher, nil
	}
	matcher.includeTerms = append([]string(nil), query.Include...)
	matcher.excludeTerms = append([]string(nil), query.Exclude...)
	if !query.CaseSensitive {
		for i := range matcher.includeTerms {
			matcher.includeTerms[i] = strings.ToLower(matcher.includeTerms[i])
		}
		for i := range matcher.excludeTerms {
			matcher.excludeTerms[i] = strings.ToLower(matcher.excludeTerms[i])
		}
	}
	return matcher, nil
}

func (m *queryMatcher) matches(record Record) bool {
	if !m.query.IncludeUnparsed && !record.Parsed {
		return false
	}
	if record.Timestamp != nil {
		if !m.query.Start.IsZero() && record.Timestamp.Before(m.query.Start) {
			return false
		}
		if !m.query.End.IsZero() && record.Timestamp.After(m.query.End) {
			return false
		}
	}
	if !matchesField(record.Level, m.query.Levels) ||
		!matchesField(record.SearchID, m.query.SearchIDs) ||
		!matchesField(record.UserID, m.query.UserIDs) ||
		!matchesField(record.Source, m.query.Sources) {
		return false
	}
	if m.query.Regex {
		for _, expression := range m.includeRegexps {
			if !expression.MatchString(record.Raw) {
				return false
			}
		}
		for _, expression := range m.excludeRegexps {
			if expression.MatchString(record.Raw) {
				return false
			}
		}
		return true
	}
	text := record.Raw
	if !m.query.CaseSensitive {
		text = strings.ToLower(text)
	}
	for _, term := range m.includeTerms {
		if !strings.Contains(text, term) {
			return false
		}
	}
	for _, term := range m.excludeTerms {
		if strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func matchesField(value string, accepted []string) bool {
	if len(accepted) == 0 {
		return true
	}
	for _, candidate := range accepted {
		if strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
}

func scanSearchFile(ctx context.Context, catalog *Catalog, file File, maxLineBytes int, matcher *queryMatcher, emit func(Record) error) (int64, error) {
	file, handle, err := catalog.Open(file.ID)
	if err != nil {
		return 0, err
	}
	defer handle.Close()

	reader := bufio.NewReaderSize(handle, 64*1024)
	var offset int64
	var lastTimestamp *time.Time
	for {
		if err := ctx.Err(); err != nil {
			return offset, err
		}
		lineOffset := offset
		line := make([]byte, 0, min(maxLineBytes, 64*1024))
		truncated := false
		for {
			fragment, readErr := reader.ReadSlice('\n')
			offset += int64(len(fragment))
			remaining := maxLineBytes - len(line)
			if remaining > 0 {
				if len(fragment) > remaining {
					line = append(line, fragment[:remaining]...)
					truncated = true
				} else {
					line = append(line, fragment...)
				}
			} else if len(fragment) > 0 {
				truncated = true
			}

			if readErr == nil || errors.Is(readErr, io.EOF) {
				lineText := strings.TrimSuffix(string(line), "\n")
				lineText = strings.TrimSuffix(lineText, "\r")
				if len(line) > 0 || len(fragment) > 0 {
					record := ParseLine(file.ID, file.RelativePath, lineOffset, []byte(lineText))
					record.Truncated = truncated
					if record.Timestamp != nil {
						timestamp := *record.Timestamp
						lastTimestamp = &timestamp
					} else if !record.Parsed && lastTimestamp != nil {
						timestamp := *lastTimestamp
						record.Timestamp = &timestamp
					}
					if matcher.matches(record) {
						if err := emit(record); err != nil {
							return offset, err
						}
					}
				}
				if errors.Is(readErr, io.EOF) {
					return offset, nil
				}
				break
			}
			if !errors.Is(readErr, bufio.ErrBufferFull) {
				return offset, readErr
			}
			truncated = true
			if err := ctx.Err(); err != nil {
				return offset, err
			}
		}
	}
}

func cloneProgress(progress Progress) *Progress {
	copy := progress
	return &copy
}
