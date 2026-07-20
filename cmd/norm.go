package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var normCmd = &cobra.Command{
	Use:   "norm",
	Short: "Normalize clipboard code punctuation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNorm(cmd.OutOrStdout(), clipboard.ReadAll, clipboard.WriteAll)
	},
}

func init() {
	rootCmd.AddCommand(normCmd)
}

func runNorm(stdout io.Writer, readClipboard func() (string, error), writeClipboard func(string) error) error {
	text, err := readClipboard()
	if err != nil {
		return err
	}
	if err := writeClipboard(normalizeCode(text)); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "\u5df2\u89c4\u8303\u5316\u526a\u8d34\u677f")
	return err
}

func normalizeCode(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '\uff01' && r <= '\uff5e':
			return r - 0xfee0
		case r == '\u3000':
			return ' '
		}

		switch r {
		case '\u3001':
			return ','
		case '\u3002':
			return '.'
		case '\u3008', '\u300a':
			return '<'
		case '\u3009', '\u300b':
			return '>'
		case '\u3010', '\u3014':
			return '['
		case '\u3011', '\u3015':
			return ']'
		case '\u2018', '\u2019':
			return '\''
		case '\u201c', '\u201d':
			return '"'
		default:
			return r
		}
	}, text)
}
