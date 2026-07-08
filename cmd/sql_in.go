package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leo/leo-cli/internal/termio"
	"github.com/spf13/cobra"
)

type sqlInFormat int

const (
	sqlInFormatComma sqlInFormat = iota + 1
	sqlInFormatParen
	sqlInFormatInClause
	sqlInFormatQuoted
)

type sqlInSource struct {
	columns []string
	rows    [][]string
}

type sqlInSelection struct {
	column int
	format sqlInFormat
}

type sqlInPicker func(sqlInSource) (sqlInSelection, bool, error)

var joinCmd = &cobra.Command{
	Use:   "join [FILE]",
	Short: "Build a SQL IN value list from file or clipboard",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSQLInArgs(args, cmd.OutOrStdout(), clipboard.ReadAll, clipboard.WriteAll, runSQLInPicker)
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
}

func runSQLInArgs(args []string, stdout io.Writer, readClipboard func() (string, error), writeClipboard func(string) error, pick sqlInPicker) error {
	if len(args) == 0 {
		text, err := readClipboard()
		if err != nil {
			return err
		}
		return runSQLInSource(sqlInSourceFromText(text), stdout, writeClipboard, pick)
	}
	return runSQLIn(args[0], stdout, writeClipboard, pick)
}

func runSQLIn(path string, stdout io.Writer, writeClipboard func(string) error, pick sqlInPicker) error {
	source, err := loadSQLInSource(path)
	if err != nil {
		return err
	}
	return runSQLInSource(source, stdout, writeClipboard, pick)
}

func runSQLInSource(source sqlInSource, stdout io.Writer, writeClipboard func(string) error, pick sqlInPicker) error {
	selection, ok, err := pick(source)
	if err != nil {
		return err
	}
	if !ok {
		_, err := fmt.Fprintln(stdout, "已取消")
		return err
	}

	values := source.values(selection.column)
	if len(values) == 0 {
		return fmt.Errorf("no values found")
	}
	result := renderSQLIn(values, selection.format, sqlInFieldName(source.columns[selection.column]))
	if err := writeClipboard(result); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "数量: %d\n示例: %s\n已复制到剪贴板\n", len(values), previewSQLIn(result, 120))
	return err
}

func loadSQLInSource(path string) (sqlInSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sqlInSource{}, err
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt":
		return sqlInSourceFromText(string(data)), nil
	case ".csv":
		records, err := readSQLInCSV(strings.NewReader(string(data)))
		if err != nil {
			return sqlInSource{}, err
		}
		return sqlInSourceFromCSV(records, guessSQLInCSVHeader(records))
	default:
		return sqlInSource{}, fmt.Errorf("unsupported file type %q; use .txt or .csv", filepath.Ext(path))
	}
}

func sqlInSourceFromText(text string) sqlInSource {
	values := parseSQLInTextValues(text)
	rows := make([][]string, 0, len(values))
	for _, value := range values {
		rows = append(rows, []string{value})
	}
	return sqlInSource{columns: []string{"id"}, rows: rows}
}

func parseSQLInTextValues(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		if value := strings.TrimSpace(field); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func readSQLInCSV(reader io.Reader) ([][]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("csv has no rows")
	}
	return records, nil
}

func sqlInSourceFromCSV(records [][]string, hasHeader bool) (sqlInSource, error) {
	if len(records) == 0 {
		return sqlInSource{}, fmt.Errorf("csv has no rows")
	}

	columnCount := len(records[0])
	if columnCount == 0 {
		return sqlInSource{}, fmt.Errorf("csv has no columns")
	}

	columns := make([]string, columnCount)
	start := 0
	if hasHeader {
		start = 1
		for i := range columns {
			if i < len(records[0]) {
				columns[i] = strings.TrimSpace(records[0][i])
			}
			if columns[i] == "" {
				columns[i] = fmt.Sprintf("column %d", i+1)
			}
		}
	} else {
		for i := range columns {
			columns[i] = fmt.Sprintf("column %d", i+1)
		}
	}

	return sqlInSource{columns: columns, rows: records[start:]}, nil
}

