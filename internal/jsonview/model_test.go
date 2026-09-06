package jsonview

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/leo/leo-cli/internal/payload"
)

func TestViewerSelectiveExpansionAndUndo(t *testing.T) {
	input := `{"Content":"{\"text\":\"hello\"}","Ext":{"mc:ext_json":"{\"commands\":[{\"payload\":\"{\\\"id\\\":900000000000000002}\"}]}"},"id":900000000000000001,"numeric":"123"}`
	value, err := payload.ParseJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(value, nil)
	if len(m.candidates) != 2 || m.candidates[0].label != `$["Content"]` || m.candidates[1].label != `$["Ext"]["mc:ext_json"]` {
		t.Fatalf("candidates = %#v", m.candidates)
	}
	original, _ := payload.MarshalJSON(m.value, true)
	m = press(m, "enter")
	obj := m.value.(map[string]any)
	if obj["Content"].(map[string]any)["text"] != "hello" {
		t.Fatal("Content not expanded")
	}
	if _, ok := obj["Ext"].(map[string]any)["mc:ext_json"].(string); !ok {
		t.Fatal("unselected Ext changed")
	}
	if obj["id"] != json.Number("900000000000000001") || obj["numeric"] != "123" {
		t.Fatal("unselected scalar types changed")
	}
	m = press(m, "enter")
	if len(m.candidates) != 1 || m.candidates[0].label != `$["Ext"]["mc:ext_json"]["commands"][0]["payload"]` {
		t.Fatalf("deeper candidates = %#v", m.candidates)
	}
	m = press(m, "enter")
	if len(m.candidates) != 0 || len(m.history) != 3 {
		t.Fatal("deeper expansion failed")
	}
	for range 3 {
		m = press(m, "u")
	}
	restored, _ := payload.MarshalJSON(m.value, true)
	if restored != original {
		t.Fatalf("undo changed original: %s", restored)
	}
}

func TestViewerPathsAndCopy(t *testing.T) {
	value := map[string]any{"a.b/c": []any{map[string]any{"": `{"ok":true}`}}}
	var copied string
	m := newModel(value, func(text string) error { copied = text; return nil })
	if m.candidates[0].label != `$["a.b/c"][0][""]` {
		t.Fatal("ambiguous field path")
	}
	m = press(m, "enter")
	m = press(m, "c")
	if !json.Valid([]byte(copied)) || strings.Contains(copied, `\"ok\"`) || m.status != "\u5df2\u590d\u5236\u5230\u526a\u8d34\u677f" {
		t.Fatalf("copy = %q, status = %q", copied, m.status)
	}
	if value["a.b/c"].([]any)[0].(map[string]any)[""] != `{"ok":true}` {
		t.Fatal("original value was mutated")
	}
	m.copy = func(string) error { return errors.New("unavailable") }
	m = press(m, "c")
	if !strings.Contains(m.status, "unavailable") {
		t.Fatal("copy error not displayed")
	}
}

func TestViewerNavigationAndMouse(t *testing.T) {
	m := newModel([]any{`{"first":1}`, `{"second":2}`}, nil)
	m = press(m, "down")
	if m.selected != 1 {
		t.Fatal("selection did not move")
	}
	m = press(m, "enter")
	if _, ok := m.value.([]any)[0].(string); !ok {
		t.Fatal("wrong array element expanded")
	}
	m = clickShortcut(t, m, "enter \u89e3\u6790")
	if len(m.history) != 2 {
		t.Fatal("Parse action did not expand selected string")
	}
	if !clickShortcut(t, m, "q \u5b8c\u6210").accepted {
		t.Fatal("Done action did not accept")
	}
	if press(newModel(nil, nil), "esc").accepted {
		t.Fatal("cancel accepted changes")
	}
}

func TestViewerFitsTerminal(t *testing.T) {
	value := map[string]any{strings.Repeat("long key ", 20): `{"value":"` + strings.Repeat("\u4e2d\u6587", 200) + `"}`}
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}, {20, 8}} {
		m := newModel(value, nil)
		updated, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = updated.(model)
		for range 2 {
			lines := strings.Split(m.View(), "\n")
			if len(lines) > size[1] {
				t.Fatalf("height = %d exceeds %d", len(lines), size[1])
			}
			for _, line := range lines {
				if lipgloss.Width(line) > size[0] {
					t.Fatalf("line exceeds width %d: %q", size[0], line)
				}
			}
			m = press(m, "enter")
		}
	}
}

