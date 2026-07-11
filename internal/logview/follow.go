package logview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type FollowEvent struct {
	Type    string  `json:"type"`
	FileID  string  `json:"fileId,omitempty"`
	Record  *Record `json:"record,omitempty"`
	Message string  `json:"message,omitempty"`
}

type Follower struct {
	Catalog      *Catalog
	PollInterval time.Duration
	TailBytes    int64
	MaxLineBytes int
}

func NewFollower(catalog *Catalog) *Follower {
	return &Follower{
		Catalog:      catalog,
		PollInterval: 500 * time.Millisecond,
		TailBytes:    64 * 1024,
		MaxLineBytes: 256 * 1024,
	}
}

type followState struct {
	file      File
	handle    *os.File
	identity  os.FileInfo
	offset    int64
	pendingAt int64
	pending   []byte
	truncated bool
	missing   bool
	lastTime  *time.Time
}

func (f *Follower) Follow(ctx context.Context, ids []string, emit func(FollowEvent) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		for _, file := range f.Catalog.Files() {
			ids = append(ids, file.ID)
		}
	}
	maxLineBytes := f.MaxLineBytes
	if maxLineBytes < 1 {
		maxLineBytes = 256 * 1024
	}
	tailBytes := f.TailBytes
	if tailBytes < 1 {
		tailBytes = 64 * 1024
	}

	states := make([]*followState, 0, len(ids))
	for _, id := range ids {
		file, err := f.Catalog.Resolve(id)
		if err != nil {
			return err
		}
		state, err := openFollowState(f.Catalog, file, tailBytes)
		if err != nil {
			return err
		}
		states = append(states, state)
	}
	defer func() {
		for _, state := range states {
			state.handle.Close()
		}
	}()

	pollInterval := f.PollInterval
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		for _, state := range states {
			if err := f.pollState(ctx, state, maxLineBytes, emit); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func openFollowState(catalog *Catalog, file File, tailBytes int64) (*followState, error) {
	file, handle, err := catalog.Open(file.ID)
	if err != nil {
		return nil, err
	}
	info, err := handle.Stat()
	if err != nil {
		handle.Close()
		return nil, err
	}
	offset, err := tailOffset(handle, info.Size(), tailBytes)
	if err != nil {
		handle.Close()
		return nil, err
	}
	return &followState{
		file:      file,
		handle:    handle,
		identity:  info,
		offset:    offset,
		pendingAt: offset,
	}, nil
}

func tailOffset(file *os.File, size, tailBytes int64) (int64, error) {
	if size <= tailBytes {
		return 0, nil
	}
	start := size - tailBytes
	buffer := make([]byte, tailBytes)
	n, err := file.ReadAt(buffer, start)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	if index := strings.IndexByte(string(buffer[:n]), '\n'); index >= 0 {
		return start + int64(index) + 1, nil
	}
	return size, nil
}

func (f *Follower) pollState(ctx context.Context, state *followState, maxLineBytes int, emit func(FollowEvent) error) error {
	pathInfo, statErr := os.Stat(state.file.Path)
	if statErr != nil {
		if err := readFollowAvailable(ctx, state, maxLineBytes, emit); err != nil {
			return err
		}
		if !state.missing {
			state.missing = true
			message := fmt.Sprintf("%s deleted or unavailable: %v", state.file.RelativePath, statErr)
			return emit(FollowEvent{Type: "system", FileID: state.file.ID, Message: message})
		}
		return nil
	}

	if state.missing || !os.SameFile(state.identity, pathInfo) {
		if err := readFollowAvailable(ctx, state, maxLineBytes, emit); err != nil {
			return err
		}
		if err := flushFollowPending(state, emit); err != nil {
			return err
		}
		file, newHandle, err := f.Catalog.Open(state.file.ID)
		if err != nil {
			return emit(FollowEvent{Type: "system", FileID: state.file.ID, Message: err.Error()})
		}
		newInfo, err := newHandle.Stat()
		if err != nil {
			newHandle.Close()
			return err
		}
		state.handle.Close()
		state.handle = newHandle
		state.identity = newInfo
		state.file = file
		state.offset = 0
		state.pendingAt = 0
		state.pending = nil
		state.truncated = false
		state.missing = false
		state.lastTime = nil
		if err := emit(FollowEvent{Type: "system", FileID: state.file.ID, Message: state.file.RelativePath + " rotated; following replacement"}); err != nil {
			return err
		}
	}

	if pathInfo.Size() < state.offset {
		if _, err := state.handle.Seek(0, io.SeekStart); err != nil {
			return err
		}
		state.offset = 0
		state.pendingAt = 0
		state.pending = nil
		state.truncated = false
		state.lastTime = nil
		if err := emit(FollowEvent{Type: "system", FileID: state.file.ID, Message: state.file.RelativePath + " truncated; restarted at beginning"}); err != nil {
			return err
		}
	}
	return readFollowAvailable(ctx, state, maxLineBytes, emit)
}

func readFollowAvailable(ctx context.Context, state *followState, maxLineBytes int, emit func(FollowEvent) error) error {
	info, err := state.handle.Stat()
	if err != nil {
		return err
	}
	remaining := info.Size() - state.offset
	if remaining <= 0 {
		return nil
	}
	if _, err := state.handle.Seek(state.offset, io.SeekStart); err != nil {
		return err
	}
	buffer := make([]byte, min(int64(64*1024), remaining))
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		n, readErr := state.handle.Read(buffer[:readSize])
		if n > 0 {
			if err := consumeFollowBytes(state, buffer[:n], maxLineBytes, emit); err != nil {
				return err
			}
			state.offset += int64(n)
			remaining -= int64(n)
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if n == 0 {
			break
		}
	}
	return nil
}

func consumeFollowBytes(state *followState, data []byte, maxLineBytes int, emit func(FollowEvent) error) error {
	position := state.offset
	for _, value := range data {
		if value == '\n' {
			line := strings.TrimSuffix(string(state.pending), "\r")
			record := ParseLine(state.file.ID, state.file.RelativePath, state.pendingAt, []byte(line))
			record.Truncated = state.truncated
			record = inheritFollowTime(state, record)
			if err := emit(FollowEvent{Type: "record", FileID: state.file.ID, Record: &record}); err != nil {
				return err
			}
			state.pending = state.pending[:0]
			state.truncated = false
			state.pendingAt = position + 1
			position++
			continue
		}
		if len(state.pending) < maxLineBytes {
			state.pending = append(state.pending, value)
		} else {
			state.truncated = true
		}
		position++
	}
	return nil
}

func flushFollowPending(state *followState, emit func(FollowEvent) error) error {
	if len(state.pending) == 0 && !state.truncated {
		return nil
	}
	record := ParseLine(state.file.ID, state.file.RelativePath, state.pendingAt, state.pending)
	record.Truncated = state.truncated
	record = inheritFollowTime(state, record)
	state.pending = nil
	state.truncated = false
	return emit(FollowEvent{Type: "record", FileID: state.file.ID, Record: &record})
}

func inheritFollowTime(state *followState, record Record) Record {
	if record.Timestamp != nil {
		timestamp := *record.Timestamp
		state.lastTime = &timestamp
	} else if !record.Parsed && state.lastTime != nil {
		timestamp := *state.lastTime
		record.Timestamp = &timestamp
	}
	return record
}
