package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/atotto/clipboard"
	"github.com/leo/leo-cli/internal/jsonview"
	"github.com/leo/leo-cli/internal/payload"
	"github.com/leo/leo-cli/internal/termio"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newJSONCommand() *cobra.Command {
	var compact, copyResult, plain, interactive bool
	cmd := &cobra.Command{
		Use:   "json [PAYLOAD]",
		Short: "Repair and format JSON, escaped JSON, or Python dicts",
		Long: `Repair JSON-like payloads and print standard JSON.

Input is taken from PAYLOAD, piped stdin, or the clipboard, in that order.
Handles single quotes, Python True/False/None, comments, missing punctuation,
truncated JSON, Markdown code fences, log prefixes, and nested string encoding.
Only top-level encoded payloads are unwrapped; object field strings stay strings.
On a terminal, select JSON strings to parse deeper in an interactive viewer.
Piped output, --plain, and --compact print JSON directly; --interactive forces the viewer.
Repair is heuristic and does not execute Python or JavaScript code.`,
		Example: `  leo json
  leo json "{'name': 'Leo', 'active': True, 'value': None}"
  cat payload.txt | leo json
  leo json --copy
  leo json --compact
  leo json --plain
  leo json --interactive > expanded.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stdin := cmd.InOrStdin()
			var inspect jsonInspector
			if interactive || (!plain && !compact && jsonTerminalOutput(cmd.OutOrStdout())) {
				inspect = runJSONInspector
			}
			return runJSON(args, stdin, hasPipedStdin(stdin), cmd.OutOrStdout(), compact, copyResult, clipboard.ReadAll, clipboard.WriteAll, inspect)
		},
	}
	cmd.Flags().BoolVarP(&compact, "compact", "c", false, "Print JSON on one line")
	cmd.Flags().BoolVar(&copyResult, "copy", false, "Also copy the result to the clipboard")
	cmd.Flags().BoolVar(&plain, "plain", false, "Print JSON without the interactive viewer")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Open the viewer even when stdout is redirected")
	cmd.MarkFlagsMutuallyExclusive("plain", "interactive")
	return cmd
}

func init() {
	rootCmd.AddCommand(newJSONCommand())
}

type jsonInspector func(any) (any, bool, error)

func jsonTerminalOutput(output io.Writer) bool {
	file, ok := output.(*os.File)
	return ok && (isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd()))
}

func runJSONInspector(value any) (any, bool, error) {
	terminal, err := termio.Open()
	if err != nil {
		return nil, false, err
	}
	defer terminal.Close()
	return jsonview.Run(value, terminal.Input, terminal.Output, clipboard.WriteAll)
}

func runJSON(args []string, stdin io.Reader, stdinHasData bool, stdout io.Writer, compact, copyResult bool, readClipboard func() (string, error), writeClipboard func(string) error, inspect jsonInspector) error {
	var text string
	switch {
	case len(args) > 0:
		text = args[0]
	case stdinHasData:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		text = string(data)
	default:
		var err error
		text, err = readClipboard()
		if err != nil {
			return fmt.Errorf("read clipboard: %w", err)
		}
	}

	value, err := payload.ParseJSON(text)
	if err != nil {
		return errors.New("内容里没有 JSON")
	}
	if inspect != nil {
		var accepted bool
		value, accepted, err = inspect(value)
		if err != nil || !accepted {
			return err
		}
	}
	result, err := payload.MarshalJSON(value, compact)
	if err != nil {
		return err
	}
	if copyResult {
		if err := writeClipboard(result); err != nil {
			return fmt.Errorf("write clipboard: %w", err)
		}
	}
	_, err = fmt.Fprintln(stdout, result)
	return err
}
