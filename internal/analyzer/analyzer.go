package analyzer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/esnunes/akb/internal/claude"
	"github.com/esnunes/akb/internal/manifest"
	"github.com/esnunes/akb/internal/walker"
)

// Result summarizes the outcome of a generate run.
type Result struct {
	Processed int
	Failed    int
	Cached    int
	Errors    []FileError
}

// FileError records a failure for a specific file.
type FileError struct {
	RelPath string
	Err     error
}

// Run processes the given files concurrently, writing markdown output and
// updating the manifest. It returns a summary of results.
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, m manifest.Manifest, workers int, force bool) Result {
	absRoot, _ := filepath.Abs(repoPath)

	// Determine which files need processing.
	type job struct {
		file walker.FileInfo
		hash string
	}

	var jobs []job
	cached := 0

	for _, f := range files {
		hash, err := manifest.HashFile(f.AbsPath)
		if err != nil {
			slog.Warn("cannot hash file", "path", f.RelPath, "error", err)
			continue
		}

		if !force && !m.Changed(f.RelPath, hash) {
			slog.Debug("file unchanged, skipping", "path", f.RelPath)
			cached++
			continue
		}

		slog.Debug("file needs processing", "path", f.RelPath, "force", force)
		jobs = append(jobs, job{file: f, hash: hash})
	}

	total := len(jobs)
	slog.Info("files to process", "total", total+cached, "changed", total, "cached", cached)

	if total == 0 {
		return Result{Cached: cached}
	}

	// Worker pool.
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []FileError
	var processed atomic.Int32

	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()

			idx := processed.Add(1)
			slog.Info("processing file", "index", idx, "total", total, "path", j.file.RelPath)

			content, err := os.ReadFile(j.file.AbsPath)
			if err != nil {
				slog.Error("failed to read file", "path", j.file.RelPath, "error", err)
				mu.Lock()
				errors = append(errors, FileError{RelPath: j.file.RelPath, Err: err})
				mu.Unlock()
				return
			}

			md, err := claude.AnalyzeFile(ctx, j.file.RelPath, string(content))
			if err != nil {
				slog.Error("failed to analyze file", "path", j.file.RelPath, "error", err)
				mu.Lock()
				errors = append(errors, FileError{RelPath: j.file.RelPath, Err: err})
				mu.Unlock()
				return
			}

			outPath := filepath.Join(absRoot, "docs", "akb", j.file.RelPath+".md")
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				slog.Error("failed to create output directory", "path", outPath, "error", err)
				mu.Lock()
				errors = append(errors, FileError{RelPath: j.file.RelPath, Err: err})
				mu.Unlock()
				return
			}

			if err := os.WriteFile(outPath, []byte(md+"\n"), 0644); err != nil {
				slog.Error("failed to write markdown", "path", outPath, "error", err)
				mu.Lock()
				errors = append(errors, FileError{RelPath: j.file.RelPath, Err: err})
				mu.Unlock()
				return
			}

			slog.Debug("file processed successfully", "path", j.file.RelPath, "output", outPath)

			// Update manifest.
			mu.Lock()
			m[j.file.RelPath] = j.hash
			mu.Unlock()
		}(j)
	}

	wg.Wait()

	return Result{
		Processed: total - len(errors),
		Failed:    len(errors),
		Cached:    cached,
		Errors:    errors,
	}
}

// CleanStale removes markdown files for source files that no longer exist.
// It also removes their entries from the manifest.
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int {
	absRoot, _ := filepath.Abs(repoPath)

	current := make(map[string]struct{}, len(currentFiles))
	for _, f := range currentFiles {
		current[f.RelPath] = struct{}{}
	}

	removed := 0
	for relPath := range m {
		if _, exists := current[relPath]; exists {
			continue
		}

		mdPath := filepath.Join(absRoot, "docs", "akb", relPath+".md")
		slog.Debug("removing stale file", "source", relPath, "markdown", mdPath)
		if err := os.Remove(mdPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("could not remove stale file", "path", mdPath, "error", err)
		}

		delete(m, relPath)
		removed++
	}

	// Clean up empty directories.
	docsAkb := filepath.Join(absRoot, "docs", "akb")
	filepath.WalkDir(docsAkb, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == docsAkb {
			return nil
		}
		os.Remove(path)
		return nil
	})

	return removed
}
