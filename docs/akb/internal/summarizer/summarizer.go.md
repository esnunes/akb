# summarizer.go

Orchestrates bottom-up folder summary generation for a repository knowledge base. Processes folders from deepest to shallowest, using Claude CLI to generate markdown summaries from child file/subfolder content, with concurrency control, caching, and stale cleanup.

## Types

### Result

```go
type Result struct {
    Processed int
    Failed    int
    Cached    int
    Errors    []FolderError
}
```

Summarizes the outcome of folder summary generation — counts of processed, failed, and cached folders, plus detailed error records.

### FolderError

```go
type FolderError struct {
    FolderPath string
    Err        error
}
```

Records a failure for a specific folder during summarization.

## Functions

### Run

```go
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, processedFiles []string, m manifest.Manifest, workers int, force bool) Result
```

Main entry point for folder summary generation. Collects all folders containing source files, determines which are dirty (need regeneration), groups them by depth, and processes bottom-up (deepest first) so child summaries are available when summarizing parents. Uses goroutine pool with semaphore for concurrency. Each folder's summary is generated via `claude.SummarizeFolder`, written to `docs/akb/`, and recorded in the manifest with a `dir:` prefix key. Returns aggregated results.

**Parameters:**
- `ctx` — context for cancellation
- `repoPath` — repository root path
- `files` — all source files discovered by walker
- `processedFiles` — files whose per-file summaries were regenerated this run (used for dirty detection)
- `m` — manifest map, mutated in-place to record generated folder summaries
- `workers` — max concurrent summarization goroutines
- `force` — if true, regenerate all folder summaries regardless of cache

**Dependencies:** `claude.SummarizeFolder`, `walker.FileInfo`, `manifest.Manifest`

### CleanStale

```go
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int
```

Removes folder summary files and manifest entries for folders that no longer contain any source files. Iterates manifest keys with `dir:` prefix, checks if the folder still exists in the current file set, and removes orphaned summaries from disk and manifest. Returns count of removed entries.

### collectFolders

```go
func collectFolders(files []walker.FileInfo) []string
```

Returns all unique folder paths (including ancestors up to root `.`) that contain source files directly or transitively. Walks up from each file's directory, deduplicating along the way.

### buildDirtySet

```go
func buildDirtySet(folders []string, processedFiles []string, absRoot string, force bool) map[string]bool
```

Determines which folders need regeneration. A folder is dirty if: `force` is true, it contains a recently processed file (propagated to ancestors), or its summary file doesn't exist on disk (also propagated to ancestors). Returns a map of folder path to dirty boolean.

### groupByDepth

```go
func groupByDepth(folders []string) [][]string
```

Groups folders by path depth and returns them sorted deepest-first. Enables bottom-up processing so child summaries exist before parent summarization begins.

### outputPath

```go
func outputPath(absRoot string, folder string) string
```

Returns the absolute path for a folder summary file. Root `.` maps to `docs/akb/README.md`; other folders map to `docs/akb/<parent>/<name>.md`.

### manifestKey

```go
func manifestKey(folder string) string
```

Returns the manifest key for a folder entry, prefixed with `dir:` (e.g., `dir:internal/claude`).

### fromManifestKey

```go
func fromManifestKey(key string) (string, bool)
```

Extracts the folder path from a `dir:`-prefixed manifest key. Returns the folder path and `true` if valid, or `("", false)` otherwise.

### depth

```go
func depth(folder string) int
```

Returns the depth of a folder path (number of path separators + 1). Root `.` has depth 0.

### gatherChildContent

```go
func gatherChildContent(absRoot string, folder string, files []walker.FileInfo) (string, error)
```

Reads per-file markdown summaries and subfolder summary markdowns for direct children of the given folder. Assembles them into a single string with `## File:` and `## Subfolder:` headers, which becomes the input prompt content for `claude.SummarizeFolder`. Has special handling for root folder (`.`) when identifying child folders.
