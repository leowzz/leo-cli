package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leo/leo-cli/internal/config"
	"github.com/leo/leo-cli/internal/maccy"
	"github.com/leo/leo-cli/internal/termio"
	"github.com/spf13/cobra"
)

var (
	clipFuzzy bool
	clipMix   bool
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
			clipMix && !clipFuzzy,
			clipLimit,
			cfg.Clipboard.SearchInterval,
			cmd.OutOrStdout(),
			runClipboardPicker,
			clipboard.WriteAll,
		)
	},
}

func init() {
	clipCmd.Flags().BoolVar(&clipFuzzy, "fuzzy", false, "Use fuzzy search (requires at least 3 query characters)")
	clipCmd.Flags().BoolVarP(&clipMix, "mix", "m", true, "Use semantic and full-text hybrid search")
	clipCmd.Flags().IntVarP(&clipLimit, "limit", "n", maccy.DefaultLimit, "Maximum number of entries to load")
	clipCmd.MarkFlagsMutuallyExclusive("fuzzy", "mix")
	rootCmd.AddCommand(clipCmd)
}

type clipboardSearcher interface {
	Search(context.Context, maccy.SearchParams) (maccy.SearchResponse, error)
}

type clipboardPickerOptions struct {
	ctx      context.Context
	searcher clipboardSearcher
	query    string
	mode     maccy.SearchMode
	limit    int
	interval time.Duration
}

type clipboardPicker func([]maccy.Entry, clipboardPickerOptions) (maccy.Entry, bool, error)

func runClipboardSearch(
	ctx context.Context,
	searcher clipboardSearcher,
	query string,
	fuzzy bool,
	mix bool,
	limit int,
	searchInterval time.Duration,
	stdout io.Writer,
	pick clipboardPicker,
	writeClipboard func(string) error,
) error {
	if fuzzy && mix {
		return errors.New("--fuzzy and --mix cannot be used together")
	}
	preferredMode := maccy.SearchContains
	if fuzzy {
		preferredMode = maccy.SearchFuzzy
	} else if mix {
		preferredMode = maccy.SearchHybrid
	}
	requestMode := preferredMode
	if requestMode == maccy.SearchHybrid && strings.TrimSpace(query) == "" {
		requestMode = maccy.SearchContains
	}
	result, err := searcher.Search(ctx, maccy.SearchParams{Query: query, Mode: requestMode, Limit: limit})
	if err != nil {
		return err
	}

	selected, ok, err := pick(result.Entries, clipboardPickerOptions{
		ctx:      ctx,
		searcher: searcher,
		query:    query,
		mode:     preferredMode,
		limit:    limit,
		interval: searchInterval,
	})
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
	list        list.Model
	ctx         context.Context
	searcher    clipboardSearcher
	query       string
	mode        maccy.SearchMode
	limit       int
	interval    time.Duration
	revision    uint64
	searching   bool
	searchError string
	selected    maccy.Entry
	accepted    bool
}

type clipboardEntryItem struct {
	entry maccy.Entry
}

type clipboardSearchResultMsg struct {
	entries  []maccy.Entry
	query    string
	mode     maccy.SearchMode
	revision uint64
	err      error
}

type clipboardSearchIntervalMsg struct {
	query    string
	revision uint64
}

func runClipboardPicker(entries []maccy.Entry, options clipboardPickerOptions) (maccy.Entry, bool, error) {
	terminal, err := termio.Open()
	if err != nil {
		return maccy.Entry{}, false, err
	}
	defer terminal.Close()

	restoreRenderer := configureClipboardRenderer(terminal.Output)
	defer restoreRenderer()

	program := tea.NewProgram(
		newClipboardPickerModel(entries, options),
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

func newClipboardPickerModel(entries []maccy.Entry, options clipboardPickerOptions) clipboardPickerModel {
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.mode == "" {
		options.mode = maccy.SearchHybrid
	}
	if options.limit == 0 {
		options.limit = maccy.DefaultLimit
	}
	if options.interval <= 0 {
		options.interval = config.DefaultClipboardSearchInterval
	}

	itemsList := list.New(clipboardEntryItems(entries), list.NewDefaultDelegate(), 0, 0)
	itemsList.Title = "Clipboard History"
	itemsList.SetShowStatusBar(false)
	itemsList.SetFilteringEnabled(true)
	itemsList.Filter = clipboardSearchFilter
	itemsList.Styles.Title = lipgloss.NewStyle().Bold(true)
	return clipboardPickerModel{
		list:     itemsList,
		ctx:      options.ctx,
		searcher: options.searcher,
		query:    options.query,
		mode:     options.mode,
		limit:    options.limit,
		interval: options.interval,
	}
}

func clipboardSearchFilter(term string, targets []string) []list.Rank {
	matches := list.UnsortedFilter(term, targets)
	matchesByIndex := make(map[int][]int, len(matches))
	for _, match := range matches {
		matchesByIndex[match.Index] = match.MatchedIndexes
	}
	ranks := make([]list.Rank, len(targets))
	for i := range targets {
		ranks[i] = list.Rank{Index: i, MatchedIndexes: matchesByIndex[i]}
	}
	return ranks
}

func clipboardEntryItems(entries []maccy.Entry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, clipboardEntryItem{entry: entry})
	}
	return items
}

