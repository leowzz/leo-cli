package logview

import (
	"testing"
	"time"
)

func TestParseLineParsesDemo01LoguruRecord(t *testing.T) {
	line := []byte("2026-07-11 14:23:45.123 | INFO | search-123 | user-456 | app.api:handle:42 - request completed - cached")

	got := ParseLine("file-1", "api/error.log", 99, line)

	if !got.Parsed {
		t.Fatal("Parsed = false, want true")
	}
	if got.FileID != "file-1" || got.FileName != "api/error.log" || got.Offset != 99 {
		t.Fatalf("location fields = %#v", got)
	}
	if got.Level != "INFO" || got.SearchID != "search-123" || got.UserID != "user-456" {
		t.Fatalf("structured fields = %#v", got)
	}
	if got.Source != "app.api:handle:42" || got.Message != "request completed - cached" {
		t.Fatalf("source/message = %q / %q", got.Source, got.Message)
	}
	wantTime := time.Date(2026, 7, 11, 14, 23, 45, 123000000, time.Local)
	if got.Timestamp == nil || !got.Timestamp.Equal(wantTime) {
		t.Fatalf("timestamp = %v, want %v", got.Timestamp, wantTime)
	}
}

func TestParseLineKeepsStructuredRecordWithInvalidTimestamp(t *testing.T) {
	got := ParseLine("file-1", "app.log", 0, []byte("not-a-time | ERROR |  | user-1 | worker - failed"))

	if !got.Parsed || got.Timestamp != nil || got.TimeText != "not-a-time" {
		t.Fatalf("ParseLine() = %#v", got)
	}
	if got.SearchID != "" || got.UserID != "user-1" || got.Message != "failed" {
		t.Fatalf("structured fields = %#v", got)
	}
}

func TestParseLineFallsBackToRawRecord(t *testing.T) {
	got := ParseLine("file-1", "plain.log", 7, []byte("plain log message"))

	if got.Parsed || got.Timestamp != nil {
		t.Fatalf("ParseLine() = %#v, want raw record", got)
	}
	if got.Raw != "plain log message" || got.Message != "plain log message" {
		t.Fatalf("raw/message = %q / %q", got.Raw, got.Message)
	}
}

func TestParseLineReplacesInvalidUTF8(t *testing.T) {
	got := ParseLine("file-1", "plain.log", 0, []byte{'o', 'k', 0xff})
	if got.Raw != "ok\ufffd" || got.Message != "ok\ufffd" {
		t.Fatalf("ParseLine() raw = %q", got.Raw)
	}
}
