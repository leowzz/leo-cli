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
		"autoload -Uz compinit",
		"eval \"$(leo completion zsh)\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("zsh script missing %q:\n%s", want, got)
		}
	}
}

func TestBashInitDoesNotIncludeZshCompletion(t *testing.T) {
	got, err := Script("bash")
	if err != nil {
		t.Fatalf("Script() error = %v", err)
	}

	if strings.Contains(got, "compinit") {
		t.Fatalf("bash script contains zsh completion setup:\n%s", got)
	}
}

func TestScriptRejectsUnsupportedShell(t *testing.T) {
	if _, err := Script("fish"); err == nil {
		t.Fatalf("Script(fish) error = nil, want error")
	}
}
