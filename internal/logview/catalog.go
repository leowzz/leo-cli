package logview

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leo/leo-cli/internal/config"
)

type File struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RelativePath string    `json:"relativePath"`
	Root         string    `json:"root"`
	Path         string    `json:"-"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"modifiedAt"`
}

type Catalog struct {
	roots []string
	files []File
	byID  map[string]File
}

func BuildCatalog(projectRoot string, configured []string) (*Catalog, []string, error) {
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve project root: %w", err)
	}

	catalog := &Catalog{byID: make(map[string]File)}
	warnings := make([]string, 0)
	for _, entry := range configured {
		root, err := resolveConfiguredRoot(projectRoot, entry)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skip log directory %q: %v", entry, err))
			continue
		}
		if containsString(catalog.roots, root) {
			continue
		}
		catalog.roots = append(catalog.roots, root)

		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				warnings = append(warnings, fmt.Sprintf("skip %q: %v", path, walkErr))
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if path == root {
				return nil
			}
			if strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() || !isLogName(entry.Name()) {
				return nil
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
			binary, err := hasBinaryPrefix(path)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("skip %q: %v", path, err))
				return nil
			}
			if binary {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			id := fileID(root, path)
			file := File{
				ID:           id,
				Name:         entry.Name(),
				RelativePath: relative,
				Root:         root,
				Path:         path,
				Size:         info.Size(),
				ModTime:      info.ModTime(),
			}
			catalog.files = append(catalog.files, file)
			catalog.byID[id] = file
			return nil
		})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("scan log directory %q: %v", root, err))
		}
	}

	if len(catalog.roots) == 0 {
		return nil, warnings, errors.New("no usable log directories")
	}
	sort.Slice(catalog.files, func(i, j int) bool {
		if catalog.files[i].Root == catalog.files[j].Root {
			return catalog.files[i].RelativePath < catalog.files[j].RelativePath
		}
		return catalog.files[i].Root < catalog.files[j].Root
	})
	return catalog, warnings, nil
}

func (c *Catalog) Roots() []string {
	return append([]string(nil), c.roots...)
}

func (c *Catalog) Files() []File {
	return append([]File(nil), c.files...)
}

func (c *Catalog) Resolve(id string) (File, error) {
	file, ok := c.byID[id]
	if !ok {
		return File{}, fmt.Errorf("unknown file ID %q", id)
	}
	info, err := os.Lstat(file.Path)
	if err != nil {
		return File{}, fmt.Errorf("revalidate %q: %w", file.RelativePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return File{}, fmt.Errorf("revalidate %q: symlink is not allowed", file.RelativePath)
	}
	if !info.Mode().IsRegular() {
		return File{}, fmt.Errorf("revalidate %q: not a regular file", file.RelativePath)
	}
	canonical, err := filepath.EvalSymlinks(file.Path)
	if err != nil {
		return File{}, fmt.Errorf("revalidate %q: %w", file.RelativePath, err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return File{}, fmt.Errorf("revalidate %q: %w", file.RelativePath, err)
	}
	if !pathWithin(file.Root, canonical) {
		return File{}, fmt.Errorf("revalidate %q: file escaped configured root", file.RelativePath)
	}
	file.Path = canonical
	file.Size = info.Size()
	file.ModTime = info.ModTime()
	return file, nil
}

func (c *Catalog) Open(id string) (File, *os.File, error) {
	file, err := c.Resolve(id)
	if err != nil {
		return File{}, nil, err
	}
	opened, err := os.Open(file.Path)
	if err != nil {
		return File{}, nil, fmt.Errorf("open %q: %w", file.RelativePath, err)
	}
	if err := validateOpenedFile(opened, file); err != nil {
		opened.Close()
		return File{}, nil, err
	}
	info, err := opened.Stat()
	if err != nil {
		opened.Close()
		return File{}, nil, err
	}
	file.Size = info.Size()
	file.ModTime = info.ModTime()
	return file, opened, nil
}

func validateOpenedFile(opened *os.File, file File) error {
	openedInfo, err := opened.Stat()
	if err != nil {
		return fmt.Errorf("validate opened %q: %w", file.RelativePath, err)
	}
	if !openedInfo.Mode().IsRegular() {
		return fmt.Errorf("validate opened %q: not a regular file", file.RelativePath)
	}
	pathInfo, err := os.Lstat(file.Path)
	if err != nil {
		return fmt.Errorf("validate opened %q: %w", file.RelativePath, err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("validate opened %q: symlink is not allowed", file.RelativePath)
	}
	if !pathInfo.Mode().IsRegular() || !os.SameFile(openedInfo, pathInfo) {
		return fmt.Errorf("validate opened %q: file changed while opening", file.RelativePath)
	}
	canonical, err := filepath.EvalSymlinks(file.Path)
	if err != nil {
		return fmt.Errorf("validate opened %q: %w", file.RelativePath, err)
	}
	if !pathWithin(file.Root, canonical) {
		return fmt.Errorf("validate opened %q: file escaped configured root", file.RelativePath)
	}
	return nil
}

func resolveConfiguredRoot(projectRoot, entry string) (string, error) {
	expanded := os.ExpandEnv(entry)
	var err error
	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		expanded, err = config.ExpandPath(expanded)
		if err != nil {
			return "", err
		}
	} else if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(projectRoot, expanded)
	}
	expanded, err = filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(expanded)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("not a directory")
	}
	canonical, err := filepath.EvalSymlinks(expanded)
	if err != nil {
		return "", err
	}
	dir, err := os.Open(canonical)
	if err != nil {
		return "", err
	}
	if _, readErr := dir.Readdirnames(1); readErr != nil && !errors.Is(readErr, io.EOF) {
		dir.Close()
		return "", readErr
	}
	if err := dir.Close(); err != nil {
		return "", err
	}
	return filepath.Clean(canonical), nil
}

func isLogName(name string) bool {
	lower := strings.ToLower(name)
	for _, suffix := range []string{".gz", ".zip", ".xz", ".bz2"} {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	return strings.HasSuffix(lower, ".log") ||
		strings.Contains(lower, ".log.") ||
		strings.HasSuffix(lower, ".out") ||
		strings.HasSuffix(lower, ".err") ||
		strings.HasSuffix(lower, ".tsv") ||
		strings.HasSuffix(lower, ".jsonl") ||
		strings.HasSuffix(lower, ".ndjson")
}

func hasBinaryPrefix(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	buffer := make([]byte, 8*1024)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	for _, value := range buffer[:n] {
		if value == 0 {
			return true, nil
		}
	}
	return false, nil
}

func fileID(root, path string) string {
	sum := sha256.Sum256([]byte(root + "\x00" + path))
	return hex.EncodeToString(sum[:12])
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
