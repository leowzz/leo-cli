package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestParseSQLInTextValuesSplitsWhitespaceAndCommas(t *testing.T) {
	got := parseSQLInTextValues("20317\n20318, 4\t5\n\n6")
	want := []string{"20317", "20318", "4", "5", "6"}

	if !slices.Equal(got, want) {
		t.Fatalf("parseSQLInTextValues() = %#v, want %#v", got, want)
	}
}

func TestJoinCommandUse(t *testing.T) {
	if got, want := joinCmd.Use, "join [FILE]"; got != want {
		t.Fatalf("joinCmd.Use = %q, want %q", got, want)
	}
}

func TestSQLInSourceFromCSVUsesHeaderAndSelectedColumn(t *testing.T) {
	records := [][]string{
		{"id", "name"},
		{"20317", "alpha"},
		{"20318", "beta"},
		{"", "blank"},
	}

	source, err := sqlInSourceFromCSV(records, true)
	if err != nil {
		t.Fatalf("sqlInSourceFromCSV() error = %v", err)
	}

	got := source.values(0)
	want := []string{"20317", "20318"}
	if !slices.Equal(got, want) {
		t.Fatalf("values() = %#v, want %#v", got, want)
	}
	if source.columns[0] != "id" {
		t.Fatalf("column = %q, want id", source.columns[0])
	}
}

func TestRenderSQLInEscapesQuotedValues(t *testing.T) {
	got := renderSQLIn([]string{"20317", "O'Reilly"}, sqlInFormatQuoted, "id")
	want := "'20317','O''Reilly'"

	if got != want {
		t.Fatalf("renderSQLIn() = %q, want %q", got, want)
	}
}

func TestRunSQLInCopiesSelectedCSVColumnAsInClause(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ids.csv")
	if err := os.WriteFile(path, []byte("user_id,name\n20317,alpha\n20317,duplicate\n20318,beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var copied string
	err := runSQLIn(path, &stdout, func(value string) error {
		copied = value
		return nil
	}, func(source sqlInSource) (sqlInSelection, bool, error) {
		return sqlInSelection{column: 0, format: sqlInFormatInClause}, true, nil
	})
	if err != nil {
		t.Fatalf("runSQLIn() error = %v", err)
	}

	want := "user_id in (20317,20318)"
	if copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	wantStdout := "数量: 2\n示例: user_id in (20317,20318)\n已复制到剪贴板\n"
	if stdout.String() != wantStdout {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantStdout)
	}
}

func TestRunSQLInReadsClipboardWhenNoFile(t *testing.T) {
	var stdout bytes.Buffer
	var copied string
	err := runSQLInArgs(nil, strings.NewReader(""), false, &stdout, func() (string, error) {
		return "20317\n20318,20319", nil
	}, func(value string) error {
		copied = value
		return nil
	}, func(source sqlInSource) (sqlInSelection, bool, error) {
		if got, want := source.values(0), []string{"20317", "20318", "20319"}; !slices.Equal(got, want) {
			t.Fatalf("clipboard values = %#v, want %#v", got, want)
		}
		return sqlInSelection{column: 0, format: sqlInFormatParen}, true, nil
	})
	if err != nil {
		t.Fatalf("runSQLInArgs() error = %v", err)
	}

	if want := "(20317,20318,20319)"; copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	if want := "数量: 3\n示例: (20317,20318,20319)\n已复制到剪贴板\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunSQLInReadsPipedStdinWhenNoFile(t *testing.T) {
	var stdout bytes.Buffer
	var copied string
	err := runSQLInArgs(nil, strings.NewReader("20317\n20318\n"), true, &stdout, func() (string, error) {
		return "", fmt.Errorf("clipboard should not be read")
	}, func(value string) error {
		copied = value
		return nil
	}, func(source sqlInSource) (sqlInSelection, bool, error) {
		if got, want := source.values(0), []string{"20317", "20318"}; !slices.Equal(got, want) {
			t.Fatalf("stdin values = %#v, want %#v", got, want)
		}
		return sqlInSelection{column: 0, format: sqlInFormatParen}, true, nil
	})
	if err != nil {
		t.Fatalf("runSQLInArgs() error = %v", err)
	}

	if want := "(20317,20318)"; copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
}

func TestSQLInPickerMovesAcrossColumnsAndFormats(t *testing.T) {
	source := sqlInSource{
		columns: []string{"user_id", "role_id"},
		rows: [][]string{
			{"20317", "4"},
			{"20318", "5"},
		},
	}

	model := newSQLInPickerModel(source)
	if model.list.Title != "SQL IN" {
		t.Fatalf("title = %q, want SQL IN", model.list.Title)
	}
	if got, want := len(model.list.Items()), len(sqlInFormats); got != want {
		t.Fatalf("items = %d, want %d", got, want)
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(sqlInPickerModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(sqlInPickerModel)

	if model.column != 1 {
		t.Fatalf("column = %d, want 1", model.column)
	}
	if got, want := model.result(), "(4,5)"; got != want {
		t.Fatalf("result() = %q, want %q", got, want)
	}
	item, ok := model.list.SelectedItem().(sqlInFormatItem)
	if !ok {
		t.Fatalf("selected item type = %T, want sqlInFormatItem", model.list.SelectedItem())
	}
	if !bytes.Contains([]byte(item.Description()), []byte("(4,5)")) {
		t.Fatalf("selected item description = %q, want preview", item.Description())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(sqlInPickerModel)
	if !model.accepted {
		t.Fatal("accepted = false, want true")
	}
}

func TestSQLInPickerFitsTerminalHeight(t *testing.T) {
	model := newSQLInPickerModel(sqlInSource{
		columns: []string{"user", "cyber", "conversation_mode", "uc_message_count"},
		rows: [][]string{
			{"746139524639227913", "37", "0", "303634"},
			{"786487795047727122", "51", "0", "291503"},
			{"796897623599480856", "25", "0", "289260"},
			{"371486326274392080", "3", "0", "191200"},
		},
	})

	const width, height = 80, 24
	updated, _ := model.Update(tea.WindowSizeMsg{Width: width, Height: height})
	view := updated.(sqlInPickerModel).View()
	got := 0
	for _, line := range strings.Split(strings.TrimSuffix(view, "\n"), "\n") {
		got += maxInt(1, (lipgloss.Width(line)+width-1)/width)
	}
	if got > height {
		t.Fatalf("picker height = %d lines, want at most %d", got, height)
	}
}

func TestSQLInPickerTogglesUniqueAndOriginalValues(t *testing.T) {
	source := sqlInSource{
		columns: []string{"user_id"},
		rows: [][]string{
			{"20317"},
			{"20317"},
			{"20318"},
		},
	}

	model := newSQLInPickerModel(source)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(sqlInPickerModel)

	if got, want := model.result(), "(20317,20318)"; got != want {
		t.Fatalf("default result = %q, want %q", got, want)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	model = updated.(sqlInPickerModel)
	if got, want := model.result(), "(20317,20317,20318)"; got != want {
		t.Fatalf("original result = %q, want %q", got, want)
	}
}
