package cmd

import (
	"fmt"

	"github.com/leo/leo-cli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version.Info())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
