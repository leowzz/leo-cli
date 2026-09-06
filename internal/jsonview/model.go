package jsonview

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leo/leo-cli/internal/payload"
)

type candidate struct {
	path  []string
	label string
	value any
	err   error
}

type revision struct {
	value any
	path  string
}

type model struct {
	value       any
	history     []revision
	candidates  []candidate
	selected    int
	viewport    viewport.Model
	width       int
	height      int
	status      string
	statusError bool
	accepted    bool
	copy        func(string) error
}

func Run(value any, input io.Reader, output io.Writer, copyResult func(string) error) (any, bool, error) {
	previous := lipgloss.DefaultRenderer()
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(output))
	defer lipgloss.SetDefaultRenderer(previous)
	program := tea.NewProgram(newModel(value, copyResult), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen(), tea.WithMouseCellMotion())
	final, err := program.Run()
	if err != nil {
		return nil, false, err
	}
	result, ok := final.(model)
	if !ok {
		return nil, false, fmt.Errorf("unexpected JSON viewer model %T", final)
	}
	return result.value, result.accepted, nil
}

func newModel(value any, copyResult func(string) error) model {
	m := model{value: value, width: 80, height: 24, copy: copyResult}
	m.viewport = viewport.New(80, 1)
	m.viewport.SetHorizontalStep(8)
	m.refresh("")
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.resize()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+d":
			m.accepted = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			return m, tea.Quit
		case "down", "j", "tab":
			m.selectCandidate(1)
			return m, nil
		case "up", "k", "shift+tab":
			m.selectCandidate(-1)
			return m, nil
		case "enter":
			m.expand()
			return m, nil
		case "u", "U", "backspace":
			m.undo()
			return m, nil
		case "c", "C":
			m.copyValue()
			return m, nil
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			layout := m.layout()
			_, actions := m.footer()
			for _, action := range actions {
				if msg.Y == layout.footerY+action.row && msg.X >= action.start && msg.X < action.end {
					return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(action.key)})
				}
			}
			first := m.candidateOffset()
			index := first + msg.Y - layout.candidateY
			if msg.Y >= layout.candidateY && msg.Y < layout.candidateY+layout.candidateRows && index < len(m.candidates) {
				m.selected = index
				m.focus(m.candidates[index].path)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *model) selectCandidate(delta int) {
	if len(m.candidates) == 0 {
		return
	}
	m.selected = (m.selected + delta + len(m.candidates)) % len(m.candidates)
	m.focus(m.candidates[m.selected].path)
}

func (m *model) expand() {
	if len(m.candidates) == 0 {
		m.status, m.statusError = "\u6ca1\u6709\u53ef\u89e3\u6790\u7684 JSON \u5b57\u7b26\u4e32", false
		return
	}
	selected := m.candidates[m.selected]
	if selected.err != nil {
		m.status, m.statusError = "\u89e3\u6790\u5931\u8d25: "+selected.err.Error(), true
		return
	}
	m.history = append(m.history, revision{value: m.value, path: selected.label})
	m.value = replace(m.value, selected.path, selected.value)
	m.status, m.statusError = "\u5df2\u89e3\u6790: "+selected.label, false
	m.refresh(selected.label)
	m.focus(selected.path)
}

func (m *model) undo() {
	if len(m.history) == 0 {
		return
	}
	previous := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.value = previous.value
	m.status, m.statusError = "\u5df2\u64a4\u9500: "+previous.path, false
	m.refresh(previous.path)
	if len(m.candidates) > 0 {
		m.focus(m.candidates[m.selected].path)
	}
}

func (m *model) copyValue() {
	text, err := payload.MarshalJSON(m.value, false)
	if err == nil && m.copy != nil {
		err = m.copy(text)
	}
	if err != nil {
		m.status, m.statusError = "\u590d\u5236\u5931\u8d25: "+err.Error(), true
	} else if m.copy != nil {
		m.status, m.statusError = "\u5df2\u590d\u5236\u5230\u526a\u8d34\u677f", false
	}
}

func (m *model) refresh(preferredPath string) {
	text, err := payload.MarshalJSON(m.value, false)
	if err != nil {
		m.status, m.statusError = err.Error(), true
		return
	}
	m.viewport.SetContent(text)
	m.resize()
	m.candidates = findCandidates(m.value, nil, "$", nil)
	m.selected = min(m.selected, max(0, len(m.candidates)-1))
	for i, candidate := range m.candidates {
		if strings.HasPrefix(candidate.label, preferredPath) {
			m.selected = i
			break
		}
	}
}

func findCandidates(value any, path []string, label string, result []candidate) []candidate {
	switch value := value.(type) {
	case string:
		if parsed, err := payload.InspectStringJSON(value); parsed != nil || err != nil {
			result = append(result, candidate{path: path, label: label, value: parsed, err: err})
		}
	case map[string]any:
		for _, key := range sortedKeys(value) {
			quoted, _ := payload.MarshalJSON(key, true)
			result = findCandidates(value[key], childPath(path, key), label+"["+quoted+"]", result)
		}
	case []any:
		for i, item := range value {
			key := strconv.Itoa(i)
			result = findCandidates(item, childPath(path, key), label+"["+key+"]", result)
		}
	}
	return result
}

func childPath(path []string, key string) []string {
	return append(append([]string(nil), path...), key)
}

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (m *model) focus(path []string) {
	value, line := m.value, 0
	for _, part := range path {
		line++
		switch container := value.(type) {
		case map[string]any:
			for _, key := range sortedKeys(container) {
				if key == part {
					value = container[key]
					break
				}
				text, _ := payload.MarshalJSON(container[key], false)
				line += strings.Count(text, "\n") + 1
			}
		case []any:
			index, _ := strconv.Atoi(part)
			for _, item := range container[:index] {
				text, _ := payload.MarshalJSON(item, false)
				line += strings.Count(text, "\n") + 1
			}
			value = container[index]
		}
	}
	m.viewport.SetXOffset(0)
	m.viewport.SetYOffset(max(0, line-1))
}

// Clone only the ancestors of the replacement so undo snapshots stay intact.
func replace(value any, path []string, replacement any) any {
	if len(path) == 0 {
		return replacement
	}
	switch value := value.(type) {
	case map[string]any:
		copy := make(map[string]any, len(value))
		for key, item := range value {
			copy[key] = item
		}
		copy[path[0]] = replace(value[path[0]], path[1:], replacement)
		return copy
	case []any:
		copy := append([]any(nil), value...)
		index, _ := strconv.Atoi(path[0])
		copy[index] = replace(value[index], path[1:], replacement)
		return copy
	}
	return value
}

func (m model) candidateOffset() int { return max(0, m.selected-m.layout().candidateRows+1) }
