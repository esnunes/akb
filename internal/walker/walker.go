package walker

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/esnunes/akb/internal/config"
)

// FileInfo holds metadata about a source file found during walking.
type FileInfo struct {
	RelPath string // path relative to repo root
	AbsPath string // absolute path
}

// DiscoverExtensions walks the repo and returns a deduplicated list of file
// extensions found, excluding common non-source directories.
func DiscoverExtensions(repoPath string) ([]string, error) {
	seen := make(map[string]struct{})

	absRoot, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	slog.Debug("walking repo for extension discovery", "root", absRoot)

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Debug("walk error", "path", path, "error", err)
			return nil
		}

		// Skip symlinks.
		if d.Type()&fs.ModeSymlink != 0 {
			slog.Debug("skipping symlink", "path", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common non-source directories.
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				slog.Debug("skipping excluded directory", "name", name, "path", path)
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		if ext != "" {
			if _, exists := seen[ext]; !exists {
				slog.Debug("discovered extension", "ext", ext, "from", d.Name())
			}
			seen[ext] = struct{}{}
		} else {
			// Track extensionless files by name (e.g. Makefile, Dockerfile).
			if _, exists := seen[d.Name()]; !exists {
				slog.Debug("discovered extensionless file", "name", d.Name())
			}
			seen[d.Name()] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	exts := make([]string, 0, len(seen))
	for ext := range seen {
		exts = append(exts, ext)
	}

	slog.Debug("extension discovery complete", "count", len(exts))
	return exts, nil
}

// WalkSourceFiles walks the repo and returns files matching the config's
// source extensions, excluding patterns from the config.
func WalkSourceFiles(repoPath string, cfg *config.Config) ([]FileInfo, error) {
	absRoot, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	extSet := make(map[string]struct{}, len(cfg.SourceExtensions))
	for _, ext := range cfg.SourceExtensions {
		extSet[ext] = struct{}{}
	}

	var files []FileInfo

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip symlinks.
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if shouldExcludeDir(rel, d.Name(), cfg.ExcludePatterns) {
				slog.Debug("excluding directory", "path", rel)
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		if _, ok := extSet[ext]; !ok {
			return nil
		}

		slog.Debug("found source file", "path", rel)
		files = append(files, FileInfo{
			RelPath: rel,
			AbsPath: path,
		})

		return nil
	})

	return files, err
}

func shouldExcludeDir(relPath, name string, patterns []string) bool {
	// Always exclude the output directory.
	if relPath == filepath.Join("docs", "akb") {
		return true
	}

	for _, pattern := range patterns {
		p := strings.TrimSuffix(pattern, "/")
		if name == p || relPath == p || strings.HasPrefix(relPath, p+string(filepath.Separator)) {
			return true
		}
	}

	return false
}
