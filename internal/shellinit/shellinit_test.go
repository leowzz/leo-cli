package shellinit

import (
	"strings"
	"testing"
)

func TestZshInitDefinesRepoFunctionThatCdIntoSelectedPath(t *testing.T) {
	got, err := Script("zsh")
	if err != nil {
		t.Fatalf("Script() error = %v", err)
	}

	for _, want := range []string{
		"repo()",
		"target=\"$(leo repo)\"",
		"cd \"$target\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("zsh script missing %q:\n%s", want, got)
		}
	}
}

func TestScriptRejectsUnsupportedShell(t *testing.T) {
	if _, err := Script("fish"); err == nil {
		t.Fatalf("Script(fish) error = nil, want error")
	}
}
