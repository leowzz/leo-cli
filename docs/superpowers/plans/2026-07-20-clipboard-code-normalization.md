# Clipboard Code Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `leo norm`, which normalizes the entire clipboard from Chinese and full-width punctuation to ASCII and writes it back.

**Architecture:** Keep the character mapping in a pure `normalizeCode(string) string` function implemented with `strings.Map`. A thin Cobra command delegates clipboard I/O to `runNorm`, whose function parameters make the read-normalize-write flow testable without changing the real clipboard.

**Tech Stack:** Go 1.25.6, Cobra, existing `github.com/atotto/clipboard`, Go standard library testing.

## Global Constraints

- The command is exactly `leo norm` and accepts no arguments.
- Normalize the entire clipboard, including strings and comments.
- Add no dependencies, configuration, language parser, or formatter integration.
- Preserve clipboard read and write errors unchanged.

---

### Task 1: Pure punctuation normalization

**Files:**
- Create: `cmd/norm.go`
- Create: `cmd/norm_test.go`

**Interfaces:**
- Consumes: a UTF-8 Go string containing clipboard text.
- Produces: `normalizeCode(text string) string`.

- [ ] **Step 1: Write the failing mapping test**

```go
package cmd

import "testing"

func TestNormalizeCodeConvertsFullWidthAndCJKPunctuation(t *testing.T) {
	input := "if\uff08a\uff1d\uff11\uff0cb\uff1d\u201c\u4f60\u597d\u3001\u4e16\u754c\u3002\u201d\uff09\u3000\u3010ok\u3011 \u300aT\u300b"
	want := "if(a=1,b=\"\u4f60\u597d,\u4e16\u754c.\") [ok] <T>"

	if got := normalizeCode(input); got != want {
		t.Fatalf("normalizeCode() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test ./cmd -run TestNormalizeCodeConvertsFullWidthAndCJKPunctuation -count=1`

Expected: compilation fails with `undefined: normalizeCode`.

- [ ] **Step 3: Implement the minimal mapping**

```go
package cmd

import "strings"

func normalizeCode(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '\uff01' && r <= '\uff5e':
			return r - 0xfee0
		case r == '\u3000':
			return ' '
		}

		switch r {
		case '\u3001':
			return ','
		case '\u3002':
			return '.'
		case '\u3008', '\u300a':
			return '<'
		case '\u3009', '\u300b':
			return '>'
		case '\u3010', '\u3014':
			return '['
		case '\u3011', '\u3015':
			return ']'
		case '\u2018', '\u2019':
			return '\''
		case '\u201c', '\u201d':
			return '"'
		default:
			return r
		}
	}, text)
}
```

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `go test ./cmd -run TestNormalizeCodeConvertsFullWidthAndCJKPunctuation -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the pure mapping**

```bash
git add cmd/norm.go cmd/norm_test.go
git commit -m "feat: normalize full-width code punctuation"
```

### Task 2: Clipboard command flow

**Files:**
- Modify: `cmd/norm.go`
- Modify: `cmd/norm_test.go`

**Interfaces:**
- Consumes: `readClipboard func() (string, error)`, `writeClipboard func(string) error`, and an `io.Writer` for status output.
- Produces: `runNorm(stdout io.Writer, readClipboard func() (string, error), writeClipboard func(string) error) error` and the registered `normCmd` Cobra command.

- [ ] **Step 1: Write failing workflow and error tests**

Append to `cmd/norm_test.go`:

```go
import (
	"bytes"
	"errors"
	"testing"
)

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
```

Keep a single import block when appending the tests.

- [ ] **Step 2: Run the focused workflow tests and verify RED**

Run: `go test ./cmd -run 'TestRunNorm' -count=1`

Expected: compilation fails with `undefined: runNorm`.

- [ ] **Step 3: Add the minimal command and runner**

Update `cmd/norm.go` to include:

```go
import (
	"fmt"
	"io"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var normCmd = &cobra.Command{
	Use:   "norm",
	Short: "Normalize clipboard code punctuation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNorm(cmd.OutOrStdout(), clipboard.ReadAll, clipboard.WriteAll)
	},
}

func init() {
	rootCmd.AddCommand(normCmd)
}

func runNorm(stdout io.Writer, readClipboard func() (string, error), writeClipboard func(string) error) error {
	text, err := readClipboard()
	if err != nil {
		return err
	}
	if err := writeClipboard(normalizeCode(text)); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "\u5df2\u89c4\u8303\u5316\u526a\u8d34\u677f")
	return err
}
```

Keep the `normalizeCode` function from Task 1 unchanged below this code.

- [ ] **Step 4: Run focused and full verification**

Run: `go test ./cmd -run 'TestNormalizeCode|TestRunNorm' -count=1`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.

Run: `git diff --check`

Expected: no output.

- [ ] **Step 5: Commit the command flow**

```bash
git add cmd/norm.go cmd/norm_test.go
git commit -m "feat: add clipboard code normalization command"
```