func TestViewerListsUnrepairableJSONAndShowsError(t *testing.T) {
	broken := `{"text":"\uZZZZ"}`
	m := newModel(map[string]any{"Ext": map[string]any{"mc:ext_json": broken}, "plain": "hello"}, nil)
	if len(m.candidates) != 1 || m.candidates[0].label != `$["Ext"]["mc:ext_json"]` {
		t.Fatalf("broken JSON was hidden: %#v", m.candidates)
	}
	m = press(m, "enter")
	if len(m.history) != 0 || !strings.Contains(m.status, "\u89e3\u6790\u5931\u8d25:") || !m.statusError {
		t.Fatalf("history = %d, status = %q", len(m.history), m.status)
	}
	if m.value.(map[string]any)["Ext"].(map[string]any)["mc:ext_json"] != broken {
		t.Fatal("failed parse replaced original string")
	}
}

func TestViewerExpandsMalformedMessageAndCanUndo(t *testing.T) {
	broken := `{"message_commands":{"cmd":"pay_success","payload":{"image":"[https://example.invalid/1.png"}}],"show_status":2}`
	m := newModel(map[string]any{"Ext": map[string]any{"mc:ext_json": broken}}, nil)
	if len(m.candidates) != 1 {
		t.Fatalf("malformed message was hidden: %#v", m.candidates)
	}
	m = press(m, "enter")
	ext, ok := m.value.(map[string]any)["Ext"].(map[string]any)["mc:ext_json"].(map[string]any)
	if !ok || ext["show_status"] != json.Number("2") || len(m.history) != 1 {
		t.Fatal("message expansion failed or lost following fields")
	}
	m = press(m, "u")
	if m.value.(map[string]any)["Ext"].(map[string]any)["mc:ext_json"] != broken {
		t.Fatal("undo did not restore original malformed string")
	}
}

func press(m model, key string) model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(model)
}

func TestViewerShowsUsableShortcutsAcrossTerminalSizes(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}, {20, 8}} {
		var copied string
		m := newModel(map[string]any{"Content": `{"ok":true}`}, func(text string) error { copied = text; return nil })
		updated, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = updated.(model)
		view := ansi.Strip(m.View())
		for _, hint := range []string{"\u2191/\u2193 \u9009\u62e9", "enter \u89e3\u6790", "c \u590d\u5236", "u \u64a4\u9500", "q \u5b8c\u6210", "esc \u53d6\u6d88"} {
			if !strings.Contains(view, hint) {
				t.Fatalf("size %v: missing %q in %q", size, hint, view)
			}
		}
		if size[0] >= 80 && !strings.Contains(view, `$["Content"]`) {
			t.Fatal("short field path was truncated")
		}
		m = clickShortcut(t, m, "c \u590d\u5236")
		if !json.Valid([]byte(copied)) || m.status != "\u5df2\u590d\u5236\u5230\u526a\u8d34\u677f" {
			t.Fatalf("size %v: copy shortcut failed", size)
		}
		copied = ""
		m = press(m, "C")
		if !json.Valid([]byte(copied)) {
			t.Fatal("uppercase C did not copy")
		}
		m = clickShortcut(t, m, "enter \u89e3\u6790")
		if len(m.history) != 1 {
			t.Fatalf("size %v: parse shortcut failed", size)
		}
		m = clickShortcut(t, m, "u \u64a4\u9500")
		if len(m.history) != 0 {
			t.Fatalf("size %v: undo shortcut failed", size)
		}
	}
}

func clickShortcut(t *testing.T, m model, label string) model {
	t.Helper()
	for y, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if index := strings.Index(line, label); index >= 0 {
			x := lipgloss.Width(line[:index])
			updated, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
			return updated.(model)
		}
	}
	t.Fatalf("shortcut %q is not visible", label)
	return m
}
