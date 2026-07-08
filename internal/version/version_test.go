package version

import "testing"

func TestStringUsesInjectedVersion(t *testing.T) {
	old := Value
	t.Cleanup(func() { Value = old })

	Value = "v1.2.3"
	if got := String(); got != "v1.2.3" {
		t.Fatalf("String() = %q, want v1.2.3", got)
	}
}

func TestStringFallsBackToDev(t *testing.T) {
	old := Value
	t.Cleanup(func() { Value = old })

	Value = ""
	if got := String(); got != "dev" {
		t.Fatalf("String() = %q, want dev", got)
	}
}

func TestCommandNameUsesInjectedValue(t *testing.T) {
	old := CommandNameValue
	t.Cleanup(func() { CommandNameValue = old })

	CommandNameValue = "custom"
	if got := CommandName(); got != "custom" {
		t.Fatalf("CommandName() = %q, want custom", got)
	}
}

func TestCommandNameFallsBackToLeo(t *testing.T) {
	old := CommandNameValue
	t.Cleanup(func() { CommandNameValue = old })

	CommandNameValue = ""
	if got := CommandName(); got != "leo" {
		t.Fatalf("CommandName() = %q, want leo", got)
	}
}

func TestInfoIncludesVersionAndCommit(t *testing.T) {
	oldValue := Value
	oldCommit := CommitValue
	t.Cleanup(func() {
		Value = oldValue
		CommitValue = oldCommit
	})

	Value = "v1.2.3"
	CommitValue = "abc1234"
	if got := Info(); got != "version=v1.2.3 commit=abc1234" {
		t.Fatalf("Info() = %q, want version=v1.2.3 commit=abc1234", got)
	}
}

func TestInfoFallsBackToUnknownCommit(t *testing.T) {
	oldValue := Value
	oldCommit := CommitValue
	t.Cleanup(func() {
		Value = oldValue
		CommitValue = oldCommit
	})

	Value = "v1.2.3"
	CommitValue = ""
	if got := Info(); got != "version=v1.2.3 commit=unknown" {
		t.Fatalf("Info() = %q, want version=v1.2.3 commit=unknown", got)
	}
}
