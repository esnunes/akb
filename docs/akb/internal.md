# internal/

Core implementation packages for the akb CLI tool, organized as a pipeline that discovers repository files, classifies them via Claude CLI, analyzes their contents, and produces a structured markdown knowledge base under `docs/akb/`.

## Pipeline Architecture

The packages form a layered data flow matching the tool's two subcommands:

**`init` subcommand path:**
`walker.DiscoverExtensions` → `claude.ClassifyExtensions` → `config.Save`

**`generate` subcommand path:**
`config.Load` → `walker.WalkSourceFiles` → `analyzer.Run` → `summarizer.Run` → `manifest.Save`

## Package Roles

- **config/** — Single source of truth for analysis scope. Bridges `init` (writes config) and `generate` (reads config) by persisting `SourceExtensions` and `ExcludePatterns` in `docs/akb/.config.yaml`.

- **walker/** — Filesystem traversal layer. Discovers file extensions for classification (`init`) and collects matching source files for analysis (`generate`). Handles directory exclusion via config patterns and hardcoded rules.

- **claude/** — LLM integration layer and sole interface to the Claude CLI subprocess. Provides three capabilities: extension classification (JSON), file analysis (text), and folder summarization (text). All calls go through a shared retry-with-backoff pipeline that manages subprocess execution, env filtering, and response envelope parsing.

- **analyzer/** — Concurrent file analysis engine. Orchestrates the core `generate` workflow by sending source files through `claude.AnalyzeFile`, writing markdown output, and maintaining cache coherence via content hashes. Handles stale cleanup for deleted files.

- **summarizer/** — Bottom-up folder summary generator. After file analysis completes, processes directories deepest-first so child summaries feed into parent summarization via `claude.SummarizeFolder`. Propagates dirtiness upward and cleans orphaned summaries.

- **manifest/** — Incremental processing support. Tracks SHA-256 content hashes in `.manifest.json` so only changed or new files are re-analyzed between runs. Uses atomic writes for crash safety.

## Cross-Cutting Patterns

- **Manifest as shared state** — Both `analyzer` and `summarizer` mutate the manifest in-place during processing, using different key conventions (plain paths vs. `dir:`-prefixed keys).
- **Concurrency** — `analyzer` and `summarizer` both use `sync.WaitGroup` + buffered channel semaphore for parallel processing.
- **Cache-then-process** — Both `analyzer` and `summarizer` check hashes/dirtiness before invoking Claude CLI, skipping unchanged content.
- **Error collection** — Both return structured `Result` types that aggregate per-item failures without aborting the run.
