package logview

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"sort"
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

var errSearchStartReached = errors.New("search start reached")

const maxReverseGroupRecords = 500

type Searcher struct {
	Catalog      *Catalog
	Workers      int
	MaxResults   int
	MaxDuration  time.Duration
	MaxLineBytes int
}

func NewSearcher(catalog *Catalog) *Searcher {
	workers := runtime.GOMAXPROCS(0)
	if workers > 2 {
		workers = 2
	}
	if workers < 1 {
		workers = 1
	}
	return &Searcher{
		Catalog:      catalog,
		Workers:      workers,
		MaxResults:   500,
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
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].ModTime.After(candidates[j].ModTime)
	})

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
		maxResults = 500
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

	group := make([]Record, 0)
	flush := func(anchored bool) error {
		cutoff := false
		if anchored {
			anchor := group[len(group)-1].Timestamp
			cutoff = !matcher.query.Start.IsZero() && anchor.Before(matcher.query.Start)
			for left, right := 0, len(group)-1; left < right; left, right = left+1, right-1 {
				group[left], group[right] = group[right], group[left]
			}
			var lastTimestamp *time.Time
			for i := range group {
				if group[i].Timestamp != nil {
					timestamp := *group[i].Timestamp
					lastTimestamp = &timestamp
				} else if !group[i].Parsed && lastTimestamp != nil {
					timestamp := *lastTimestamp
					group[i].Timestamp = &timestamp
				}
			}
		}
		for _, record := range group {
			if matcher.matches(record) {
				if err := emit(record); err != nil {
					return err
				}
			}
		}
		group = group[:0]
		if cutoff {
			return errSearchStartReached
		}
		return nil
	}

	bytesRead, err := scanReverseLines(ctx, handle, file.Size, maxLineBytes, 64*1024, func(line reverseLine) error {
		record := ParseLine(file.ID, file.RelativePath, line.offset, line.data)
		record.Truncated = line.truncated
		if record.Timestamp != nil {
			group = append(group, record)
			return flush(true)
		}
		// ponytail: buffer only visible match candidates; add spillable groups if more than 500 matching stack lines matter.
		if len(group) < maxReverseGroupRecords && matcher.matches(record) {
			group = append(group, record)
		}
		return nil
	})
	if errors.Is(err, errSearchStartReached) {
		return bytesRead, nil
	}
	if err != nil {
		return bytesRead, err
	}
	if err := flush(false); err != nil {
		return bytesRead, err
	}
	return bytesRead, nil
}

type reverseLine struct {
	offset    int64
	data      []byte
	truncated bool
}

func scanReverseLines(ctx context.Context, file *os.File, size int64, maxLineBytes, blockBytes int, emit func(reverseLine) error) (int64, error) {
	buffer := make([]byte, blockBytes)
	position := size
	lineEnd := size
	var bytesRead int64
	var firstBlock []byte
	emitLine := func(start, end, blockStart int64, block []byte) error {
		length := end - start
		kept := min(length, int64(maxLineBytes))
		line := make([]byte, int(kept))
		if kept > 0 {
			blockEnd := blockStart + int64(len(block))
			if start >= blockStart && start+kept <= blockEnd {
				copy(line, block[start-blockStart:start-blockStart+kept])
			} else {
				n, err := file.ReadAt(line, start)
				if err != nil && !errors.Is(err, io.EOF) {
					return err
				}
				line = line[:n]
			}
		}
		line = bytes.TrimSuffix(line, []byte{'\r'})
		return emit(reverseLine{offset: start, data: line, truncated: length > int64(maxLineBytes)})
	}

	for position > 0 {
		if err := ctx.Err(); err != nil {
			return bytesRead, err
		}
		start := max(int64(0), position-int64(len(buffer)))
		chunk := buffer[:position-start]
		n, err := file.ReadAt(chunk, start)
		bytesRead += int64(n)
		if err != nil && !errors.Is(err, io.EOF) {
			return bytesRead, err
		}
		chunk = chunk[:n]
		if start == 0 {
			firstBlock = chunk
		}
		for index := len(chunk) - 1; index >= 0; index-- {
			if err := ctx.Err(); err != nil {
				return bytesRead, err
			}
			if chunk[index] != '\n' {
				continue
			}
			lineStart := start + int64(index+1)
			if lineStart != size {
				if err := emitLine(lineStart, lineEnd, start, chunk); err != nil {
					return bytesRead, err
				}
			}
			lineEnd = start + int64(index)
		}
		position = start
	}

	if size > 0 {
		if err := emitLine(0, lineEnd, 0, firstBlock); err != nil {
			return bytesRead, err
		}
	}
	return bytesRead, nil
}

func cloneProgress(progress Progress) *Progress {
	copy := progress
	return &copy
}
