package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leo/leo-cli/internal/config"
	"github.com/leo/leo-cli/internal/store"
	"github.com/leo/leo-cli/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           version.CommandName(),
	Short:         "Personal command-line tools",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Fprintln(cmd.OutOrStdout(), version.Info())
			return nil
		}
		return cmd.Help()
	},
}

var showVersion bool

func init() {
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Print version")
}

func RootCommand() *cobra.Command {
	return rootCmd
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig() (config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return config.Config{}, err
	}
	if err := config.Ensure(path); err != nil {
		return config.Config{}, err
	}
	return config.Load(path)
}

func openStore() (*store.Store, error) {
	path, err := store.DefaultPath()
	if err != nil {
		return nil, err
	}
	return store.Open(path)
}

func defaultRefreshLockPath() (string, error) {
	dataPath, err := store.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dataPath), "repo-refresh.lock"), nil
}

func commandContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
