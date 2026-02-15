# analyzer.go

Core analysis orchestrator that concurrently processes repository files through Claude CLI, writes markdown knowledge base outputs, and manages cache/staleness via a manifest.

## Types

### Result

```go
type Result struct {
    Processed      int
    Failed         int
    Cached         int
    Errors         []FileError
    ProcessedFiles []string
}
```

Summary of a `Run` execution. `Processed` counts successfully analyzed files, `Failed` counts errors, `Cached` counts files skipped due to unchanged hashes. `ProcessedFiles` holds relative paths of files that were successfully written.

### FileError

```go
type FileError struct {
    RelPath string
    Err     error
}
```

Records a failure for a specific file, pairing the relative path with the error encountered during processing.

## Functions

### Run

```go
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, m manifest.Manifest, workers int, force bool) Result
```

Concurrently analyzes source files and writes markdown outputs to `docs/akb/<relPath>.md`.

**Parameters:**
- `ctx` — context for cancellation propagation (passed to `claude.AnalyzeFile`)
- `repoPath` — repository root path (resolved to absolute internally)
- `files` — list of files to consider for processing
- `m` — manifest map used for change detection and updated in-place with new hashes on success
- `workers` — concurrency limit (buffered channel semaphore)
- `force` — when true, skips cache check and reprocesses all files

**Behavior:**
1. Hashes each file via `manifest.HashFile`; skips unchanged files unless `force` is set
2. Spawns goroutines (bounded by `workers` semaphore) that read file content, call `claude.AnalyzeFile`, create output directories, and write markdown
3. Uses `sync.Mutex` to safely collect errors, update manifest entries, and track processed file paths
4. Uses `atomic.Int32` for progress logging

**Dependencies:** `claude.AnalyzeFile`, `manifest.HashFile`, `manifest.Manifest.Changed`

### CleanStale

```go
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int
```

Removes markdown files and manifest entries for source files that no longer exist in the repository.

**Parameters:**
- `repoPath` — repository root path
- `currentFiles` — the current set of valid source files
- `m` — manifest map, modified in-place (stale entries deleted)

**Returns:** count of removed stale entries.

**Behavior:**
1. Builds a set of current relative paths
2. Iterates manifest entries; skips `dir:` prefixed keys (folder summaries managed elsewhere)
3. Removes the corresponding `docs/akb/<relPath>.md` file and deletes the manifest entry
4. Walks `docs/akb/` to clean up empty directories via bottom-up `os.Remove` (fails silently on non-empty dirs)
