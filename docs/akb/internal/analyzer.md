# internal/analyzer/

Orchestrates concurrent file analysis through Claude CLI and manages the resulting markdown knowledge base, including caching, staleness cleanup, and progress tracking.

## Key Components

- **`Run`** — Main entry point that concurrently processes repository files. Uses a buffered channel semaphore (`workers`) to bound goroutines, hashes files against a manifest for change detection, calls `claude.AnalyzeFile` for each file, and writes output to `docs/akb/<relPath>.md`. Collects results (processed/failed/cached counts, errors, processed file paths) via mutex-protected shared state.

- **`CleanStale`** — Removes orphaned markdown files and manifest entries for source files no longer in the repository. Skips `dir:`-prefixed manifest keys (folder summaries). Cleans up empty directories bottom-up after removal.

## Architecture

The analyzer sits between the file walker (which discovers files) and the Claude CLI integration (which generates analysis). It owns the concurrency model (`sync.WaitGroup` + semaphore), the caching layer (manifest hash comparison), and the output file lifecycle (creation, directory scaffolding, and stale cleanup). The manifest is modified in-place throughout, serving as both cache and bookkeeping for the knowledge base.
