# internal/summarizer/

Generates bottom-up folder summaries for the repository knowledge base, processing directories from deepest to shallowest so that child summaries feed into parent summarization via Claude CLI.

## Architecture

The package implements a single orchestration pipeline in `Run`:

1. **Collect** — `collectFolders` discovers all unique folder paths (including ancestors) that transitively contain source files.
2. **Dirty detection** — `buildDirtySet` determines which folders need regeneration based on recently processed files, missing summary files on disk, or the `force` flag. Dirtiness propagates upward to ancestor folders.
3. **Depth grouping** — `groupByDepth` sorts folders deepest-first, enabling bottom-up processing so child summaries exist before their parents are summarized.
4. **Concurrent generation** — A goroutine pool with semaphore processes each depth level, calling `gatherChildContent` to assemble per-file and subfolder markdown, then `claude.SummarizeFolder` to generate the summary. Results are written to `docs/akb/` and recorded in the manifest with `dir:` prefixed keys.
5. **Stale cleanup** — `CleanStale` removes orphaned folder summaries for directories that no longer contain source files.

## Key Types

- **`Result`** — Aggregated outcome (processed, failed, cached counts plus `FolderError` details).
- **`FolderError`** — Records a per-folder failure with path and error.

## Output Convention

- Root folder (`.`) maps to `docs/akb/README.md`
- Other folders map to `docs/akb/<parent>/<name>.md`
- Manifest keys use `dir:` prefix (e.g., `dir:internal/claude`)

## Dependencies

- `claude.SummarizeFolder` — LLM-powered summary generation
- `walker.FileInfo` — source file metadata from the file discovery phase
- `manifest.Manifest` — content-hash manifest for caching, mutated in-place

## Test Coverage

`summarizer_test.go` provides table-driven tests for all helper functions: folder collection, depth grouping, output path mapping, dirty-set computation, depth calculation, and manifest key round-tripping.
