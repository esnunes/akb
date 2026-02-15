---
title: "feat: Add folder-level summary generation"
type: feat
date: 2026-02-15
---

# feat: Add folder-level summary generation

## Overview

After generating per-file markdown documentation, akb should also produce one LLM-generated summary per folder. Summaries are built bottom-up: leaf folders first, then parents incorporate child summaries. This gives AI agents hierarchical context about project structure.

## Problem Statement / Motivation

Currently akb generates one markdown file per source file, but there's no higher-level view. An AI agent reading `docs/akb/internal/claude/claude.go.md` has no way to understand what `internal/claude/` does as a whole, or how it relates to `internal/analyzer/`. Folder summaries bridge this gap.

## Proposed Solution

Add a new `internal/summarizer` package that runs as a post-analysis stage in `cmdGenerate`. It:

1. Derives the folder tree from the discovered file list
2. Determines which folders need (re)generation
3. Processes folders bottom-up by depth level, using the same worker pool pattern
4. Calls a new `claude.SummarizeFolder()` function to generate each summary
5. Writes output and tracks entries in the manifest for stale cleanup

## Technical Approach

### Output paths

| Source folder | Output path |
|---|---|
| `internal/config/` | `docs/akb/internal/config.md` |
| `internal/` | `docs/akb/internal.md` |
| `.` (root) | `docs/akb/README.md` |

No collision with per-file docs: those use `<name>.<ext>.md` (e.g., `config.go.md` vs `config.md`).

### Determining which folders need regeneration

After `analyzer.Run` completes, we know which files were **processed** (changed) vs **cached**. Build a "dirty set" of folders:

1. For each processed file, mark its containing folder as dirty.
2. Propagate upward: if a folder is dirty, its parent is also dirty.
3. Additionally mark a folder as dirty if its summary file doesn't exist on disk.
4. If `--force`, all folders are dirty.

This avoids needing to hash folder contents. The dirty signal flows naturally from the analyzer result.

### Bottom-up processing order

1. Collect all unique folder paths from the file list (including intermediate ancestors).
2. Group by depth (number of path separators). Root `.` = depth 0.
3. Process from highest depth to lowest. Within each depth level, process concurrently using the semaphore + WaitGroup pattern from `analyzer.Run`.
4. Wait for each depth level to complete before starting the next (parents depend on child summaries).

### LLM input per folder

For each folder, read and concatenate:
- Full content of each direct child's per-file `<name>.<ext>.md` file
- Full content of each direct child subfolder's summary `.md` file (already generated in a previous depth level)

Send to `claude.SummarizeFolder()` with a prompt asking for a cohesive high-level overview. Output structure is free-form (LLM decides).

### Manifest tracking

Store folder entries with a `dir:` key prefix to avoid collision with file entries:
- `dir:internal/config` for `internal/config/`
- `dir:.` for the root folder

Value: `"generated"` (sentinel). The manifest entry exists solely for stale cleanup — change detection uses the dirty-set approach described above.

### Stale cleanup

Extend cleanup to handle folder summaries. A folder summary is stale when the `dir:` manifest entry exists but the folder no longer contains any source files (directly or transitively). Remove the summary file and the manifest entry.

## Implementation Phases

### Phase 1: `claude.SummarizeFolder` function

Add to `internal/claude/claude.go`:

```go
// SummarizeFolder sends folder documentation to Claude CLI for synthesis.
func SummarizeFolder(ctx context.Context, folderPath string, childrenContent string) (string, error)
```

- Prompt includes: folder path, concatenated child content, instructions for high-level synthesis
- Uses `--output-format text`, same `callWithRetry` and envelope extraction as `AnalyzeFile`
- Free-form output structure

### Phase 1.5: Extend `analyzer.Result` with processed file paths

The current `analyzer.Result` only has counts (`Processed int`). The summarizer needs to know *which* files changed to build its dirty set. Add a field:

```go
type Result struct {
    Processed      int
    Failed         int
    Cached         int
    Errors         []FileError
    ProcessedFiles []string // relative paths of successfully processed files
}
```

Populate `ProcessedFiles` in the worker goroutine after successful write (alongside the manifest update, under the existing mutex).

### Phase 2: `internal/summarizer` package

New file `internal/summarizer/summarizer.go`:

