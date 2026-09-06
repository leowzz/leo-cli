package payload

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"standard", `{"name":"Leo","active":true}`, `{"active":true,"name":"Leo"}`},
		{"python dict", `{'active': True, 'deleted': False, 'value': None}`, `{"active":true,"deleted":false,"value":null}`},
		{"nested python", `{'items': [{'id': 1}, {'id': 2,}], 'ok': True}`, `{"items":[{"id":1},{"id":2}],"ok":true}`},
		{"python repr", `'{"name": "Leo", "line": "a\\nb"}'`, `{"line":"a\nb","name":"Leo"}`},
		{"python repr apostrophe", `'{"name": "it\'s fine"}'`, `{"name":"it's fine"}`},
		{"encoded json", `"{\"name\":\"Leo\",\"ok\":true}"`, `{"name":"Leo","ok":true}`},
		{"encoded python", `"{'active': True, 'value': None}"`, `{"active":true,"value":null}`},
		{"bare escaped", `{\"name\":\"Leo\",\"ok\":true}`, `{"name":"Leo","ok":true}`},
		{"bare escaped content", `{\"path\":\"C:\\\\temp\",\"text\":\"a\\nb\",\"quote\":\"say \\\"hi\\\"\"}`, `{"path":"C:\\temp","quote":"say \"hi\"","text":"a\nb"}`},
		{"python escaped strings", `{'text': 'a\nb', 'path': 'C:\\temp', 'name': "it's fine"}`, `{"name":"it's fine","path":"C:\\temp","text":"a\nb"}`},
		{"python hex escapes", `{'text': '\x1b[0m', 'name': '\U0001f600'}`, "{\"name\":\"\U0001f600\",\"text\":\"\\u001b[0m\"}"},
		{"missing syntax", `{name: 'Leo' active: True}`, `{"active":true,"name":"Leo"}`},
		{"truncated", `{"items": [1, 2`, `{"items":[1,2]}`},
		{"comments", "{/* comment */ 'name': 'Leo', // comment\n 'ok': True,}", `{"name":"Leo","ok":true}`},
		{"markdown", "```json\n{'ok': True,}\n```", `{"ok":true}`},
		{"ndjson", "{\"id\":1}\n{\"id\":2}", `[{"id":1},{"id":2}]`},
		{"large numbers", `{id: 9223372036854775808123, n: 1.234567890123456789, e: 1e400}`, `{"e":1e400,"id":9223372036854775808123,"n":1.234567890123456789}`},
		{"unicode", `{'name': '\u4f60\u597d'}`, "{\"name\":\"\u4f60\u597d\"}"},
		{"string contents", `{"path":"C:\\temp\\file","word":"True None False","json":"{\"ok\":true}","html":"<b>&</b>"}`, `{"html":"<b>&</b>","json":"{\"ok\":true}","path":"C:\\temp\\file","word":"True None False"}`},
		{"string", `"hello"`, `"hello"`},
		{"numeric string", `"123"`, `"123"`},
		{"null", `null`, `null`},
		{"number", `42`, `42`},
		{"bom", " \ufeff{\"ok\": true} ", `{"ok":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatJSON(tt.input, true)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFormatJSONMultipleEncodingLayers(t *testing.T) {
	text := `{"id":9223372036854775808123,"text":"a\nb","path":"C:\\temp"}`
	for range 5 {
		encoded, err := json.Marshal(text)
		if err != nil {
			t.Fatal(err)
		}
		text = string(encoded)
	}
	got, err := FormatJSON(text, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":9223372036854775808123,"path":"C:\\temp","text":"a\nb"}`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestFormatJSONPretty(t *testing.T) {
	got, err := FormatJSON(`{'ok': True}`, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"ok\": true\n}"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatJSONInvalid(t *testing.T) {
	for _, input := range []string{"", " \n\t", "\ufeff", `{:'value'}`, `{"text":"\uZZZZ"}`} {
		if got, err := FormatJSON(input, false); err == nil {
			t.Errorf("input %q: got %q without error", input, got)
		}
	}
}

func FuzzFormatJSON(f *testing.F) {
	for _, seed := range []string{`{'ok': True}`, `{\"x\":1}`, `"{\"x\":1}"`, "```json\n[1,2", "", "'\\'"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 4096 {
			t.Skip()
		}
		got, err := FormatJSON(input, false)
		if err == nil && !json.Valid([]byte(got)) {
			t.Fatalf("invalid JSON %q from %q", got, input)
		}
		if err == nil && strings.TrimSpace(got) == "" {
			t.Fatalf("empty output from %q", input)
		}
	})
}