func (m clipboardPickerModel) Init() tea.Cmd {
	return nil
}

func (m clipboardPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(clipboardSearchResultMsg); ok {
		if result.revision != m.revision {
			return m, nil
		}
		m.searching = false
		if result.err != nil {
			m.searchError = result.err.Error()
			return m, nil
		}
		m.mode = result.mode
		m.query = result.query
		m.searchError = ""
		cmd := m.list.SetItems(clipboardEntryItems(result.entries))
		m.list.ResetSelected()
		return m, cmd
	}
	if interval, ok := msg.(clipboardSearchIntervalMsg); ok {
		query := strings.TrimSpace(m.list.FilterValue())
		if interval.revision != m.revision || interval.query == "" || interval.query != query || !m.list.SettingFilter() {
			return m, nil
		}
		return m.startClipboardSearch(query, m.mode)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "m":
			if m.list.SettingFilter() || m.searching {
				break
			}
			mode := maccy.SearchHybrid
			if m.mode == maccy.SearchHybrid {
				mode = maccy.SearchContains
			}
			if strings.TrimSpace(m.query) == "" {
				m.mode = mode
				m.searchError = ""
				return m, nil
			}
			return m.startClipboardSearch(m.query, mode)
		case "enter":
			if m.list.SettingFilter() {
				query := strings.TrimSpace(m.list.FilterValue())
				if query == "" {
					return m, nil
				}
				m.list.SetFilterState(list.FilterApplied)
				if m.searching {
					return m, nil
				}
				return m.startClipboardSearch(query, m.mode)
			}
			item, ok := m.list.SelectedItem().(clipboardEntryItem)
			if !ok {
				return m, tea.Quit
			}
			m.selected = item.entry
			m.accepted = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, maxInt(1, msg.Height-5))
	}

	wasFiltering := m.list.SettingFilter()
	previousQuery := strings.TrimSpace(m.list.FilterValue())
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if wasFiltering && m.list.SettingFilter() {
		query := strings.TrimSpace(m.list.FilterValue())
		if query != previousQuery {
			m.revision++
			m.searching = false
			if query != "" {
				revision := m.revision
				interval := m.interval
				return m, tea.Batch(cmd, tea.Tick(interval, func(time.Time) tea.Msg {
					return clipboardSearchIntervalMsg{query: query, revision: revision}
				}))
			}
		}
	}
	return m, cmd
}

func (m clipboardPickerModel) startClipboardSearch(query string, mode maccy.SearchMode) (clipboardPickerModel, tea.Cmd) {
	if m.searcher == nil {
		m.searchError = "clipboard search is unavailable"
		return m, nil
	}
	m.revision++
	revision := m.revision
	m.searching = true
	m.searchError = ""
	return m, func() tea.Msg {
		result, err := m.searcher.Search(m.ctx, maccy.SearchParams{Query: query, Mode: mode, Limit: m.limit})
		return clipboardSearchResultMsg{entries: result.Entries, query: query, mode: mode, revision: revision, err: err}
	}
}

func (m clipboardPickerModel) View() string {
	l := m.list
	l.SetShowTitle(false)

	var b strings.Builder
	fmt.Fprintln(&b, m.list.Styles.Title.Render(m.list.Title))
	query := m.query
	if strings.TrimSpace(query) == "" {
		query = "最近记录"
	}
	fmt.Fprintf(&b, "模式: %s · 查询: %s\n", clipboardSearchModeLabel(m.mode), previewClipboardText(query, 80))
	status := ""
	if m.searching {
		status = "检索中..."
	} else if m.searchError != "" {
		status = "检索失败: " + m.searchError
	}
	fmt.Fprintln(&b, status)
	fmt.Fprintln(&b)
	b.WriteString(l.View())
	fmt.Fprintln(&b, "\n↑/↓ 选择 · / 远端搜索 · m 切普通/混合 · enter 复制 · esc 取消")
	return b.String()
}

func clipboardSearchModeLabel(mode maccy.SearchMode) string {
	switch mode {
	case maccy.SearchContains:
		return "普通"
	case maccy.SearchFuzzy:
		return "模糊"
	default:
		return "混合"
	}
}

func (i clipboardEntryItem) FilterValue() string {
	return i.Title()
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
