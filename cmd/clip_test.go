package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestClipCommandHasMixFlag(t *testing.T) {
	flag := clipCmd.Flags().Lookup("mix")
	if flag == nil {
		t.Fatal("clip command is missing --mix flag")
	}
	if got, want := flag.Shorthand, "m"; got != want {
		t.Fatalf("mix shorthand = %q, want %q", got, want)
	}
	if got, want := flag.DefValue, "true"; got != want {
		t.Fatalf("mix default = %q, want %q", got, want)
	}
	if flag := clipCmd.Flags().Lookup("hybrid"); flag != nil {
		t.Fatalf("clip command should not expose --hybrid flag")
	}
}

func TestRunClipboardSearchUsesFuzzyModeAndCopiesSelection(t *testing.T) {
	searcher := &fakeClipboardSearcher{result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "selected"}}}}
	var stdout bytes.Buffer
	var copied string
	err := runClipboardSearch(context.Background(), searcher, "sel", true, false, 20, &stdout,
		func(entries []maccy.Entry, _ clipboardPickerOptions) (maccy.Entry, bool, error) {
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

func TestRunClipboardSearchUsesMixMode(t *testing.T) {
	searcher := &fakeClipboardSearcher{result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "selected"}}}}
	err := runClipboardSearch(context.Background(), searcher, "semantic query", false, true, 20, &bytes.Buffer{},
		func(entries []maccy.Entry, options clipboardPickerOptions) (maccy.Entry, bool, error) {
			if options.mode != maccy.SearchHybrid {
				t.Fatalf("picker mode = %q, want %q", options.mode, maccy.SearchHybrid)
			}
			return entries[0], true, nil
		},
		func(string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if searcher.params.Mode != maccy.SearchHybrid {
		t.Fatalf("search mode = %q, want %q", searcher.params.Mode, maccy.SearchHybrid)
	}
}

func TestRunClipboardSearchRejectsConflictingModes(t *testing.T) {
	searcher := &fakeClipboardSearcher{}
	err := runClipboardSearch(context.Background(), searcher, "query", true, true, 20, &bytes.Buffer{}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunClipboardSearchHandlesEmptyAndCancel(t *testing.T) {
	for _, test := range []struct {
		name   string
		result maccy.SearchResponse
		pick   clipboardPicker
		want   string
	}{
		{name: "empty", pick: func(entries []maccy.Entry, _ clipboardPickerOptions) (maccy.Entry, bool, error) {
			if len(entries) != 0 {
				t.Fatalf("entries = %#v, want empty", entries)
			}
			return maccy.Entry{}, false, nil
		}, want: "已取消\n"},
		{name: "cancel", result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "text"}}}, pick: func([]maccy.Entry, clipboardPickerOptions) (maccy.Entry, bool, error) {
			return maccy.Entry{}, false, nil
		}, want: "已取消\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			searcher := &fakeClipboardSearcher{result: test.result}
			var stdout bytes.Buffer
			if err := runClipboardSearch(context.Background(), searcher, "", false, false, 50, &stdout, test.pick, func(string) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != test.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.want)
			}
		})
	}
}

