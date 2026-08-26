package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leo/leo-cli/internal/maccy"
)

type fakeClipboardSearcher struct {
	params maccy.SearchParams
	result maccy.SearchResponse
	err    error
}

func (f *fakeClipboardSearcher) Search(_ context.Context, params maccy.SearchParams) (maccy.SearchResponse, error) {
	f.params = params
	return f.result, f.err
}

func TestClipCommandHasClipboardAlias(t *testing.T) {
	if got, want := clipCmd.Use, "clip [QUERY]"; got != want {
		t.Fatalf("clipCmd.Use = %q, want %q", got, want)
	}
	found := false
	for _, alias := range clipCmd.Aliases {
		if alias == "cb" {
			found = true
		}
	}
	if !found {
		t.Fatalf("clip aliases = %#v, want cb", clipCmd.Aliases)
	}
}

func TestRunClipboardSearchUsesFuzzyModeAndCopiesSelection(t *testing.T) {
	searcher := &fakeClipboardSearcher{result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "selected"}}}}
	var stdout bytes.Buffer
	var copied string
	err := runClipboardSearch(context.Background(), searcher, "sel", true, 20, &stdout,
		func(entries []maccy.Entry) (maccy.Entry, bool, error) {
			if len(entries) != 1 {
				t.Fatalf("entries = %#v", entries)
			}
			return entries[0], true, nil
		}, func(value string) error {
			copied = value
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if searcher.params.Query != "sel" || searcher.params.Mode != maccy.SearchFuzzy || searcher.params.Limit != 20 {
		t.Fatalf("search params = %#v", searcher.params)
	}
	if copied != "selected" || stdout.String() != "已复制到剪贴板\n" {
		t.Fatalf("copied = %q, stdout = %q", copied, stdout.String())
	}
}

func TestRunClipboardSearchHandlesEmptyAndCancel(t *testing.T) {
	for _, test := range []struct {
		name   string
		result maccy.SearchResponse
		pick   clipboardPicker
		want   string
	}{
		{name: "empty", want: "没有找到剪贴板记录\n"},
		{name: "cancel", result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "text"}}}, pick: func([]maccy.Entry) (maccy.Entry, bool, error) { return maccy.Entry{}, false, nil }, want: "已取消\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			searcher := &fakeClipboardSearcher{result: test.result}
			var stdout bytes.Buffer
			pick := test.pick
			if pick == nil {
				pick = func([]maccy.Entry) (maccy.Entry, bool, error) {
					t.Fatal("picker should not be called")
					return maccy.Entry{}, false, nil
				}
			}
			if err := runClipboardSearch(context.Background(), searcher, "", false, 50, &stdout, pick, func(string) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != test.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.want)
			}
		})
	}
}

func TestRunClipboardSearchPropagatesErrors(t *testing.T) {
	wantErr := errors.New("search failed")
	searcher := &fakeClipboardSearcher{err: wantErr}
	if err := runClipboardSearch(context.Background(), searcher, "", false, 50, &bytes.Buffer{}, nil, nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestClipboardPickerModelFiltersAndSelectsEntries(t *testing.T) {
	model := newClipboardPickerModel([]maccy.Entry{{PlainText: "first\nline"}, {PlainText: "second"}})
	if got := model.list.Title; got != "Clipboard History" {
		t.Fatalf("title = %q", got)
	}
	if got := previewClipboardText("first\nline", 120); got != "first line" {
		t.Fatalf("preview = %q", got)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(clipboardPickerModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(clipboardPickerModel)
	if !model.accepted || model.selected.PlainText != "second" {
		t.Fatalf("model = %#v", model)
	}
	if strings.Contains((clipboardEntryItem{entry: maccy.Entry{PlainText: "first\nline"}}).Title(), "\n") {
		t.Fatal("entry title contains a newline")
	}
}
