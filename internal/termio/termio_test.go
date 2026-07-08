package termio

import "testing"

func TestTerminalPathsUseNativeConsoleDevices(t *testing.T) {
	tests := []struct {
		goos       string
		wantInput  string
		wantOutput string
	}{
		{goos: "darwin", wantInput: "/dev/tty", wantOutput: "/dev/tty"},
		{goos: "linux", wantInput: "/dev/tty", wantOutput: "/dev/tty"},
		{goos: "windows", wantInput: "CONIN$", wantOutput: "CONOUT$"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			gotInput, gotOutput := terminalPaths(tt.goos)
			if gotInput != tt.wantInput || gotOutput != tt.wantOutput {
				t.Fatalf("terminalPaths(%q) = %q, %q; want %q, %q", tt.goos, gotInput, gotOutput, tt.wantInput, tt.wantOutput)
			}
		})
	}
}
