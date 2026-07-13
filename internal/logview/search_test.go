package logview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewSearcherUsesConservativeDefaults(t *testing.T) {
	searcher := NewSearcher(nil)
	if searcher.Workers < 1 || searcher.Workers > 2 {
		t.Fatalf("Workers = %d, want 1..2", searcher.Workers)
	}
	if searcher.MaxResults != 500 {
		t.Fatalf("MaxResults = %d, want 500", searcher.MaxResults)
	}
}

func TestSearchDenseShortLinesUsesBoundedAllocation(t *testing.T) {
	catalog := testCatalogWithLine(t, strings.Repeat("plain log line\n", 200_000))
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	_, err := collectSearch(t, NewSearcher(catalog), Query{Include: []string{"never matches"}})
	if err != nil {
		t.Fatal(err)
	}
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated >= 64<<20 {
		t.Fatalf("allocated %d bytes, want less than %d", allocated, 64<<20)
	}
}

func TestSearchUnparsedGroupUsesBoundedAllocation(t *testing.T) {
	catalog := testCatalogWithLine(t, strings.Repeat("plain log line\n", 200_000))
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	_, err := collectSearch(t, NewSearcher(catalog), Query{IncludeUnparsed: true})
	if err != nil {
		t.Fatal(err)
	}
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated >= 64<<20 {
		t.Fatalf("allocated %d bytes, want less than %d", allocated, 64<<20)
	}
}

func TestSearchReturnsRecordsNewestFirst(t *testing.T) {
	contents := "2026-07-13 10:00:00 | INFO | old | user | api - old\n" +
		"2026-07-13 10:04:00 | INFO | middle | user | api - middle\n" +
		"2026-07-13 10:05:00 | INFO | newest | user | api - newest\n"

	events, err := collectSearch(t, NewSearcher(testCatalogWithLine(t, contents)), Query{})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 3 {
		t.Fatalf("records = %#v", records)
	}
	if records[0].SearchID != "newest" || records[1].SearchID != "middle" || records[2].SearchID != "old" {
		t.Fatalf("search IDs = %q, %q, %q", records[0].SearchID, records[1].SearchID, records[2].SearchID)
	}
}

