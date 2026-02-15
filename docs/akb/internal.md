# internal/

Core implementation packages for the akb CLI tool, organized as a pipeline that discovers source files, analyzes them via Claude CLI, and produces a structured markdown knowledge base with incremental caching.

## Pipeline Architecture

The packages form a linear data flow matching the `generate` subcommand's execution:

1. **config/** — Loads `.akb.yaml` to determine which file extensions to analyze and which paths to exclude. Shared by both `init` and `generate` subcommands as the single source of truth for analysis scope.

2. **walker/** — Traverses the repository filesystem to discover files. Provides two modes: `DiscoverExtensions` (for `init`, collecting all unique extensions) and `WalkSourceFiles` (for `generate`, collecting files matching the config). Hardcodes exclusion of `docs/akb/` to prevent self-referential analysis.

3. **analyzer/** — Orchestrates concurrent file processing using a semaphore-bounded goroutine pool. Checks the manifest for unchanged files (cache hits), delegates new/modified files to `claude.AnalyzeFile`, writes markdown output to `docs/akb/`, and cleans up stale entries.

4. **summarizer/** — Generates folder-level summaries bottom-up (deepest directories first) so child summaries feed into parent summaries. Uses dirty-set propagation to avoid redundant LLM calls, and writes results with `dir:`-prefixed manifest keys.

5. **claude/** — The sole LLM integration layer, wrapping the `claude` CLI subprocess. Provides `ClassifyExtensions` (JSON output for `init`), `AnalyzeFile` (text output for per-file docs), and `SummarizeFolder` (text output for folder rollups). All calls use retry with exponential backoff and handle the CLI's JSON envelope format.

6. **manifest/** — Persistence layer for incremental processing. Tracks SHA-256 hashes of source files in `.manifest.json` with atomic writes. Used by both `analyzer` and `summarizer` to skip unchanged content between runs.

## Cross-Package Dependencies

```
config ←── walker ←── analyzer ──→ claude
                          │            ↑
                          ↓            │
                      manifest    summarizer
```

- **walker** reads config to filter files; **analyzer** consumes walker output
- **analyzer** and **summarizer** both read/write the shared manifest and call into **claude**
- **summarizer** runs after **analyzer**, consuming its processed file list to determine dirty folders
- **config** and **manifest** are pure data packages with no upstream dependencies
