package logview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	if records[0].SearchID != "start" || records[1].SearchID != "raw" || records[2].SearchID != "end" {
		t.Fatalf("search IDs = %q, %q, %q", records[0].SearchID, records[1].SearchID, records[2].SearchID)
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

	events, err := collectSearch(t, searcher, Query{Include: []string{"match"}})
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

	events, err := collectSearch(t, searcher, Query{Include: []string{"abc"}})
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

	events, err := collectSearch(t, NewSearcher(catalog), Query{Include: []string{"match"}})
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
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app.log"), []byte(contents))
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
