//go:build integration

package logview

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSearchStreamsSparseGigabyteWithBoundedAllocations(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.log")
	writeTestFile(t, path, []byte("match first\n"))
	catalog := testCatalog(t, root)
	if err := os.Truncate(path, 1<<30); err != nil {
		t.Fatal(err)
	}

	searcher := NewSearcher(catalog)
	searcher.MaxDuration = 2 * time.Minute
	firstResult := make(chan struct{}, 1)
	done := make(chan error, 1)
	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	go func() {
		done <- searcher.Search(context.Background(), Query{Include: []string{"match"}, IncludeUnparsed: true}, func(event Event) error {
			if event.Type == "result" {
				select {
				case firstResult <- struct{}{}:
				default:
				}
			}
			return nil
		})
	}()

	select {
	case <-firstResult:
	case <-time.After(2 * time.Second):
		t.Fatal("first result was not streamed promptly")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Minute):
		t.Fatal("gigabyte scan timed out")
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 64<<20 {
		t.Fatalf("scan allocated %d bytes, want at most %d", allocated, 64<<20)
	}
}
