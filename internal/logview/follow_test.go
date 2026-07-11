package logview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFollowStartsNearEndAndStreamsAppends(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.log")
	var contents strings.Builder
	for i := range 100 {
		fmt.Fprintf(&contents, "old-%03d\n", i)
	}
	contents.WriteString("latest\n")
	writeTestFile(t, path, []byte(contents.String()))
	catalog := testCatalog(t, root)
	follower := NewFollower(catalog)
	follower.TailBytes = 24
	follower.PollInterval = 10 * time.Millisecond

	events, cancel, done := startFollow(t, follower, []string{catalog.Files()[0].ID})
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Record != nil && event.Record.Message == "latest"
	})
	appendTestFile(t, path, "appended\n")
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Record != nil && event.Record.Message == "appended"
	})
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Follow() error = %v", err)
	}

	for _, event := range events.snapshot() {
		if event.Record != nil && event.Record.Message == "old-000" {
			t.Fatal("follow read the entire file instead of tailing")
		}
	}
}

func TestFollowReportsTruncationAndContinues(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte("initial line\n"))
	catalog := testCatalog(t, root)
	follower := NewFollower(catalog)
	follower.PollInterval = 10 * time.Millisecond
	events, cancel, done := startFollow(t, follower, []string{catalog.Files()[0].ID})
	defer cancel()

	waitForFollow(t, events, func(event FollowEvent) bool { return event.Record != nil })
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Type == "system" && strings.Contains(event.Message, "truncated")
	})
	appendTestFile(t, path, "after truncate\n")
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Record != nil && event.Record.Message == "after truncate"
	})
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Follow() error = %v", err)
	}
}

func TestFollowHandlesRenameRotation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte("initial\n"))
	catalog := testCatalog(t, root)
	follower := NewFollower(catalog)
	follower.PollInterval = 10 * time.Millisecond
	events, cancel, done := startFollow(t, follower, []string{catalog.Files()[0].ID})
	defer cancel()
	waitForFollow(t, events, func(event FollowEvent) bool { return event.Record != nil })

	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, []byte("new file\n"))
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Type == "system" && strings.Contains(event.Message, "rotated")
	})
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Record != nil && event.Record.Message == "new file"
	})
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Follow() error = %v", err)
	}
}

func TestFollowReportsDeletion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte("initial\n"))
	catalog := testCatalog(t, root)
	follower := NewFollower(catalog)
	follower.PollInterval = 10 * time.Millisecond
	events, cancel, done := startFollow(t, follower, []string{catalog.Files()[0].ID})
	defer cancel()
	waitForFollow(t, events, func(event FollowEvent) bool { return event.Record != nil })

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	waitForFollow(t, events, func(event FollowEvent) bool {
		return event.Type == "system" && strings.Contains(event.Message, "deleted")
	})
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Follow() error = %v", err)
	}
}

func TestConsumeFollowBytesTracksEachLineOffset(t *testing.T) {
	state := &followState{file: File{ID: "file-1", RelativePath: "app.log"}}
	var records []Record
	err := consumeFollowBytes(state, []byte("one\ntwo\n"), 1024, func(event FollowEvent) error {
		records = append(records, *event.Record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].Offset != 0 || records[1].Offset != 4 {
		t.Fatalf("records = %#v, want offsets 0 and 4", records)
	}
}

type followEvents struct {
	mu     sync.Mutex
	events []FollowEvent
}

func (e *followEvents) append(event FollowEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)
}

func (e *followEvents) snapshot() []FollowEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]FollowEvent(nil), e.events...)
}

func startFollow(t *testing.T, follower *Follower, ids []string) (*followEvents, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	events := &followEvents{}
	done := make(chan error, 1)
	go func() {
		done <- follower.Follow(ctx, ids, func(event FollowEvent) error {
			events.append(event)
			return nil
		})
	}()
	return events, cancel, done
}

func waitForFollow(t *testing.T, events *followEvents, match func(FollowEvent) bool) FollowEvent {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, event := range events.snapshot() {
			if match(event) {
				return event
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for follow event; got %#v", events.snapshot())
	return FollowEvent{}
}

func appendTestFile(t *testing.T, path, contents string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(contents); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
