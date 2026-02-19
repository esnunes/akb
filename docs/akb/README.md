# akb — AI Knowledge Base Generator

CLI tool that analyzes source repositories and produces a hierarchical markdown knowledge base designed for consumption by AI coding agents. It uses the Claude CLI as a subprocess to classify file types, analyze individual source files, and generate folder-level summaries.

## Commands

- **`init`** — Discovers file extensions in a repo, classifies them as source/non-source via Claude CLI, and writes a config file (`docs/akb/.config.yaml`).
- **`generate`** — Incrementally analyzes source files and produces markdown documentation under `docs/akb/`, with concurrent processing and crash-resilient manifest tracking.

## Processing Pipeline (`generate`)

```
config → walker → analyzer → summarizer
                     ↓            ↓
                  manifest ← (shared)
                     ↑
                  claude (LLM subprocess)
```

1. **config** loads analysis scope (source extensions, exclude patterns)
2. **walker** traverses the repo tree, collecting matching files
3. **manifest** tracks SHA-256 hashes for incremental builds, skipping unchanged files
4. **analyzer** runs concurrent per-file analysis via Claude CLI, producing per-file markdown docs
5. **summarizer** generates bottom-up folder summaries from the per-file output
6. Both analyzer and summarizer clean stale outputs for deleted sources

## Key Design Decisions

- **Claude CLI as subprocess** — all LLM calls route through `claude -p <prompt>` with retry logic and response envelope parsing
- **Incremental processing** — manifest saves after each processed item for crash resilience
- **Bounded concurrency** — `sync.WaitGroup` + buffered channel semaphore (1–20 workers)
- **Atomic writes** — temp-file + `os.Rename` for both manifest and output files
- **stdlib only** — uses `flag.NewFlagSet` per subcommand, `slog` for logging; sole external dependency is `gopkg.in/yaml.v3`
