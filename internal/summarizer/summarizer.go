package summarizer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/esnunes/akb/internal/claude"
	"github.com/esnunes/akb/internal/manifest"
	"github.com/esnunes/akb/internal/walker"
)

// Result summarizes the outcome of folder summary generation.
type Result struct {
	Processed int
	Failed    int
	Cached    int
	Errors    []FolderError
}

// FolderError records a failure for a specific folder.
type FolderError struct {
	FolderPath string
	Err        error
}

// Run generates folder summaries bottom-up for all folders containing source files.
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, processedFiles []string, m manifest.Manifest, workers int, force bool) Result {
	absRoot, _ := filepath.Abs(repoPath)

	folders := collectFolders(files)
	if len(folders) == 0 {
		return Result{}
	}

	dirtySet := buildDirtySet(folders, processedFiles, absRoot, force)
	depthGroups := groupByDepth(folders)

	var totalProcessed, totalFailed, totalCached int

	// Process bottom-up: deepest folders first.
	for _, group := range depthGroups {
		var jobs []string
		for _, folder := range group {
			if dirtySet[folder] {
				jobs = append(jobs, folder)
			} else {
				totalCached++
			}
		}

		if len(jobs) == 0 {
			continue
		}

		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []FolderError
		var processed atomic.Int32

		for _, folder := range jobs {
			wg.Add(1)
			sem <- struct{}{}
			go func(folder string) {
				defer wg.Done()
				defer func() { <-sem }()

				idx := processed.Add(1)
				slog.Info("summarizing folder", "index", idx, "total", len(jobs), "path", folder)

				content, err := gatherChildContent(absRoot, folder, files)
				if err != nil {
					slog.Error("failed to gather content for folder", "path", folder, "error", err)
					mu.Lock()
					errors = append(errors, FolderError{FolderPath: folder, Err: err})
					mu.Unlock()
					return
				}

				md, err := claude.SummarizeFolder(ctx, folder, content)
				if err != nil {
					slog.Error("failed to summarize folder", "path", folder, "error", err)
					mu.Lock()
					errors = append(errors, FolderError{FolderPath: folder, Err: err})
					mu.Unlock()
					return
				}

				outPath := outputPath(absRoot, folder)
				if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
					slog.Error("failed to create output directory", "path", outPath, "error", err)
					mu.Lock()
					errors = append(errors, FolderError{FolderPath: folder, Err: err})
					mu.Unlock()
					return
				}

				if err := os.WriteFile(outPath, []byte(md+"\n"), 0644); err != nil {
					slog.Error("failed to write folder summary", "path", outPath, "error", err)
					mu.Lock()
					errors = append(errors, FolderError{FolderPath: folder, Err: err})
					mu.Unlock()
					return
				}

				slog.Debug("folder summarized successfully", "path", folder, "output", outPath)

				mu.Lock()
				m[manifestKey(folder)] = "generated"
				if err := manifest.Save(repoPath, m); err != nil {
					slog.Warn("failed to save manifest incrementally", "error", err)
				}
				mu.Unlock()
			}(folder)
		}

		wg.Wait()

		totalProcessed += len(jobs) - len(errors)
		totalFailed += len(errors)
	}

	return Result{
		Processed: totalProcessed,
		Failed:    totalFailed,
		Cached:    totalCached,
	}
}

// CleanStale removes folder summary files for folders that no longer contain
// any source files. It also removes their entries from the manifest.
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int {
	absRoot, _ := filepath.Abs(repoPath)

	currentFolders := make(map[string]struct{})
	for _, folder := range collectFolders(currentFiles) {
		currentFolders[folder] = struct{}{}
	}

	removed := 0
	for key := range m {
		folder, ok := fromManifestKey(key)
		if !ok {
			continue
		}

		if _, exists := currentFolders[folder]; exists {
			continue
		}

		outPath := outputPath(absRoot, folder)
		slog.Debug("removing stale folder summary", "folder", folder, "path", outPath)
		if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("could not remove stale folder summary", "path", outPath, "error", err)
		}

		delete(m, key)
		removed++
	}

	return removed
}

// collectFolders returns all unique folder paths (including ancestors) that
// contain source files directly or transitively.
func collectFolders(files []walker.FileInfo) []string {
	seen := make(map[string]struct{})

	for _, f := range files {
		dir := filepath.Dir(f.RelPath)
		for dir != "." {
			if _, ok := seen[dir]; ok {
				break // ancestors already added
			}
			seen[dir] = struct{}{}
			dir = filepath.Dir(dir)
		}
		seen["."] = struct{}{}
	}

	folders := make([]string, 0, len(seen))
	for folder := range seen {
		folders = append(folders, folder)
	}
	return folders
}

