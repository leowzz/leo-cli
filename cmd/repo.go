package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/leo/leo-cli/internal/config"
	"github.com/leo/leo-cli/internal/refresh"
	"github.com/leo/leo-cli/internal/repoindex"
	"github.com/leo/leo-cli/internal/repoui"
	"github.com/leo/leo-cli/internal/store"
	"github.com/spf13/cobra"
)

const repoMetadataRefreshTTL = 10 * time.Minute

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Browse indexed repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := commandContext(cmd)
		db, err := openStore()
		if err != nil {
			return err
		}
		defer db.Close()

		roots, err := configuredRepoRoots()
		if err != nil {
			return err
		}
		initialResult, err := ensureInitialRepoIndex(ctx, db, roots, time.Now())
		if err != nil {
			return err
		}
		for _, warning := range initialResult.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
		}

		repos, err := db.ListRepos(ctx)
		if err != nil {
			return err
		}
		if len(repos) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No repositories indexed. Run `leo-cli repo reindex` first.")
			return nil
		}
		startRepoMetadataRefresh()

		selected, ok, err := repoui.Run(repos)
		if err != nil {
			return err
		}
		if ok {
			fmt.Fprintln(cmd.OutOrStdout(), selected)
		}
		return nil
	},
}

var repoReindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Update the local repository index",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := commandContext(cmd)
		roots, err := configuredRepoRoots()
		if err != nil {
			return err
		}

		result, err := fullRepoReindex(ctx, roots, time.Now())
		if err != nil {
			return err
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
		}
		if len(result.Repos) == 0 && len(result.Warnings) >= len(roots) {
			return fmt.Errorf("no usable repository roots configured")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Indexed %d repositories.\n", len(result.Repos))
		return nil
	},
}

var repoRefreshMetadataCmd = &cobra.Command{
	Use:    "refresh-metadata",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := commandContext(cmd)
		lockPath, err := defaultRefreshLockPath()
		if err != nil {
			return err
		}

		db, err := openStore()
		if err != nil {
			return err
		}
		defer db.Close()

		repos, err := db.ListRepos(ctx)
		if err != nil {
			return err
		}

		_, err = refresh.MaybeRefresh(ctx, db, refresh.Options{
			Now:          time.Now(),
			TTL:          repoMetadataRefreshTTL,
			LockPath:     lockPath,
			MetadataOnly: true,
			Scan: func(_ []string, now time.Time) refresh.ScanResult {
				result := repoindex.RefreshRepos(repos, now)
				return refresh.ScanResult{Repos: result.Repos, Warnings: result.Warnings}
			},
		})
		return err
	},
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoReindexCmd)
	repoCmd.AddCommand(repoRefreshMetadataCmd)
}

func configuredRepoRoots() ([]string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return config.ExpandedRepoRoots(cfg)
}

func ensureInitialRepoIndex(ctx context.Context, db *store.Store, roots []string, now time.Time) (refresh.Result, error) {
	return refresh.EnsureInitialIndex(ctx, db, roots, now, func(roots []string, now time.Time) refresh.ScanResult {
		result := repoindex.ScanRoots(roots, now)
		return refresh.ScanResult{Repos: result.Repos, Warnings: result.Warnings}
	})
}

func fullRepoReindex(ctx context.Context, roots []string, now time.Time) (repoindex.ScanResult, error) {
	result := repoindex.ScanRoots(roots, now)
	db, err := openStore()
	if err != nil {
		return repoindex.ScanResult{}, err
	}
	defer db.Close()

	if err := db.UpsertRepos(ctx, result.Repos); err != nil {
		return repoindex.ScanResult{}, err
	}
	return result, nil
}

func startRepoMetadataRefresh() {
	executable, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.Command(executable, "repo", "refresh-metadata")
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Start()
}
