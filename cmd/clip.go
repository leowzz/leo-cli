package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leo/leo-cli/internal/maccy"
	"github.com/leo/leo-cli/internal/termio"
	"github.com/spf13/cobra"
)

var (
	clipFuzzy bool
	clipLimit int
)

var clipCmd = &cobra.Command{
	Use:     "clip [QUERY]",
	Aliases: []string{"cb"},
	Short:   "Search remote clipboard history",
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := maccy.New(cfg.Clipboard.BaseURL, cfg.Clipboard.Token)
		if err != nil {
			return err
		}
		query := strings.Join(args, " ")
		return runClipboardSearch(
			commandContext(cmd),
			client,
			query,
			clipFuzzy,
			clipLimit,
			cmd.OutOrStdout(),
			runClipboardPicker,
			clipboard.WriteAll,
		)
	},
}

func init() {
	clipCmd.Flags().BoolVar(&clipFuzzy, "fuzzy", false, "Use fuzzy search (requires at least 3 query characters)")
	clipCmd.Flags().IntVarP(&clipLimit, "limit", "n", maccy.DefaultLimit, "Maximum number of entries to load")
	rootCmd.AddCommand(clipCmd)
}

type clipboardSearcher interface {
	Search(context.Context, maccy.SearchParams) (maccy.SearchResponse, error)
}

type clipboardPicker func([]maccy.Entry) (maccy.Entry, bool, error)

func runClipboardSearch(
	ctx context.Context,
	searcher clipboardSearcher,
	query string,
	fuzzy bool,
	limit int,
	stdout io.Writer,
	pick clipboardPicker,
	writeClipboard func(string) error,
) error {
	mode := maccy.SearchContains
	if fuzzy {
		mode = maccy.SearchFuzzy
	}
	result, err := searcher.Search(ctx, maccy.SearchParams{Query: query, Mode: mode, Limit: limit})
	if err != nil {
		return err
	}
	if len(result.Entries) == 0 {
		_, err := fmt.Fprintln(stdout, "没有找到剪贴板记录")
		return err
	}

	selected, ok, err := pick(result.Entries)
	if err != nil {
		return err
	}
	if !ok {
		_, err := fmt.Fprintln(stdout, "已取消")
		return err
	}
	if err := writeClipboard(selected.PlainText); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "已复制到剪贴板")
	return err
}

type clipboardPickerModel struct {
	list     list.Model
	selected maccy.Entry
	accepted bool
}

type clipboardEntryItem struct {
	entry maccy.Entry
}

func runClipboardPicker(entries []maccy.Entry) (maccy.Entry, bool, error) {
	terminal, err := termio.Open()
	if err != nil {
		return maccy.Entry{}, false, err
	}
	defer terminal.Close()

	restoreRenderer := configureClipboardRenderer(terminal.Output)
	defer restoreRenderer()

	program := tea.NewProgram(
		newClipboardPickerModel(entries),
		tea.WithInput(terminal.Input),
		tea.WithOutput(terminal.Output),
		tea.WithAltScreen(),
	)
	finalModel, err := program.Run()
	if err != nil {
		return maccy.Entry{}, false, err
	}
	model, ok := finalModel.(clipboardPickerModel)
	if !ok {
		return maccy.Entry{}, false, fmt.Errorf("unexpected clipboard picker model type %T", finalModel)
	}
	return model.selected, model.accepted, nil
}

func newClipboardPickerModel(entries []maccy.Entry) clipboardPickerModel {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, clipboardEntryItem{entry: entry})
	}
	itemsList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	itemsList.Title = "Clipboard History"
	itemsList.SetShowStatusBar(false)
	itemsList.SetFilteringEnabled(true)
	itemsList.Styles.Title = lipgloss.NewStyle().Bold(true)
	return clipboardPickerModel{list: itemsList}
}

func (m clipboardPickerModel) Init() tea.Cmd {
	return nil
}

func (m clipboardPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			item, ok := m.list.SelectedItem().(clipboardEntryItem)
			if !ok {
				return m, tea.Quit
			}
			m.selected = item.entry
			m.accepted = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, maxInt(1, msg.Height-3))
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m clipboardPickerModel) View() string {
	l := m.list
	l.SetShowTitle(false)

	var b strings.Builder
	fmt.Fprintln(&b, m.list.Styles.Title.Render(m.list.Title))
	fmt.Fprintln(&b)
	b.WriteString(l.View())
	fmt.Fprintln(&b, "\n↑/↓ 选择 · / 搜索 · enter 复制 · esc 取消")
	return b.String()
}

func (i clipboardEntryItem) FilterValue() string {
	return i.entry.PlainText
}

func (i clipboardEntryItem) Title() string {
	return previewClipboardText(i.entry.PlainText, 120)
}

func (i clipboardEntryItem) Description() string {
	when := "unknown time"
	if !i.entry.LastCopiedAt.IsZero() {
		when = i.entry.LastCopiedAt.Local().Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s · %d occurrences", when, i.entry.OccurrenceCount)
}

func previewClipboardText(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func configureClipboardRenderer(output io.Writer) func() {
	prevRenderer := lipgloss.DefaultRenderer()
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(output))
	return func() {
		lipgloss.SetDefaultRenderer(prevRenderer)
	}
}

var _ clipboardSearcher = (*maccy.Client)(nil)
