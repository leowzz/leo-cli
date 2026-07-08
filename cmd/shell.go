package cmd

import (
	"fmt"

	"github.com/leo/leo-cli/internal/shellinit"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell integration helpers",
}

var shellInitCmd = &cobra.Command{
	Use:   "init SHELL",
	Short: "Print shell integration script",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		script, err := shellinit.Script(args[0])
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), script)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
	shellCmd.AddCommand(shellInitCmd)
}
