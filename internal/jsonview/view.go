package jsonview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	keyStyle      = lipgloss.NewStyle().Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("160"))
)

type shortcut struct{ key, label, action string }

var primaryShortcuts = []shortcut{
	{"\u2191/\u2193", "\u9009\u62e9", "down"},
	{"enter", "\u89e3\u6790", "enter"},
	{"c", "\u590d\u5236", "c"},
	{"u", "\u64a4\u9500", "u"},
	{"q", "\u5b8c\u6210", "q"},
	{"esc", "\u53d6\u6d88", "esc"},
}

var scrollShortcuts = []shortcut{
	{"\u2190/\u2192", "\u6a2a\u5411\u6eda\u52a8", ""},
	{"pgup/pgdn", "\u7ffb\u9875", ""},
	{"\u6eda\u8f6e", "\u6eda\u52a8", ""},
}

type shortcutHit struct {
	start, end, row int
	key             string
}

// Use rendered columns for both wrapping and mouse hit targets.
func (m model) footer() ([]string, []shortcutHit) {
	groups := [][]shortcut{primaryShortcuts}
	if m.height >= 16 {
		groups = append(groups, scrollShortcuts)
	}
	var lines []string
	var hits []shortcutHit
	for _, group := range groups {
		line, columns := "", 0
		for _, item := range group {
			text := keyStyle.Render(item.key) + " " + mutedStyle.Render(item.label)
			width := lipgloss.Width(text)
			if columns > 0 && columns+3+width > m.width {
				lines = append(lines, line)
				line, columns = "", 0
			}
			if columns > 0 {
				line += mutedStyle.Render(" \u00b7 ")
				columns += 3
			}
			if item.action != "" {
				hits = append(hits, shortcutHit{start: columns, end: min(m.width, columns+width), row: len(lines), key: item.action})
			}
			line += text
			columns += width
		}
		lines = append(lines, line)
	}
	return lines, hits
}

type viewLayout struct {
	compact                                           bool
	previewHeight, candidateY, candidateRows, footerY int
}

func (m model) layout() viewLayout {
	footer, _ := m.footer()
	compact := m.height < 16
	header, fieldHeader, spacer := 3, 1, 1
	if compact {
		header, fieldHeader, spacer = 1, 0, 0
	}
	fixed := header + fieldHeader + spacer + 1 + len(footer)
	candidateRows := min(3, max(1, m.height-fixed-1))
	previewHeight := max(1, m.height-fixed-candidateRows)
	candidateY := header + previewHeight + fieldHeader
	return viewLayout{
		compact: compact, previewHeight: previewHeight,
		candidateY: candidateY, candidateRows: candidateRows,
		footerY: candidateY + candidateRows + 1 + spacer,
	}
}

func (m *model) resize() {
	m.viewport.Width = m.width
	m.viewport.Height = m.layout().previewHeight
	m.viewport.SetYOffset(m.viewport.YOffset)
}

func (m model) View() string {
	layout := m.layout()
	lines := []string{keyStyle.Render("JSON")}
	if !layout.compact {
		lines = append(lines, "", fmt.Sprintf("\u5df2\u89e3\u6790: %d \u00b7 JSON \u5b57\u7b26\u4e32: %d", len(m.history), len(m.candidates)))
	}
	lines = append(lines, m.viewport.View())
	if !layout.compact {
		index := 0
		if len(m.candidates) > 0 {
			index = m.selected + 1
		}
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("\u5b57\u6bb5: %d/%d", index, len(m.candidates))))
	}
	first := m.candidateOffset()
	for i := first; i < first+layout.candidateRows; i++ {
		line := ""
		if i < len(m.candidates) {
			label := m.candidates[i].label
			if m.candidates[i].err != nil {
				label += " [invalid]"
			}
			available := max(1, m.width-2)
			if width := lipgloss.Width(label); width > available {
				label = ansi.TruncateLeft(label, width-available+3, "...")
			}
			line = "  " + mutedStyle.Render(label)
			if i == m.selected {
				line = selectedStyle.Render("\u2502 " + label)
			}
		} else if i == 0 {
			line = mutedStyle.Render("  \u65e0 JSON \u5b57\u7b26\u4e32")
		}
		lines = append(lines, line)
	}
	statusStyle := mutedStyle
	if m.statusError {
		statusStyle = errorStyle
	}
	lines = append(lines, statusStyle.Render(m.status))
	if !layout.compact {
		lines = append(lines, "")
	}
	footer, _ := m.footer()
	lines = append(lines, footer...)
	var fitted []string
	for _, line := range strings.Split(strings.Join(lines, "\n"), "\n") {
		fitted = append(fitted, ansi.Truncate(line, m.width, "..."))
	}
	return strings.Join(fitted[:min(len(fitted), m.height)], "\n")
}