func TestScanReverseLinesPreservesOffsetsAcrossBlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte("alpha\r\nbeta-long\nlast"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var lines []reverseLine
	_, err = scanReverseLines(context.Background(), file, 21, 8, 4, func(line reverseLine) error {
		copy := line
		copy.data = append([]byte(nil), line.data...)
		lines = append(lines, copy)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("lines = %#v", lines)
	}
	if lines[0].offset != 17 || string(lines[0].data) != "last" || lines[0].truncated {
		t.Fatalf("last line = %#v", lines[0])
	}
	if lines[1].offset != 7 || string(lines[1].data) != "beta-lon" || !lines[1].truncated {
		t.Fatalf("middle line = %#v", lines[1])
	}
	if lines[2].offset != 0 || string(lines[2].data) != "alpha" || lines[2].truncated {
		t.Fatalf("first line = %#v", lines[2])
	}
}

func TestScanReverseLinesKeepsEmptyFirstLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte("\nlast"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var lines []reverseLine
	_, err = scanReverseLines(context.Background(), file, 5, 256, 4, func(line reverseLine) error {
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].offset != 1 || string(lines[0].data) != "last" || lines[1].offset != 0 || len(lines[1].data) != 0 {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestSearchStopsReadingAfterTimestampBeforeStart(t *testing.T) {
	old := strings.Repeat("2026-07-13 09:00:00 | INFO | old | user | api - old padding padding padding\n", 3000)
	recent := "2026-07-13 10:04:00 | INFO | recent | user | api - recent\n"
	start := time.Date(2026, 7, 13, 10, 0, 0, 0, time.Local)
	catalog := testCatalogWithLineAt(t, old+recent, start.Add(time.Minute))

	events, err := collectSearch(t, NewSearcher(catalog), Query{Start: start})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 1 || records[0].SearchID != "recent" {
		t.Fatalf("records = %#v", records)
	}
	done := events[len(events)-1]
	if done.Progress == nil || done.Progress.ScannedBytes >= int64(len(old)+len(recent)) {
		t.Fatalf("progress = %#v, want early reverse stop", done.Progress)
	}
}

func TestSearchCutoffKeepsInvalidTimestampRecord(t *testing.T) {
	contents := "2026-07-13 09:00:00 | INFO | old | user | api - old\n" +
		"not-a-time | INFO | invalid | user | api - invalid time\n"
	start := time.Date(2026, 7, 13, 10, 0, 0, 0, time.Local)
	catalog := testCatalogWithLineAt(t, contents, start.Add(time.Minute))

	events, err := collectSearch(t, NewSearcher(catalog), Query{Start: start})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 1 || records[0].SearchID != "invalid" {
		t.Fatalf("records = %#v", records)
	}
}

func TestSearchDoesNotEmitLargeContinuationGroupBeforeOldAnchor(t *testing.T) {
	contents := "2026-07-13 09:00:00 | ERROR | old | user | api - old\n" + strings.Repeat("stack line\n", 501)
	start := time.Date(2026, 7, 13, 10, 0, 0, 0, time.Local)
	catalog := testCatalogWithLineAt(t, contents, start.Add(time.Minute))

	events, err := collectSearch(t, NewSearcher(catalog), Query{Start: start, IncludeUnparsed: true})
	if err != nil {
		t.Fatal(err)
	}
	if records := searchRecords(events); len(records) != 0 {
		t.Fatalf("record count = %d, want 0", len(records))
	}
}

func TestSearchLargeContinuationGroupKeepsOlderMatchingCandidate(t *testing.T) {
	contents := "2026-07-13 10:00:00 | ERROR | anchor | user | api - anchor\n" +
		"needle in first stack line\n" + strings.Repeat("noise\n", 500)

	events, err := collectSearch(t, NewSearcher(testCatalogWithLine(t, contents)), Query{
		Include:         []string{"needle"},
		IncludeUnparsed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 1 || records[0].Raw != "needle in first stack line" || records[0].Timestamp == nil {
		t.Fatalf("records = %#v", records)
	}
}

func TestSearchLargeContinuationGroupKeepsNewestMatches(t *testing.T) {
	var contents strings.Builder
	contents.WriteString("2026-07-13 10:00:00 | ERROR | anchor | user | api - anchor\n")
	for index := 1; index <= 501; index++ {
		fmt.Fprintf(&contents, "stack %03d\n", index)
	}

	events, err := collectSearch(t, NewSearcher(testCatalogWithLine(t, contents.String())), Query{
		Include:         []string{"stack"},
		IncludeUnparsed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 500 {
		t.Fatalf("record count = %d, want 500", len(records))
	}
	if records[0].Raw != "stack 002" || records[499].Raw != "stack 501" {
		t.Fatalf("first/last = %q/%q", records[0].Raw, records[499].Raw)
	}
}

func TestSearchPrioritizesRecentlyModifiedFiles(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "a-old.log")
	newPath := filepath.Join(root, "z-new.log")
	writeTestFile(t, oldPath, []byte("2026-07-13 10:00:00 | INFO | old | user | api - old\n"))
	writeTestFile(t, newPath, []byte("2026-07-13 10:05:00 | INFO | new | user | api - new\n"))
	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, now, now); err != nil {
		t.Fatal(err)
	}
	searcher := NewSearcher(testCatalog(t, root))
	searcher.Workers = 1
	searcher.MaxResults = 1

	events, err := collectSearch(t, searcher, Query{})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 1 || records[0].SearchID != "new" {
		t.Fatalf("records = %#v", records)
	}
}

func TestSearchAppliesTextAndStructuredFilters(t *testing.T) {
	root := t.TempDir()
	lines := []string{
		"2026-07-11 10:10:00.000 | INFO | search-1 | user-1 | api.pay - payment completed",
		"2026-07-11 10:11:00.000 | INFO | search-1 | user-1 | api.pay - payment debug completed",
		"2026-07-11 10:12:00.000 | ERROR | search-1 | user-1 | api.pay - payment completed",
		"2026-07-11 10:13:00.000 | INFO | search-2 | user-1 | api.pay - payment completed",
	}
	writeTestFile(t, filepath.Join(root, "app.log"), []byte(strings.Join(lines, "\n")+"\n"))
	catalog := testCatalog(t, root)

	query := Query{
		Include:   []string{"payment", "completed"},
		Exclude:   []string{"debug"},
		Levels:    []string{"INFO"},
		SearchIDs: []string{"search-1"},
		UserIDs:   []string{"user-1"},
		Sources:   []string{"api.pay"},
	}
	events, err := collectSearch(t, NewSearcher(catalog), query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	records := searchRecords(events)
	if len(records) != 1 || records[0].Message != "payment completed" {
		t.Fatalf("records = %#v", records)
	}
	if events[len(events)-1].Type != "done" || events[len(events)-1].Reason != "complete" {
		t.Fatalf("last event = %#v", events[len(events)-1])
	}
}

func TestSearchHonorsParsedTimeAndKeepsUnparseableTime(t *testing.T) {
	root := t.TempDir()
	contents := "2026-07-11 09:59:59.000 | INFO | old | user | api - old\n" +
		"2026-07-11 10:00:00.000 | INFO | start | user | api - start\n" +
		"not-a-time | INFO | raw | user | api - unknown time\n" +
		"2026-07-11 11:00:00.000 | INFO | end | user | api - end\n" +
		"2026-07-11 11:00:00.001 | INFO | new | user | api - new\n"
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte(contents))
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatal(err)
	}
	catalog := testCatalog(t, root)
	start := time.Date(2026, 7, 11, 10, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 11, 11, 0, 0, 0, time.Local)

	events, err := collectSearch(t, NewSearcher(catalog), Query{Start: start, End: end})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 3 {
		t.Fatalf("records = %#v, want start, unknown, end", records)
	}
	if records[0].SearchID != "end" || records[1].SearchID != "start" || records[2].SearchID != "raw" {
		t.Fatalf("search IDs = %q, %q, %q", records[0].SearchID, records[1].SearchID, records[2].SearchID)
	}
}

func TestSearchInheritsTimestampForUnparsedRecords(t *testing.T) {
	root := t.TempDir()
	contents := "2026-07-11 10:00:00.000 | ERROR | search | user | api - request failed\n" +
		"java.lang.IllegalStateException: failed\n" +
		"2026-07-11 10:02:00.000 | INFO | search | user | api - recovered\n"
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte(contents))
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 11, 9, 59, 0, 0, time.Local)
	end := time.Date(2026, 7, 11, 10, 1, 0, 0, time.Local)

	events, err := collectSearch(t, NewSearcher(testCatalog(t, root)), Query{
		Start: start, End: end, IncludeUnparsed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 2 {
		t.Fatalf("records = %#v, want parsed record and inherited exception", records)
	}
	if records[1].Parsed || records[1].Timestamp == nil || !records[1].Timestamp.Equal(*records[0].Timestamp) {
		t.Fatalf("inherited record = %#v, want unparsed record at %v", records[1], records[0].Timestamp)
	}
}

func TestSearchKeepsLeadingUnparsedRecordTimelessAndIsolatesFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.log"), []byte("2026-07-11 10:00:00.000 | INFO | a | user | api - anchor\n"))
	writeTestFile(t, filepath.Join(root, "b.log"), []byte("leading stack line\n"))

	events, err := collectSearch(t, NewSearcher(testCatalog(t, root)), Query{IncludeUnparsed: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range searchRecords(events) {
		if record.FileName == "b.log" && record.Timestamp != nil {
			t.Fatalf("leading record inherited timestamp across files: %#v", record)
		}
	}
}

func TestSearchCanExcludeUnparsedRecords(t *testing.T) {
	catalog := testCatalogWithLine(t, "2026-07-11 10:00:00.000 | INFO | search | user | api - parsed\nstack trace\n")
	events, err := collectSearch(t, NewSearcher(catalog), Query{IncludeUnparsed: false})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 1 || !records[0].Parsed {
		t.Fatalf("records = %#v, want only parsed record", records)
	}
}

func TestSearchSkipsFilesOlderThanStart(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "old.log")
	writeTestFile(t, path, []byte("old match\n"))
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	catalog := testCatalog(t, root)

	events, err := collectSearch(t, NewSearcher(catalog), Query{Start: time.Now().Add(-time.Hour), Include: []string{"match"}})
	if err != nil {
		t.Fatal(err)
	}
	if records := searchRecords(events); len(records) != 0 {
		t.Fatalf("records = %#v, want none", records)
	}
}

func TestSearchRejectsInvalidRegex(t *testing.T) {
	catalog := testCatalogWithLine(t, "hello")
	_, err := collectSearch(t, NewSearcher(catalog), Query{Regex: true, Include: []string{"["}})
	if err == nil || !strings.Contains(err.Error(), "invalid include regex") {
		t.Fatalf("Search() error = %v", err)
	}
}

func TestSearchStopsAtResultLimit(t *testing.T) {
	catalog := testCatalogWithLine(t, "match\nmatch\nmatch")
	searcher := NewSearcher(catalog)
	searcher.MaxResults = 2

	events, err := collectSearch(t, searcher, Query{Include: []string{"match"}, IncludeUnparsed: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(searchRecords(events)); got != 2 {
		t.Fatalf("record count = %d, want 2", got)
	}
	if events[len(events)-1].Reason != "limit" {
		t.Fatalf("done event = %#v", events[len(events)-1])
	}
}

func TestSearchReturnsPromptlyWhenCancelled(t *testing.T) {
	catalog := testCatalogWithLine(t, strings.Repeat("line\n", 10000))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()

	err := NewSearcher(catalog).Search(ctx, Query{}, func(Event) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Search() error = %v, want context canceled", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("cancellation took %v", elapsed)
	}
}

func TestSearchMarksOversizedLinesTruncated(t *testing.T) {
	catalog := testCatalogWithLine(t, "abcdefghijklmno")
	searcher := NewSearcher(catalog)
	searcher.MaxLineBytes = 8

	events, err := collectSearch(t, searcher, Query{Include: []string{"abc"}, IncludeUnparsed: true})
	if err != nil {
		t.Fatal(err)
	}
	records := searchRecords(events)
	if len(records) != 1 || !records[0].Truncated || records[0].Raw != "abcdefgh" {
		t.Fatalf("records = %#v", records)
	}
}

func TestSearchWarnsWhenOneFileDisappears(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good.log")
	missing := filepath.Join(root, "missing.log")
	writeTestFile(t, good, []byte("match\n"))
	writeTestFile(t, missing, []byte("match\n"))
	catalog := testCatalog(t, root)
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}

	events, err := collectSearch(t, NewSearcher(catalog), Query{Include: []string{"match"}, IncludeUnparsed: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(searchRecords(events)); got != 1 {
		t.Fatalf("record count = %d, want 1", got)
	}
	foundWarning := false
	for _, event := range events {
		foundWarning = foundWarning || event.Type == "warning"
	}
	if !foundWarning {
		t.Fatalf("events = %#v, want warning", events)
	}
}

func testCatalog(t *testing.T, root string) *Catalog {
	t.Helper()
	catalog, warnings, err := BuildCatalog(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	return catalog
}

func testCatalogWithLine(t *testing.T, contents string) *Catalog {
	t.Helper()
	return testCatalogWithLineAt(t, contents, time.Now())
}

func testCatalogWithLineAt(t *testing.T, contents string, modTime time.Time) *Catalog {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte(contents))
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	return testCatalog(t, root)
}

func collectSearch(t *testing.T, searcher *Searcher, query Query) ([]Event, error) {
	t.Helper()
	events := make([]Event, 0)
	err := searcher.Search(context.Background(), query, func(event Event) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

func searchRecords(events []Event) []Record {
	records := make([]Record, 0)
	for _, event := range events {
		if event.Record != nil {
			records = append(records, *event.Record)
		}
	}
	return records
}
