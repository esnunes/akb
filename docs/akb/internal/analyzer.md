# internal/analyzer/

Concurrent file analysis engine that orchestrates the core generate workflow — sending source files to Claude CLI for LLM-powered analysis, writing markdown output to `docs/akb/`, and maintaining cache coherence through content-hash tracking.

## Key Responsibilities

- **Concurrent processing:** Uses a buffered-channel semaphore pattern to analyze multiple files in parallel, with configurable worker count.
- **Cache invalidation:** Compares file content hashes against a manifest to skip unchanged files, avoiding redundant LLM calls. Supports a `force` flag to bypass caching.
- **Stale cleanup:** Detects and removes markdown output files and manifest entries for source files that no longer exist, then prunes empty directories under `docs/akb/`.
- **Error resilience:** Collects per-file errors without aborting the run, returning a full `Result` summary of processed, failed, and cached counts.

## Public API

- **`Run`** — Main entry point for the generate workflow. Processes a list of `walker.FileInfo` files concurrently, delegates analysis to `claude.AnalyzeFile`, writes markdown output, and mutates the manifest in-place.
- **`CleanStale`** — Post-processing step that removes orphaned output files and manifest entries for deleted source files.
- **`Result` / `FileError`** — Data types for reporting run outcomes and per-file failures.

## Dependencies

Sits between the file discovery layer (`walker`) and the LLM interface (`claude`), using `manifest` for hash-based caching. Writes output to the `docs/akb/` directory tree mirroring the repository structure.
