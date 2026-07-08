package repoui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/leo/leo-cli/internal/store"
)

func TestRepoItemDescriptionShowsBranchLastCommitAndPath(t *testing.T) {
	item := repoItem{repo: store.Repo{
		Path:          "/tmp/project",
		CurrentBranch: "main",
		LastCommitAt:  time.Date(2026, 6, 24, 9, 30, 0, 0, time.Local),
	}}

	got := item.Description()

	for _, want := range []string{"main", "2026-06-24 09:30", "/tmp/project"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Description() = %q, want it to contain %q", got, want)
		}
	}
}

func TestConfigureUIRendererUsesOutputWriter(t *testing.T) {
	prev := lipgloss.DefaultRenderer()
	var buf bytes.Buffer

	restore := configureUIRenderer(&buf)
	if lipgloss.DefaultRenderer() == prev {
		t.Fatal("configureUIRenderer() did not change the default renderer")
	}

	restore()
	if lipgloss.DefaultRenderer() != prev {
		t.Fatal("restore function did not reset the default renderer")
	}
}
