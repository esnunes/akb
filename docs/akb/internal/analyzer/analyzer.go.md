# internal/analyzer/analyzer.go

Core orchestration module that concurrently processes repository files through Claude CLI analysis, writing markdown output files and maintaining a content-addressed manifest for incremental builds.

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

Summarizes the outcome of a `Run` invocation. `Processed` counts successfully analyzed files, `Failed` counts errors, `Cached` counts files skipped due to unchanged content hashes. `ProcessedFiles` holds relative paths of successfully processed files.

### FileError

```go
type FileError struct {
    RelPath string
    Err     error
}
```

Records a processing failure for a specific file, pairing its relative path with the error encountered.

## Functions

### Run

```go
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, m manifest.Manifest, workers int, force bool) Result
```

Processes repository files concurrently using a bounded worker pool, producing markdown knowledge base entries under `docs/akb/`.

**Parameters:**
- `ctx` — context for cancellation propagation to `claude.AnalyzeFile`
- `repoPath` — repository root path (used to resolve output directory and save manifest)
- `files` — list of files to consider for processing
- `m` — manifest map used for change detection and updated in-place with new hashes
- `workers` — concurrency limit (size of the semaphore channel)
- `force` — when true, bypasses the content hash check and reprocesses all files

**Behavior:**
1. Hashes each file via `manifest.HashFile` and skips unchanged files unless `force` is set.
2. Spawns goroutines bounded by a buffered channel semaphore (`workers` capacity).
3. For each file: reads content, calls `claude.AnalyzeFile`, writes markdown to `docs/akb/<relPath>.md`.
4. On success, updates the manifest entry and saves the manifest incrementally via `manifest.Save`.
5. Errors at any stage (read, analyze, mkdir, write) are collected into `FileError` slices under a shared mutex.

**Returns:** `Result` with counts and error details.

**Dependencies:** `claude.AnalyzeFile`, `manifest.HashFile`, `manifest.Save`, `walker.FileInfo`.

**Side effects:** Creates directories under `docs/akb/`, writes `.md` files, mutates and saves the manifest to disk after each successful file.

### CleanStale

```go
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int
```

Removes markdown output files and manifest entries for source files that no longer exist in the repository.

**Parameters:**
- `repoPath` — repository root path
- `currentFiles` — the current set of tracked files (used to determine what still exists)
- `m` — manifest map, mutated in-place to remove stale entries

**Behavior:**
1. Builds a set of current relative paths for O(1) lookup.
2. Iterates manifest entries, skipping `dir:` prefixed keys (folder summaries managed by the summarizer).
3. For each stale entry, removes the corresponding `docs/akb/<relPath>.md` file and deletes the manifest key.
4. Walks `docs/akb/` and attempts to remove empty directories (bottom-up cleanup via `os.Remove` which only succeeds on empty dirs).

**Returns:** Count of removed stale entries.

**Side effects:** Deletes files and empty directories under `docs/akb/`, mutates the manifest map. Does **not** save the manifest to disk (caller is responsible).
