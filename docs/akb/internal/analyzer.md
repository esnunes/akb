# internal/analyzer/

Orchestrates concurrent file analysis through Claude CLI, producing markdown knowledge base entries under `docs/akb/` with content-addressed caching for incremental builds. This is the core processing engine of the `generate` subcommand.

## Key Responsibilities

- **Concurrent analysis** — Processes repository files through `claude.AnalyzeFile` using a bounded worker pool (`sync.WaitGroup` + buffered channel semaphore).
- **Incremental builds** — Hashes files via `manifest.HashFile` and skips unchanged content, avoiding redundant Claude CLI calls. A `force` flag overrides this for full rebuilds.
- **Manifest management** — Updates the content-addressed manifest in-place after each successful file, saving incrementally to disk for crash resilience.
- **Stale cleanup** — `CleanStale` removes output files and manifest entries for source files that no longer exist, preserving `dir:`-prefixed entries managed by the summarizer. Empty directories are cleaned up bottom-up.

## Architecture

`Run` is the main entry point. For each input file it: hashes → checks manifest → reads content → calls `claude.AnalyzeFile` → writes `docs/akb/<relPath>.md` → updates and saves manifest. Errors are collected under a shared mutex and reported via the `Result` struct.

## Dependencies

| Package | Role |
|---|---|
| `claude` | `AnalyzeFile` — sends file content to Claude CLI for analysis |
| `manifest` | `HashFile`, `Save` — content hashing and persistent manifest I/O |
| `walker` | `FileInfo` — provides the file list to process |
