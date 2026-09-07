package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestRunJSONInputSources(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		piped      bool
		copyResult bool
		wantRead   bool
		want       string
	}{
		{"clipboard default", nil, false, false, true, `{"source":"clipboard"}`},
		{"pipe before clipboard", nil, true, false, false, `{"source":"stdin"}`},
		{"argument before pipe", []string{`{source: 'argument'}`}, true, false, false, `{"source":"argument"}`},
		{"copy result", nil, false, true, true, `{"source":"clipboard"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var read, wrote bool
			var copied string
			err := runJSON(tt.args, strings.NewReader(`{source: 'stdin'}`), tt.piped, &stdout, true, tt.copyResult,
				func() (string, error) { read = true; return `{source: 'clipboard'}`, nil },
				func(text string) error { wrote = true; copied = text; return nil }, nil)
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != tt.want+"\n" || read != tt.wantRead || wrote != tt.copyResult {
				t.Fatalf("stdout = %q, read = %t, wrote = %t", stdout.String(), read, wrote)
			}
			if tt.copyResult && copied != tt.want {
				t.Fatalf("copied = %q, want %q", copied, tt.want)
			}
		})
	}
}

func TestRunJSONErrors(t *testing.T) {
	readErr := errors.New("clipboard unavailable")
	writeErr := errors.New("clipboard write failed")
	stdinErr := errors.New("stdin failed")
	stdoutErr := errors.New("stdout failed")
	tests := []struct {
		name    string
		args    []string
		piped   bool
		stdin   io.Reader
		stdout  io.Writer
		read    func() (string, error)
		write   func(string) error
		wantErr error
	}{
		{"clipboard", nil, false, nil, io.Discard, func() (string, error) { return "", readErr }, nil, readErr},
		{"stdin", nil, true, jsonErrorReader{stdinErr}, io.Discard, nil, nil, stdinErr},
		{"copy", []string{`{ok: True}`}, false, nil, io.Discard, nil, func(string) error { return writeErr }, writeErr},
		{"stdout", []string{`{ok: True}`}, false, nil, jsonErrorWriter{stdoutErr}, nil, func(string) error { return nil }, stdoutErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runJSON(tt.args, tt.stdin, tt.piped, tt.stdout, false, true, tt.read, tt.write, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunJSONInvalidInputDoesNotCopy(t *testing.T) {
	for _, input := range []string{
		"", " \n", `{:'value'}`,
		`origin/feat/payload`,
		`&#x20;{ const g=groups.get(e.sha)||[]; g.push(e.path); groups.set(e.sha,g); }`,
		` { const g=groups.get(e.sha)||[]; g.push(e.path); groups.set(e.sha,g); }`,
	} {
		for _, source := range []string{"argument", "stdin", "clipboard"} {
			t.Run(source+"/"+input, func(t *testing.T) {
				var args []string
				if source == "argument" {
					args = []string{input}
				}
				var stdout bytes.Buffer
				err := runJSON(args, strings.NewReader(input), source == "stdin", &stdout, false, true,
					func() (string, error) { return input, nil },
					func(string) error {
						t.Fatal("invalid input copied to clipboard")
						return nil
					},
					func(any) (any, bool, error) {
						t.Fatal("viewer opened for invalid input")
						return nil, false, nil
					})
				if err == nil || err.Error() != "内容里没有 JSON" || stdout.Len() != 0 {
					t.Fatalf("input %q: err = %v, stdout = %q", input, err, stdout.String())
				}
			})
		}
	}
}

func TestJSONCommand(t *testing.T) {
	cmd := newJSONCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--compact", `{'ok': True}`})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if want := "{\"ok\":true}\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if err := cmd.Args(cmd, []string{"one", "two"}); err == nil {
		t.Fatal("expected too many arguments error")
	}
}

func TestRunJSONInspector(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		var stdout bytes.Buffer
		copied := false
		err := runJSON([]string{`prefix {"Content":"{\"ok\":true}"}`}, nil, false, &stdout, true, true, nil,
			func(string) error { copied = true; return nil },
			func(value any) (any, bool, error) {
				if _, ok := value.(map[string]any)["Content"].(string); !ok {
					t.Fatal("viewer received pre-expanded string")
				}
				return map[string]any{"Content": map[string]any{"ok": true}}, accepted, nil
			})
		if err != nil || copied != accepted {
			t.Fatalf("accepted = %t: copied = %t, err = %v", accepted, copied, err)
		}
		want := ""
		if accepted {
			want = "{\"Content\":{\"ok\":true}}\n"
		}
		if stdout.String() != want {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	viewerErr := errors.New("terminal unavailable")
	err := runJSON([]string{`{}`}, nil, false, io.Discard, true, true, nil, nil,
		func(any) (any, bool, error) { return nil, false, viewerErr })
	if !errors.Is(err, viewerErr) {
		t.Fatalf("viewer error = %v", err)
	}
}

type jsonErrorReader struct{ err error }

func (r jsonErrorReader) Read([]byte) (int, error) { return 0, r.err }

type jsonErrorWriter struct{ err error }

func (w jsonErrorWriter) Write([]byte) (int, error) { return 0, w.err }
