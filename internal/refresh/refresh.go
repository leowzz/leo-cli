package refresh

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/leo/leo-cli/internal/store"
)

type ScanResult struct {
	Repos    []store.Repo
	Warnings []string
}

type ScanFunc func(roots []string, now time.Time) ScanResult

type Options struct {
	Roots        []string
	Now          time.Time
	TTL          time.Duration
	LockPath     string
	MetadataOnly bool
	Scan         ScanFunc
}

type Result struct {
	Ran           bool
	Warnings      []string
	IndexedCount  int
	SkippedReason string
}

func EnsureInitialIndex(ctx context.Context, db *store.Store, roots []string, now time.Time, scan ScanFunc) (Result, error) {
	count, err := db.RepoCount(ctx)
	if err != nil {
		return Result{}, err
	}
	if count > 0 {
		return Result{SkippedReason: "not-empty"}, nil
	}

	result := scan(roots, now)
	if err := db.UpsertRepos(ctx, result.Repos); err != nil {
		return Result{}, err
	}
	return Result{
		Ran:          true,
		Warnings:     result.Warnings,
		IndexedCount: len(result.Repos),
	}, nil
}

func MaybeRefresh(ctx context.Context, db *store.Store, opts Options) (Result, error) {
	if opts.Scan == nil {
		return Result{}, errors.New("refresh scan function is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.TTL <= 0 {
		opts.TTL = time.Hour
	}

	latest, err := db.LatestIndexedAt(ctx)
	if err != nil {
		return Result{}, err
	}
	if !latest.IsZero() && opts.Now.Sub(latest) < opts.TTL {
		return Result{SkippedReason: "fresh"}, nil
	}

	unlock, ok, err := acquireLock(opts.LockPath)
	if err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{SkippedReason: "locked"}, nil
	}
	defer unlock()

	scanResult := opts.Scan(opts.Roots, opts.Now)
	if opts.MetadataOnly {
		err = db.UpdateRepoMetadata(ctx, scanResult.Repos)
	} else {
		err = db.UpsertRepos(ctx, scanResult.Repos)
	}
	if err != nil {
		return Result{}, err
	}

	return Result{
		Ran:          true,
		Warnings:     scanResult.Warnings,
		IndexedCount: len(scanResult.Repos),
	}, nil
}

func acquireLock(path string) (func(), bool, error) {
	if path == "" {
		return func() {}, true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, false, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	file.Close()

	return func() {
		_ = os.Remove(path)
	}, true, nil
}
