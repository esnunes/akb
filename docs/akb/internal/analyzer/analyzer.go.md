# analyzer.go

Orchestrates concurrent file analysis using Claude CLI, writing markdown knowledge base entries and maintaining a manifest for incremental processing. Provides both the main processing pipeline (`Run`) and cleanup of stale outputs (`CleanStale`).

## Types

### Result

```go
type Result struct {
    Processed int
    Failed    int
    Cached    int
    Errors    []FileError
}
```

Summarizes the outcome of a `Run` invocation. `Processed` counts successfully analyzed files, `Failed` counts errors, and `Cached` counts files skipped because their content hash was unchanged in the manifest.

### FileError

```go
type FileError struct {
    RelPath string
    Err     error
}
```

Records a failure for a specific file, associating the repository-relative path with the error encountered during processing.

## Functions

### Run

```go
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, m manifest.Manifest, workers int, force bool) Result
```

Processes source files concurrently to produce markdown knowledge base entries. Core pipeline for the `generate` subcommand.

**Parameters:**
- `ctx` — context for cancellation propagation (passed to `claude.AnalyzeFile`)
- `repoPath` — repository root path (resolved to absolute internally)
- `files` — list of files to consider, as returned by `walker`
- `m` — manifest map tracking file hashes for incremental builds; **mutated in place** with new hashes on success and deletions on stale cleanup
- `workers` — concurrency limit (bounded via buffered channel semaphore)
- `force` — when true, reprocesses all files regardless of manifest cache

**Behavior:**
1. Hashes each file via `manifest.HashFile`; skips unchanged files unless `force` is set
2. Spawns goroutines (bounded by `workers`) that read file content, call `claude.AnalyzeFile`, and write output to `docs/akb/<relPath>.md`
3. Updates the manifest entry on success; collects `FileError` entries on failure
4. Uses `sync.Mutex` to protect shared state (errors slice, manifest map) and `atomic.Int32` for progress logging

**Returns:** `Result` with counts and any errors.

**Dependencies:** `claude.AnalyzeFile`, `manifest.HashFile`, `manifest.Manifest.Changed`, `walker.FileInfo`

**Side effects:** Creates directories under `docs/akb/`, writes `.md` files, mutates the manifest map.

### CleanStale

```go
func CleanStale(repoPath string, currentFiles []walker.FileInfo, m manifest.Manifest) int
```

Removes markdown output files and manifest entries for source files that no longer exist in the repository.

**Parameters:**
- `repoPath` — repository root path
- `currentFiles` — the current set of tracked files (used to determine what still exists)
- `m` — manifest map; **mutated in place** (stale entries are deleted)

**Returns:** Count of removed stale entries.

**Behavior:**
1. Builds a set of current relative paths
2. Iterates manifest entries; for any not in the current set, removes the corresponding `docs/akb/<relPath>.md` file and deletes the manifest entry
3. Walks `docs/akb/` attempting to remove empty directories (bottom-up cleanup via `os.Remove`, which only succeeds on empty dirs)

**Side effects:** Deletes files and empty directories under `docs/akb/`, mutates the manifest map.