// buildDirtySet determines which folders need regeneration.
func buildDirtySet(folders []string, processedFiles []string, absRoot string, force bool) map[string]bool {
	dirty := make(map[string]bool, len(folders))

	if force {
		for _, f := range folders {
			dirty[f] = true
		}
		return dirty
	}

	// Mark folders containing processed files as dirty.
	for _, relPath := range processedFiles {
		dir := filepath.Dir(relPath)
		for dir != "." {
			dirty[dir] = true
			dir = filepath.Dir(dir)
		}
		dirty["."] = true
	}

	// Also mark folders whose summary doesn't exist on disk.
	for _, folder := range folders {
		if dirty[folder] {
			continue
		}
		outPath := outputPath(absRoot, folder)
		if _, err := os.Stat(outPath); os.IsNotExist(err) {
			// Propagate upward: if this folder needs regeneration,
			// its ancestors do too.
			dir := folder
			for dir != "." {
				dirty[dir] = true
				dir = filepath.Dir(dir)
			}
			dirty["."] = true
		}
	}

	return dirty
}

// groupByDepth groups folders by their depth (number of path separators)
// and returns them sorted from deepest to shallowest.
func groupByDepth(folders []string) [][]string {
	depthMap := make(map[int][]string)
	maxDepth := 0

	for _, folder := range folders {
		d := depth(folder)
		depthMap[d] = append(depthMap[d], folder)
		if d > maxDepth {
			maxDepth = d
		}
	}

	groups := make([][]string, 0, maxDepth+1)
	for d := maxDepth; d >= 0; d-- {
		if g, ok := depthMap[d]; ok {
			groups = append(groups, g)
		}
	}

	return groups
}

// outputPath returns the absolute path for a folder summary file.
func outputPath(absRoot string, folder string) string {
	if folder == "." {
		return filepath.Join(absRoot, "docs", "akb", "README.md")
	}
	parent := filepath.Dir(folder)
	name := filepath.Base(folder)
	return filepath.Join(absRoot, "docs", "akb", parent, name+".md")
}

// manifestKey returns the manifest key for a folder entry.
func manifestKey(folder string) string {
	return "dir:" + folder
}

// fromManifestKey extracts the folder path from a manifest key.
// Returns the folder path and true if this is a folder key, or ("", false) otherwise.
func fromManifestKey(key string) (string, bool) {
	if strings.HasPrefix(key, "dir:") {
		return key[4:], true
	}
	return "", false
}

// depth returns the depth of a folder path (number of separators).
// Root "." has depth 0.
func depth(folder string) int {
	if folder == "." {
		return 0
	}
	return strings.Count(folder, string(filepath.Separator)) + 1
}

// gatherChildContent reads the per-file and subfolder summary markdowns
// for direct children of the given folder.
func gatherChildContent(absRoot string, folder string, files []walker.FileInfo) (string, error) {
	var b strings.Builder

	// Gather per-file markdowns for direct child files.
	for _, f := range files {
		fileDir := filepath.Dir(f.RelPath)
		if fileDir != folder {
			continue
		}
		mdPath := filepath.Join(absRoot, "docs", "akb", f.RelPath+".md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			slog.Warn("could not read per-file markdown", "path", mdPath, "error", err)
			continue
		}
		fmt.Fprintf(&b, "## File: %s\n\n%s\n\n", f.RelPath, strings.TrimSpace(string(data)))
	}

	// Gather subfolder summaries for direct child folders.
	childFolders := make(map[string]struct{})
	for _, f := range files {
		fileDir := filepath.Dir(f.RelPath)
		if fileDir == folder || !strings.HasPrefix(fileDir, folder+string(filepath.Separator)) {
			continue
		}
		// Find the direct child folder of the current folder.
		rel, _ := filepath.Rel(folder, fileDir)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		child := filepath.Join(folder, parts[0])
		childFolders[child] = struct{}{}
	}

	// Special handling for root folder.
	if folder == "." {
		childFolders = make(map[string]struct{})
		for _, f := range files {
			fileDir := filepath.Dir(f.RelPath)
			if fileDir == "." {
				continue
			}
			parts := strings.SplitN(fileDir, string(filepath.Separator), 2)
			childFolders[parts[0]] = struct{}{}
		}
	}

	for child := range childFolders {
		summaryPath := outputPath(absRoot, child)
		data, err := os.ReadFile(summaryPath)
		if err != nil {
			slog.Warn("could not read subfolder summary", "path", summaryPath, "error", err)
			continue
		}
		fmt.Fprintf(&b, "## Subfolder: %s/\n\n%s\n\n", child, strings.TrimSpace(string(data)))
	}

	return b.String(), nil
}
