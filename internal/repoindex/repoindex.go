package repoindex

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leo/leo-cli/internal/store"
)

type ScanResult struct {
	Repos    []store.Repo
	Warnings []string
}

func RefreshRepos(repos []store.Repo, now time.Time) ScanResult {
	result := ScanResult{Repos: make([]store.Repo, 0, len(repos))}
	for _, repo := range repos {
		gitPath := filepath.Join(repo.Path, ".git")
		if !isGitRepoMarker(repo.Path, gitPath) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: not a git repository", repo.Path))
			continue
		}

		gitDir := resolveGitDir(repo.Path, gitPath)
		branch, lastCommitAt := gitMetadata(repo.Path, gitDir)
		repo.CurrentBranch = branch
		repo.LastCommitAt = lastCommitAt
		repo.LastGitActivityAt = lastGitActivity(repo.Path, gitPath)
		repo.LastIndexedAt = now
		result.Repos = append(result.Repos, repo)
	}
	return result
}

func ScanRoots(roots []string, now time.Time) ScanResult {
	result := ScanResult{}
	seen := map[string]struct{}{}

	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", root, err))
			continue
		}
		if !info.IsDir() {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: not a directory", root))
			continue
		}

		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", path, walkErr))
				return filepath.SkipDir
			}
			if !entry.IsDir() {
				return nil
			}
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}

			gitPath := filepath.Join(path, ".git")
			if !isGitRepoMarker(path, gitPath) {
				return nil
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", path, err))
				return filepath.SkipDir
			}
			absPath = filepath.Clean(absPath)
			if _, ok := seen[absPath]; ok {
				return filepath.SkipDir
			}
			seen[absPath] = struct{}{}

			gitDir := resolveGitDir(path, gitPath)
			branch, lastCommitAt := gitMetadata(path, gitDir)
			result.Repos = append(result.Repos, store.Repo{
				Path:              absPath,
				Name:              filepath.Base(absPath),
				CurrentBranch:     branch,
				LastCommitAt:      lastCommitAt,
				LastGitActivityAt: lastGitActivity(path, gitPath),
				LastIndexedAt:     now,
			})
			return nil
		})
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", root, err))
		}
	}

	return result
}

func lastGitActivity(repoPath, gitPath string) time.Time {
	gitDir := resolveGitDir(repoPath, gitPath)
	if timestamp, ok := lastHeadLogTimestamp(filepath.Join(gitDir, "logs", "HEAD")); ok {
		return timestamp
	}
	if info, err := os.Stat(gitDir); err == nil {
		return info.ModTime()
	}
	if info, err := os.Stat(gitPath); err == nil {
		return info.ModTime()
	}
	if info, err := os.Stat(repoPath); err == nil {
		return info.ModTime()
	}
	return time.Unix(0, 0)
}

func resolveGitDir(repoPath, gitPath string) string {
	info, err := os.Stat(gitPath)
	if err == nil && info.IsDir() {
		return gitPath
	}

	gitDir, ok := parseGitFileGitDir(gitPath)
	if !ok {
		return gitPath
	}
	if filepath.IsAbs(gitDir) {
		return filepath.Clean(gitDir)
	}
	return filepath.Clean(filepath.Join(repoPath, gitDir))
}

func isGitRepoMarker(repoPath, gitPath string) bool {
	info, err := os.Lstat(gitPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// leave permission errors etc. to caller if needed
		}
		return false
	}
	if info.IsDir() {
		return true
	}
	if !info.Mode().IsRegular() {
		return false
	}

	gitDir, ok := parseGitFileGitDir(gitPath)
	if !ok {
		return false
	}
	if filepath.IsAbs(gitDir) {
		gitDir = filepath.Clean(gitDir)
	} else {
		gitDir = filepath.Clean(filepath.Join(repoPath, gitDir))
	}
	dirInfo, err := os.Stat(gitDir)
	return err == nil && dirInfo.IsDir()
}

func parseGitFileGitDir(gitPath string) (string, bool) {
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", false
	}
	line := strings.TrimSpace(string(data))
	gitDir, ok := strings.CutPrefix(line, "gitdir:")
	if !ok {
		return "", false
	}
	gitDir = strings.TrimSpace(gitDir)
	if gitDir == "" {
		return "", false
	}
	return gitDir, true
}

func lastHeadLogTimestamp(path string) (time.Time, bool) {
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer file.Close()

	var last string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			last = text
		}
	}
	if last == "" {
		return time.Time{}, false
	}

	fields := strings.Fields(last)
	if len(fields) < 5 {
		return time.Time{}, false
	}
	seconds, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0), true
}

func gitMetadata(repoPath, gitDir string) (string, time.Time) {
	refName, commitID := headRefAndCommit(gitDir)
	if commitID == "" && refName != "" {
		commitID = commitIDForRef(gitDir, refName)
	}

	branch := branchName(refName, commitID)
	if commitID == "" {
		return branch, gitLastCommitTime(repoPath)
	}
	lastCommitAt := commitTime(gitDir, commitID)
	if lastCommitAt.IsZero() {
		lastCommitAt = gitLastCommitTime(repoPath)
	}
	return branch, lastCommitAt
}

func headRefAndCommit(gitDir string) (string, string) {
	data, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", ""
	}
	line := strings.TrimSpace(string(data))
	if ref, ok := strings.CutPrefix(line, "ref:"); ok {
		return strings.TrimSpace(ref), ""
	}
	if isHexObjectID(line) {
		return "", line
	}
	return "", ""
}

func commitIDForRef(gitDir, refName string) string {
	if refName == "" {
		return ""
	}

	if data, err := os.ReadFile(filepath.Join(gitDir, filepath.FromSlash(refName))); err == nil {
		commitID := strings.TrimSpace(string(data))
		if isHexObjectID(commitID) {
			return commitID
		}
	}

	file, err := os.Open(filepath.Join(gitDir, "packed-refs"))
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == refName && isHexObjectID(fields[0]) {
			return fields[0]
		}
	}
	return ""
}

func branchName(refName, commitID string) string {
	if branch, ok := strings.CutPrefix(refName, "refs/heads/"); ok {
		return branch
	}
	if refName != "" {
		return refName
	}
	if len(commitID) >= 7 {
		return "detached@" + commitID[:7]
	}
	return ""
}

func commitTime(gitDir, commitID string) time.Time {
	if !isHexObjectID(commitID) {
		return time.Time{}
	}

	objectPath := filepath.Join(gitDir, "objects", commitID[:2], commitID[2:])
	file, err := os.Open(objectPath)
	if err != nil {
		return time.Time{}
	}
	defer file.Close()

	reader, err := zlib.NewReader(file)
	if err != nil {
		return time.Time{}
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return time.Time{}
	}

	nul := bytes.IndexByte(raw, 0)
	if nul < 0 {
		return time.Time{}
	}
	body := raw[nul+1:]
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "committer ") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return time.Time{}
			}
			seconds, err := strconv.ParseInt(fields[len(fields)-2], 10, 64)
			if err != nil {
				return time.Time{}
			}
			return time.Unix(seconds, 0)
		}
	}
	return time.Time{}
}

func isHexObjectID(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func gitLastCommitTime(repoPath string) time.Time {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%ct")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}
