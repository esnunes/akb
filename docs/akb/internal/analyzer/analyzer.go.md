# analyzer.go

Core processing engine that concurrently analyzes source files via Claude CLI, writes markdown knowledge base entries, and manages cache invalidation by tracking file hashes in a manifest.

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

Summary of a generate run's outcome. `Processed` counts successfully analyzed files, `Failed` counts errors, `Cached` counts skipped-because-unchanged files, and `ProcessedFiles` lists relative paths of files that were successfully written.

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

Concurrently processes source files into markdown knowledge base entries. For each file:
1. Computes a content hash and compares against the manifest to skip unchanged files (unless `force` is true).
2. Reads the file, sends it to `claude.AnalyzeFile` for LLM-powered analysis.
3. Writes the resulting markdown to `docs/akb/<relPath>.md`.
4. Updates the manifest map in-place with the new hash.

Uses a buffered-channel semaphore pattern with `workers` goroutines for concurrency. Errors are collected per-file rather than aborting the run. Progress is logged via `slog`.

**Parameters:**
- `ctx` — context for cancellation (passed through to Claude CLI calls)
- `repoPath` — repository root path (resolved to absolute internally)
- `files` — list of files to process (from `walker` package)
- `m` — manifest map, mutated in-place with updated hashes for processed files
- `workers` — max concurrent goroutines
- `force` — when true, reprocesses all files regardless of cache

**Returns:** `Result` summarizing processed, failed, cached counts and any errors.

**Dependencies:** `manifest.HashFile`, `claude.AnalyzeFile`, `walker.FileInfo`

**Side effects:** Writes files under `docs/akb/`, mutates the manifest map.

### CleanStale

```go
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int
```

Removes markdown output files and manifest entries for source files that no longer exist in the repository. After removing stale files, walks `docs/akb/` to clean up any empty directories left behind (using `os.Remove` which only succeeds on empty dirs).

**Parameters:**
- `repoPath` — repository root path
- `currentFiles` — the current set of tracked files; anything in the manifest but not in this set is considered stale
- `m` — manifest map, mutated in-place (stale entries deleted)

**Returns:** Count of removed stale entries.

**Side effects:** Deletes files under `docs/akb/`, removes empty directories, mutates the manifest map.
