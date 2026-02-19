# internal/summarizer/

Generates bottom-up folder-level markdown summaries for the repository's documentation tree, orchestrating concurrent Claude CLI calls to produce hierarchical context that complements per-file summaries.

## Architecture

The package processes folders **deepest-first** so child summaries are available before parent summarization begins. The pipeline:

1. **Collect** — `collectFolders` derives all unique folder paths (including ancestors) from the file list.
2. **Dirty detection** — `buildDirtySet` determines which folders need regeneration based on changed files, missing summaries, or the `force` flag. Dirtiness propagates upward to ancestors.
3. **Group by depth** — `groupByDepth` buckets folders deepest-first for bottom-up processing.
4. **Concurrent summarization** — `Run` processes each depth level using a semaphore-bounded goroutine pool. For each dirty folder, `gatherChildContent` assembles per-file and subfolder markdown, then `claude.SummarizeFolder` generates the summary. Results are written to `docs/akb/` and the manifest is saved incrementally.
5. **Stale cleanup** — `CleanStale` removes summary files and manifest entries for folders no longer containing source files.

## Key Design Decisions

- **Incremental processing** — Only dirty folders are regenerated; clean folders are counted as cached. Manifest tracks state with `dir:`-prefixed keys.
- **Bottom-up ordering** — Guarantees child summaries exist before parents consume them via `gatherChildContent`.
- **Atomic manifest saves** — Manifest is saved after each folder to preserve progress on interruption.
- **Output mapping** — Root folder (`.`) maps to `docs/akb/README.md`; others map to `docs/akb/<parent>/<name>.md`.

## Files

- **summarizer.go** — All production logic: folder collection, dirty detection, depth grouping, child content gathering, concurrent orchestration (`Run`), and stale cleanup (`CleanStale`).
- **summarizer_test.go** — Table-driven tests covering `collectFolders`, `groupByDepth`, `outputPath`, `buildDirtySet`, `depth`, and `manifestKey`/`fromManifestKey` round-tripping.
