# akb CLI — Repository Root

Go command-line tool that generates structured markdown knowledge bases from source code repositories, using Claude CLI as an AI backend for file analysis and summarization.

## Architecture

The tool operates in two phases, each mapped to a subcommand:

1. **`init`** — Discovers file extensions in the target repo, classifies them as source/non-source via Claude CLI, and writes a config file (`docs/akb/.config.yaml`).

2. **`generate`** — Reads the config, walks matching source files, concurrently analyzes each via Claude CLI, generates bottom-up folder summaries, and maintains an incremental manifest (`.manifest.json`) so only changed files are re-processed on subsequent runs.

## Entry Point

`main.go` handles CLI parsing with stdlib `flag.NewFlagSet` per subcommand, configures `slog` logging, validates inputs, and delegates to `cmdInit` or `cmdGenerate` which orchestrate the respective pipelines.

## Internal Packages (`internal/`)

The packages form a layered pipeline:

- **config** — Persists analysis scope (source extensions, exclude patterns) as YAML; bridges `init` output to `generate` input.
- **walker** — Filesystem traversal; discovers extensions (`init`) and collects source files (`generate`), respecting exclude patterns.
- **claude** — Sole interface to the Claude CLI subprocess. Provides extension classification (JSON), file analysis (text), and folder summarization (text), all through a shared retry-with-backoff pipeline that handles env filtering and response envelope parsing.
- **analyzer** — Concurrent file analysis engine using worker pool pattern (`WaitGroup` + channel semaphore). Checks content hashes against manifest to skip unchanged files, writes markdown output, cleans stale files.
- **summarizer** — Bottom-up folder summary generator. Processes directories deepest-first so child summaries feed into parent calls. Propagates dirtiness upward and cleans orphaned summaries.
- **manifest** — Tracks SHA-256 content hashes in JSON for incremental processing. Uses atomic writes (temp file + rename) for crash safety.

## Key Design Patterns

- **Incremental processing** — Manifest-based content hashing avoids redundant Claude CLI calls across runs.
- **Concurrency** — Both analyzer and summarizer use buffered channel semaphores with configurable worker counts (1–20).
- **Error aggregation** — Processing continues past individual file failures; structured `Result` types collect errors and report counts at the end.
- **Subprocess isolation** — `CLAUDECODE` env var is filtered from subprocess calls so akb can run inside a Claude Code session.
