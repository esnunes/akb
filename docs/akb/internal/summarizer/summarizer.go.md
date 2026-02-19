# summarizer.go

Generates folder-level markdown summaries bottom-up for a repository's directory tree. Orchestrates concurrent folder summarization using Claude CLI, manages caching/staleness via a manifest, and gathers child file/subfolder content to feed into each folder's summary prompt.

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

Summarizes the outcome of folder summary generation. `Processed` counts successfully summarized folders, `Failed` counts errors, and `Cached` counts folders skipped because their summaries were already up to date.

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

Generates folder summaries bottom-up (deepest folders first) for all folders containing source files. Folders are grouped by depth and processed concurrently within each depth level using a semaphore-bounded goroutine pool. For each dirty folder, it gathers child content, calls `claude.SummarizeFolder`, writes the output markdown, and incrementally saves the manifest. Folders not in the dirty set are counted as cached.

- **Parameters**: `ctx` for cancellation, `repoPath` as repo root, `files` as the full file list from walker, `processedFiles` as relative paths of files that changed (drives dirty detection), `m` as the manifest map (mutated in place), `workers` as concurrency limit, `force` to regenerate all.
- **Returns**: `Result` with counts of processed, failed, and cached folders.
- **Side effects**: Writes `.md` files under `docs/akb/`, mutates and saves manifest.
- **Dependencies**: `claude.SummarizeFolder`, `manifest.Save`, `walker.FileInfo`.

### CleanStale

```go
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int
```

Removes folder summary files and manifest entries for folders that no longer contain any source files. Iterates manifest keys with the `dir:` prefix and checks if the folder still exists in the current file set.

- **Returns**: Count of removed stale entries.
- **Side effects**: Deletes files from disk, mutates manifest map (does not save it).

### collectFolders

```go
func collectFolders(files []walker.FileInfo) []string
```

Returns all unique folder paths (including ancestor directories up to root `.`) that contain source files directly or transitively. Walks up from each file's directory.

### buildDirtySet

```go
func buildDirtySet(folders []string, processedFiles []string, absRoot string, force bool) map[string]bool
```

Determines which folders need regeneration. A folder is dirty if: `force` is true, it contains a processed file (or is an ancestor of one), or its summary file doesn't exist on disk. Missing summaries propagate dirtiness upward to all ancestors.

### groupByDepth

```go
func groupByDepth(folders []string) [][]string
```

Groups folders by path depth and returns slices sorted deepest-first. Enables bottom-up processing so child summaries exist before parent summarization begins.

### outputPath

```go
func outputPath(absRoot string, folder string) string
```

Returns the absolute path for a folder summary file. Root `.` maps to `docs/akb/README.md`; other folders map to `docs/akb/<parent>/<name>.md`.

### manifestKey

```go
func manifestKey(folder string) string
```

Returns the manifest key for a folder entry, prefixed with `dir:` (e.g., `dir:internal/walker`).

### fromManifestKey

```go
func fromManifestKey(key string) (string, bool)
```

Extracts the folder path from a `dir:`-prefixed manifest key. Returns `("", false)` for non-folder keys.

### depth

```go
func depth(folder string) int
```

Returns the depth of a folder path (number of path separators + 1). Root `.` has depth 0.

### gatherChildContent

```go
func gatherChildContent(absRoot string, folder string, files []walker.FileInfo) (string, error)
```

Reads per-file markdown summaries and subfolder summaries for direct children of the given folder. Assembles them into a single string with `## File:` and `## Subfolder:` headers. This content is passed to Claude as context for generating the folder summary. Has special handling for the root `.` folder. Warns but continues if individual child files are unreadable.
