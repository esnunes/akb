# akb — AI Knowledge Base Generator

CLI tool that analyzes source code repositories and generates structured markdown documentation for use as context by AI coding agents. It operates through two subcommands: `init` (discovers and classifies file extensions) and `generate` (produces per-file and per-folder markdown summaries with incremental caching).

## Architecture

`main.go` serves as the entry point, parsing subcommands and flags via stdlib `flag`, then delegating to two orchestration functions:

- **`cmdInit`** — One-time setup: discovers file extensions in the repo, uses Claude CLI to classify them as source/non-source, and writes `.akb.yaml` config.
- **`cmdGenerate`** — Main workflow: walks source files per config, runs concurrent LLM analysis, generates folder summaries bottom-up, cleans stale outputs, and maintains a manifest for incremental processing.

## Internal Packages (`internal/`)

The implementation follows a pipeline architecture:

1. **config/** — Loads `.akb.yaml` defining analysis scope (source extensions, exclude patterns)
2. **walker/** — Filesystem traversal with two modes: extension discovery (`init`) and source file collection (`generate`)
3. **analyzer/** — Concurrent file processing with semaphore-bounded goroutines, manifest-based cache hits, and markdown output to `docs/akb/`
4. **summarizer/** — Bottom-up folder summaries using dirty-set propagation to minimize redundant LLM calls
5. **claude/** — LLM integration layer wrapping the `claude` CLI subprocess with retry/backoff and JSON envelope handling
6. **manifest/** — SHA-256 hash tracking in `.manifest.json` for incremental processing across runs

## Data Flow

```
config → walker → analyzer → claude
                     │           ↑
                     ↓           │
                 manifest   summarizer
```

Config defines scope, walker discovers files, analyzer processes them concurrently via claude, summarizer rolls up folder-level docs, and manifest persists state for incremental runs. All markdown output lands in `docs/akb/`.
