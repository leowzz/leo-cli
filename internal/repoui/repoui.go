package repoui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leo/leo-cli/internal/store"
	"github.com/leo/leo-cli/internal/termio"
)

type repoItem struct {
	repo store.Repo
}

func (i repoItem) FilterValue() string {
	return i.repo.Name + " " + i.repo.Path
}

func (i repoItem) Title() string {
	return i.repo.Name
}

func (i repoItem) Description() string {
	branch := i.repo.CurrentBranch
	if branch == "" {
		branch = "-"
	}

	lastCommit := "-"
	if !i.repo.LastCommitAt.IsZero() {
		lastCommit = i.repo.LastCommitAt.Local().Format("2006-01-02 15:04")
	}

	return strings.Join([]string{branch, lastCommit, i.repo.Path}, " | ")
}

type model struct {
	list     list.Model
	selected string
	accepted bool
}

func Run(repos []store.Repo) (string, bool, error) {
	terminal, err := termio.Open()
	if err != nil {
		return "", false, err
	}
	defer terminal.Close()

	return runWithTerminal(repos, terminal.Input, terminal.Output)
}

func runWithTerminal(repos []store.Repo, input io.Reader, output io.Writer) (string, bool, error) {
	restoreRenderer := configureUIRenderer(output)
	defer restoreRenderer()

	items := make([]list.Item, 0, len(repos))
	for _, repo := range repos {
		items = append(items, repoItem{repo: repo})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Repositories"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	m := model{list: l}
	program := tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output))
	finalModel, err := program.Run()
	if err != nil {
		return "", false, err
	}

	if m, ok := finalModel.(model); ok {
		return m.selected, m.accepted, nil
	}
	return "", false, fmt.Errorf("unexpected UI model type %T", finalModel)
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			item, ok := m.list.SelectedItem().(repoItem)
			if ok {
				m.selected = item.repo.Path
				m.accepted = true
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return m.list.View()
}

func configureUIRenderer(output io.Writer) func() {
	prevRenderer := lipgloss.DefaultRenderer()
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(output))
	return func() {
		lipgloss.SetDefaultRenderer(prevRenderer)
	}
}