func (s sqlInSource) values(column int) []string {
	values := make([]string, 0, len(s.rows))
	for _, row := range s.rows {
		if column >= len(row) {
			continue
		}
		value := strings.TrimSpace(row[column])
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func guessSQLInCSVHeader(records [][]string) bool {
	if len(records) < 2 {
		return false
	}
	first := records[0]
	second := records[1]
	for i, cell := range first {
		name := strings.ToLower(strings.TrimSpace(cell))
		if name == "id" || strings.HasSuffix(name, "_id") || strings.Contains(name, "id") {
			return true
		}
		if i < len(second) && name != "" && !looksSQLInNumber(name) && looksSQLInNumber(strings.TrimSpace(second[i])) {
			return true
		}
	}
	return false
}

func looksSQLInNumber(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

var sqlInFormats = []sqlInFormat{
	sqlInFormatComma,
	sqlInFormatParen,
	sqlInFormatInClause,
	sqlInFormatQuoted,
}

type sqlInPickerModel struct {
	source   sqlInSource
	column   int
	list     list.Model
	accepted bool
}

type sqlInFormatItem struct {
	format sqlInFormat
	source sqlInSource
	column int
}

func runSQLInPicker(source sqlInSource) (sqlInSelection, bool, error) {
	terminal, err := termio.Open()
	if err != nil {
		return sqlInSelection{}, false, err
	}
	defer terminal.Close()

	restoreRenderer := configureSQLInRenderer(terminal.Output)
	defer restoreRenderer()

	program := tea.NewProgram(newSQLInPickerModel(source), tea.WithInput(terminal.Input), tea.WithOutput(terminal.Output), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return sqlInSelection{}, false, err
	}
	model, ok := finalModel.(sqlInPickerModel)
	if !ok {
		return sqlInSelection{}, false, fmt.Errorf("unexpected UI model type %T", finalModel)
	}
	return sqlInSelection{column: model.column, format: model.selectedFormat()}, model.accepted, nil
}

func newSQLInPickerModel(source sqlInSource) sqlInPickerModel {
	m := sqlInPickerModel{source: source}
	l := list.New(m.formatItems(), list.NewDefaultDelegate(), 0, 0)
	l.Title = "SQL IN"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	m.list = l
	return m
}

func (m sqlInPickerModel) Init() tea.Cmd {
	return nil
}

func (m sqlInPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.accepted = true
			return m, tea.Quit
		case "left", "h":
			return m.moveColumn(-1), nil
		case "right", "l":
			return m.moveColumn(1), nil
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, maxInt(1, msg.Height-6))
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m sqlInPickerModel) View() string {
	l := m.list
	l.SetShowTitle(false)

	var b strings.Builder
	fmt.Fprintln(&b, m.list.Styles.Title.Render(m.list.Title))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "字段: %s\n", m.columnLine())
	fmt.Fprintf(&b, "数量: %d\n", len(m.source.values(m.column)))
	fmt.Fprintf(&b, "预览: %s\n\n", mutedSQLInStyle.Render(previewSQLIn(m.result(), 120)))
	b.WriteString(l.View())
	fmt.Fprintln(&b, "\n←/→ 切字段 · ↑/↓ 切格式 · enter 复制 · esc 取消")
	return b.String()
}

func (m sqlInPickerModel) moveColumn(delta int) sqlInPickerModel {
	if len(m.source.columns) == 0 {
		return m
	}
	m.column = wrapIndex(m.column+delta, len(m.source.columns))
	index := m.list.Index()
	m.list.SetItems(m.formatItems())
	m.list.Select(index)
	return m
}

func (m sqlInPickerModel) columnLine() string {
	parts := make([]string, 0, len(m.source.columns))
	for i, column := range m.source.columns {
		if i == m.column {
			parts = append(parts, selectedSQLInFieldStyle.Render(column))
		} else {
			parts = append(parts, mutedSQLInStyle.Render(column))
		}
	}
	return strings.Join(parts, "  ")
}

func (m sqlInPickerModel) formatItems() []list.Item {
	items := make([]list.Item, 0, len(sqlInFormats))
	for _, format := range sqlInFormats {
		items = append(items, sqlInFormatItem{format: format, source: m.source, column: m.column})
	}
	return items
}

func (m sqlInPickerModel) result() string {
	values := m.source.values(m.column)
	if len(values) == 0 {
		return ""
	}
	return renderSQLIn(values, m.selectedFormat(), sqlInFieldName(m.source.columns[m.column]))
}

func (m sqlInPickerModel) selectedFormat() sqlInFormat {
	item, ok := m.list.SelectedItem().(sqlInFormatItem)
	if !ok {
		return sqlInFormatComma
	}
	return item.format
}

func sqlInFormatLabel(format sqlInFormat) string {
	switch format {
	case sqlInFormatParen:
		return "(1,2,3)"
	case sqlInFormatInClause:
		return "字段 in (1,2,3)"
	case sqlInFormatQuoted:
		return "'1','2','3'"
	default:
		return "1,2,3"
	}
}

func (i sqlInFormatItem) FilterValue() string {
	return i.Title()
}

func (i sqlInFormatItem) Title() string {
	return sqlInFormatLabel(i.format)
}

func (i sqlInFormatItem) Description() string {
	values := i.source.values(i.column)
	if len(values) == 0 {
		return "0 values"
	}
	field := "id"
	if i.column < len(i.source.columns) {
		field = sqlInFieldName(i.source.columns[i.column])
	}
	return previewSQLIn(renderSQLIn(values, i.format, field), 100)
}

func wrapIndex(index, length int) int {
	if length == 0 {
		return 0
	}
	index %= length
	if index < 0 {
		index += length
	}
	return index
}

var (
	mutedSQLInStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedSQLInFieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

func configureSQLInRenderer(output io.Writer) func() {
	prevRenderer := lipgloss.DefaultRenderer()
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(output))
	return func() {
		lipgloss.SetDefaultRenderer(prevRenderer)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func renderSQLIn(values []string, format sqlInFormat, field string) string {
	switch format {
	case sqlInFormatParen:
		return "(" + strings.Join(values, ",") + ")"
	case sqlInFormatInClause:
		return field + " in (" + strings.Join(values, ",") + ")"
	case sqlInFormatQuoted:
		quoted := make([]string, 0, len(values))
		for _, value := range values {
			quoted = append(quoted, "'"+strings.ReplaceAll(value, "'", "''")+"'")
		}
		return strings.Join(quoted, ",")
	default:
		return strings.Join(values, ",")
	}
}

func sqlInFieldName(column string) string {
	if strings.HasPrefix(column, "column ") || strings.TrimSpace(column) == "" {
		return "id"
	}
	return column
}

func previewSQLIn(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}