```go
// Result summarizes the outcome of folder summary generation.
type Result struct {
    Processed int
    Failed    int
    Cached    int
    Errors    []FolderError
}

type FolderError struct {
    FolderPath string
    Err        error
}

// Run generates folder summaries bottom-up for all folders containing source files.
func Run(ctx context.Context, repoPath string, files []walker.FileInfo, processedFiles []string, m manifest.Manifest, workers int, force bool) Result
```

Takes `processedFiles []string` (from `analyzer.Result.ProcessedFiles`) instead of the full `analyzer.Result` — avoids a circular import and keeps the interface minimal.

Internal logic:
1. `collectFolders(files)` — returns unique folder paths including all ancestors up to root
2. `buildDirtySet(folders, processedFiles, repoPath, m, force)` — determines which folders need regeneration
3. `groupByDepth(folders)` — returns `[][]string` sorted deepest-first
4. For each depth level, run a worker pool processing dirty folders at that depth
5. Each worker: reads child content from disk, calls `claude.SummarizeFolder`, writes output, updates manifest

New file `internal/summarizer/summarizer_test.go`:
- `TestCollectFolders` — derives correct folder set from file list
- `TestGroupByDepth` — correct depth grouping and ordering
- `TestOutputPath` — correct path for subfolders and root special case
- `TestBuildDirtySet` — propagation from processed files to ancestor folders

### Phase 3: Integration in `main.go`

Update `cmdGenerate` to add the summarizer stage after `analyzer.Run`:

```go
result := analyzer.Run(ctx, repoPath, files, m, workers, force)

// Generate folder summaries.
sumResult := summarizer.Run(ctx, repoPath, files, result.ProcessedFiles, m, workers, force)

// Clean stale files.
removed := analyzer.CleanStale(repoPath, files, m)
sumRemoved := summarizer.CleanStale(repoPath, files, m)
```

### Phase 4: Stale cleanup for folder summaries

Add `summarizer.CleanStale(repoPath, files, m)`:
- Iterate manifest entries with `dir:` prefix
- Check if the folder still has source files in the current file list
- Remove stale summary files and manifest entries

Keeps folder-summary concerns in the summarizer package, consistent with the separation rationale.

## Acceptance Criteria

- [x] `akb generate` produces one `.md` summary per folder under `docs/akb/`
- [x] Root folder summary written to `docs/akb/README.md`
- [x] Subfolder summaries written to parent directory (e.g., `docs/akb/internal/config.md`)
- [x] Summaries are LLM-generated via Claude CLI, not mechanical concatenation
- [x] Bottom-up ordering: leaf folders processed before parents
- [x] Parent summaries incorporate child folder summaries
- [x] Incremental runs skip folders where no contained files changed
- [x] `--force` flag regenerates all folder summaries
- [x] Stale folder summaries cleaned up when folders lose all source files
- [x] Folder summary entries in manifest use `dir:` prefix
- [x] Table-driven tests for folder collection, depth ordering, output paths, and dirty-set logic

## Edge Cases

- **Folder with only subfolders (no direct files)**: Still gets a summary based on child folder summaries alone.
- **Single-file folder**: Summary still generated (even if trivial).
- **First run after adding feature**: All summaries missing on disk, so all regenerated.
- **Manually deleted summary**: Detected as missing on disk, regenerated on next run.
- **All files deleted from a folder**: Summary becomes stale and is cleaned up.
- **Deeply nested folders**: Bottom-up processing handles arbitrary depth.

## Dependencies & Risks

- **Claude CLI context window**: Very large folders with many files could produce prompts exceeding the context window. Brainstorm decision: no limit — trust Claude CLI. Monitor in practice.
- **API costs**: Each folder summary is an additional Claude CLI call. For a project with N folders, this adds N calls per full run. Caching mitigates this for incremental runs.

## References

- Brainstorm: `docs/brainstorms/2026-02-15-folder-summaries-brainstorm.md`
- Existing analyzer pattern: `internal/analyzer/analyzer.go:32-137`
- Claude CLI integration: `internal/claude/claude.go:96-156`
- Manifest: `internal/manifest/manifest.go`
- Solutions doc: `docs/solutions/integration-issues/claude-cli-subprocess-integration.md`
