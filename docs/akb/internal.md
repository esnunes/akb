# internal/

Core implementation packages for the akb CLI tool, organized as a processing pipeline that discovers source files, analyzes them via Claude CLI, and produces a hierarchical markdown knowledge base under `docs/akb/`.

## Pipeline Architecture

The packages form a clear data flow through the `generate` subcommand:

1. **config** — Loads analysis scope (source extensions, exclude patterns) from `docs/akb/.config.yaml`
2. **walker** — Traverses the repository tree, collecting files that match the config
3. **manifest** — Tracks SHA-256 hashes for incremental builds, skipping unchanged files
4. **analyzer** — Orchestrates concurrent per-file analysis through Claude CLI, producing markdown docs
5. **summarizer** — Generates bottom-up folder-level summaries from the per-file output
6. **claude** — Wraps the `claude` CLI subprocess with retry logic and response parsing; used by analyzer, summarizer, and the `init` subcommand

## Package Dependency Graph

```
config ← walker ← analyzer → claude
                      ↓          ↑
                  manifest    summarizer
```

- **config** is the shared source of truth — both `walker` and the CLI subcommands depend on it.
- **claude** is the sole LLM interface — all three operations (classify, analyze, summarize) route through it.
- **manifest** is shared by `analyzer` (per-file hashes) and `summarizer` (folder-level entries with `dir:` prefix).
- **walker** feeds file lists to both `analyzer` and the `init` subcommand's extension discovery.

## Cross-Cutting Concerns

- **Incremental processing** — `manifest` enables both `analyzer` and `summarizer` to skip unchanged work. Both save the manifest after each item for crash resilience.
- **Concurrency** — `analyzer` and `summarizer` both use bounded goroutine pools (`sync.WaitGroup` + buffered channel semaphore).
- **Stale cleanup** — Both `analyzer.CleanStale` and `summarizer.CleanStale` remove output for deleted sources, coordinating via manifest key prefixes to avoid interfering with each other.
- **Atomic writes** — `manifest.Save` uses temp-file + `os.Rename`; output files follow the same pattern.
