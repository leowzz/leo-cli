package termio

import (
	"fmt"
	"os"
	"runtime"
)

type Terminal struct {
	Input      *os.File
	Output     *os.File
	closeInput bool
}

func Open() (*Terminal, error) {
	inputName, outputName := terminalPaths(runtime.GOOS)
	input := os.Stdin
	closeInput := false
	if inputName != "" {
		var err error
		input, err = os.OpenFile(inputName, os.O_RDWR, 0)
		if err != nil {
			return nil, fmt.Errorf("open terminal: %w", err)
		}
		closeInput = true
	}
	if outputName == inputName {
		return &Terminal{Input: input, Output: input, closeInput: closeInput}, nil
	}

	output, err := os.OpenFile(outputName, os.O_RDWR, 0)
	if err != nil {
		if closeInput {
			_ = input.Close()
		}
		return nil, fmt.Errorf("open terminal output: %w", err)
	}
	return &Terminal{Input: input, Output: output, closeInput: closeInput}, nil
}

func (t *Terminal) Close() error {
	if t == nil {
		return nil
	}

	var err error
	if t.Output != nil && t.Output != t.Input {
		err = t.Output.Close()
	}
	if t.closeInput && t.Input != nil {
		if inputErr := t.Input.Close(); err == nil {
			err = inputErr
		}
	}
	return err
}

func terminalPaths(goos string) (string, string) {
	if goos == "windows" {
		// Bubble Tea only uses its Windows console reader for the real stdin fd.
		return "", "CONOUT$"
	}
	return "/dev/tty", "/dev/tty"
}
