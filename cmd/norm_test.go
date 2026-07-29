package cmd

import (
	"bytes"
	"errors"
	"testing"
)

func TestNormalizeCodeConvertsFullWidthAndCJKPunctuation(t *testing.T) {
	input := "if\uff08a\uff1d\uff11\uff0cb\uff1d\u201c\u4f60\u597d\u3001\u4e16\u754c\u3002\u201d\uff09\u3000\u3010ok\u3011 \u300aT\u300b"
	want := "if(a=1,b=\"\u4f60\u597d,\u4e16\u754c.\") [ok] <T>"

	if got := normalizeCode(input); got != want {
		t.Fatalf("normalizeCode() = %q, want %q", got, want)
	}
}

func TestRunNormReadsNormalizesAndWritesClipboard(t *testing.T) {
	var stdout bytes.Buffer
	var copied string
	err := runNorm(&stdout, func() (string, error) {
		return "fmt.Println\uff08\u201chello\u201d\uff09", nil
	}, func(value string) error {
		copied = value
		return nil
	})
	if err != nil {
		t.Fatalf("runNorm() error = %v", err)
	}
	if want := `fmt.Println("hello")`; copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	if want := "\u5df2\u89c4\u8303\u5316\u526a\u8d34\u677f\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunNormReturnsClipboardErrors(t *testing.T) {
	readErr := errors.New("read failed")
	if err := runNorm(&bytes.Buffer{}, func() (string, error) {
		return "", readErr
	}, func(string) error { return nil }); !errors.Is(err, readErr) {
		t.Fatalf("read error = %v, want %v", err, readErr)
	}

	writeErr := errors.New("write failed")
	if err := runNorm(&bytes.Buffer{}, func() (string, error) {
		return "text", nil
	}, func(string) error { return writeErr }); !errors.Is(err, writeErr) {
		t.Fatalf("write error = %v, want %v", err, writeErr)
	}
}