func TestRunClipboardSearchUsesRecentEntriesButKeepsMixDefaultForEmptyQuery(t *testing.T) {
	searcher := &fakeClipboardSearcher{}
	err := runClipboardSearch(context.Background(), searcher, "", false, true, 20, &bytes.Buffer{},
		func(_ []maccy.Entry, options clipboardPickerOptions) (maccy.Entry, bool, error) {
			if options.mode != maccy.SearchHybrid {
				t.Fatalf("picker mode = %q, want %q", options.mode, maccy.SearchHybrid)
			}
			return maccy.Entry{}, false, nil
		},
		func(string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if searcher.params.Mode != maccy.SearchContains {
		t.Fatalf("initial request mode = %q, want %q", searcher.params.Mode, maccy.SearchContains)
	}
}

func TestRunClipboardSearchPropagatesErrors(t *testing.T) {
	wantErr := errors.New("search failed")
	searcher := &fakeClipboardSearcher{err: wantErr}
	if err := runClipboardSearch(context.Background(), searcher, "", false, false, 50, &bytes.Buffer{}, nil, nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestClipboardPickerModelFiltersAndSelectsEntries(t *testing.T) {
	model := newClipboardPickerModel([]maccy.Entry{{PlainText: "first\nline"}, {PlainText: "second"}}, clipboardPickerOptions{})
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

func TestClipboardPickerTogglesFromDefaultMixToContains(t *testing.T) {
	searcher := &fakeClipboardSearcher{result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "normal result"}}}}
	model := newClipboardPickerModel([]maccy.Entry{{PlainText: "hybrid result"}}, clipboardPickerOptions{
		ctx:      context.Background(),
		searcher: searcher,
		query:    "semantic query",
		mode:     maccy.SearchHybrid,
		limit:    20,
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = updated.(clipboardPickerModel)
	if !model.searching || cmd == nil {
		t.Fatalf("toggle did not start search: searching=%v cmd=%v", model.searching, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(clipboardPickerModel)
	if model.mode != maccy.SearchContains || model.searching {
		t.Fatalf("mode/searching = %q/%v", model.mode, model.searching)
	}
	if searcher.params.Mode != maccy.SearchContains || searcher.params.Query != "semantic query" {
		t.Fatalf("search params = %#v", searcher.params)
	}
	if got := model.list.Items()[0].(clipboardEntryItem).entry.PlainText; got != "normal result" {
		t.Fatalf("first result = %q", got)
	}
}

func TestClipboardPickerRunsRemoteSearchFromFilterInput(t *testing.T) {
	searcher := &fakeClipboardSearcher{result: maccy.SearchResponse{Entries: []maccy.Entry{{PlainText: "semantic result"}}}}
	model := newClipboardPickerModel(nil, clipboardPickerOptions{
		ctx:      context.Background(),
		searcher: searcher,
		mode:     maccy.SearchHybrid,
		limit:    20,
	})
	model.list.SetFilterText("semantic query")
	model.list.SetFilterState(list.Filtering)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(clipboardPickerModel)
	if !model.searching || cmd == nil {
		t.Fatalf("remote search did not start: searching=%v cmd=%v", model.searching, cmd)
	}
	updated, _ = model.Update(cmd())
	model = updated.(clipboardPickerModel)
	if searcher.params.Mode != maccy.SearchHybrid || searcher.params.Query != "semantic query" {
		t.Fatalf("search params = %#v", searcher.params)
	}
	if model.query != "semantic query" || model.searching {
		t.Fatalf("query/searching = %q/%v", model.query, model.searching)
	}
}

func TestClipboardPickerTogglesModeWithoutQuery(t *testing.T) {
	model := newClipboardPickerModel(nil, clipboardPickerOptions{})
	if model.mode != maccy.SearchHybrid {
		t.Fatalf("default mode = %q, want %q", model.mode, maccy.SearchHybrid)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = updated.(clipboardPickerModel)
	if cmd != nil || model.mode != maccy.SearchContains {
		t.Fatalf("toggle without query = mode %q, cmd %v", model.mode, cmd)
	}
}

func TestClipboardPickerAllowsMInSearchInput(t *testing.T) {
	searcher := &fakeClipboardSearcher{}
	model := newClipboardPickerModel(nil, clipboardPickerOptions{
		searcher: searcher,
		mode:     maccy.SearchHybrid,
	})
	model.list.SetFilterState(list.Filtering)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = updated.(clipboardPickerModel)
	if model.mode != maccy.SearchHybrid || model.searching {
		t.Fatalf("typing m changed mode/searching: %q/%v", model.mode, model.searching)
	}
	if got := model.list.FilterValue(); got != "m" {
		t.Fatalf("filter value = %q, want m", got)
	}
}

func TestClipboardPickerFitsTerminalHeight(t *testing.T) {
	model := newClipboardPickerModel([]maccy.Entry{{PlainText: "first"}, {PlainText: "second"}}, clipboardPickerOptions{
		query: "semantic query",
		mode:  maccy.SearchHybrid,
	})

	const width, height = 80, 24
	updated, _ := model.Update(tea.WindowSizeMsg{Width: width, Height: height})
	view := updated.(clipboardPickerModel).View()
	lines := 0
	for _, line := range strings.Split(strings.TrimSuffix(view, "\n"), "\n") {
		lines += maxInt(1, (lipgloss.Width(line)+width-1)/width)
	}
	if lines > height {
		t.Fatalf("picker height = %d lines, want at most %d", lines, height)
	}
	if !strings.Contains(view, "模式: 混合") || !strings.Contains(view, "m 切普通/混合") {
		t.Fatalf("picker view missing mode controls:\n%s", view)
	}
}
