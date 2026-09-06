package payload

import "testing"

func TestRepairUnmatchedClosers(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{`{"a":{"n":1}],"after":2}`, `{"a":{"n":1},"after":2}`},
		{`[1,2},3]`, `[1,2,3]`},
		{`{"a":1],"s":"[url] }","after":2}`, `{"a":1,"after":2,"s":"[url] }"}`},
		{`{/* [ */ "a":1], "b":2}`, `{"a":1,"b":2}`},
		{"{// [\n \"a\":1], \"b\":2}", `{"a":1,"b":2}`},
		{`{"s":"/* [ */","a":1],"b":2}`, `{"a":1,"b":2,"s":"/* [ */"}`},
	} {
		got, err := FormatJSON(tt.input, true)
		if err != nil || got != tt.want {
			t.Errorf("input %q: got %q (%v), want %q", tt.input, got, err, tt.want)
		}
	}
}

func TestUnmatchedCloserFallbackLeavesAmbiguousNestingAlone(t *testing.T) {
	input := `[{"a":1],2}`
	if got := removeUnmatchedClosers(input); got != input {
		t.Fatalf("ambiguous nesting changed: %q", got)
	}
}
