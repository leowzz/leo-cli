package version

import "fmt"

var Value = "dev"
var CommandNameValue = "leo"
var CommitValue = "unknown"

func String() string {
	if Value == "" {
		return "dev"
	}
	return Value
}

func CommandName() string {
	if CommandNameValue == "" {
		return "leo"
	}
	return CommandNameValue
}

func Commit() string {
	if CommitValue == "" {
		return "unknown"
	}
	return CommitValue
}

func Info() string {
	return fmt.Sprintf("version=%s commit=%s", String(), Commit())
}
