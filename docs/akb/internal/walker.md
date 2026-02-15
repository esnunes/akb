# internal/walker/

Filesystem traversal package that discovers file extensions and collects source files from a repository tree, serving as the input-gathering layer for both the `init` and `generate` subcommands.

## Key Components

- **`DiscoverExtensions`** — Walks the repo to collect a deduplicated list of file extensions (or filenames for extensionless files like `Makefile`). Used by `init` to determine which file types exist before classification. Skips `.git`, `node_modules`, and `vendor` but intentionally walks `docs/`.

- **`WalkSourceFiles`** — Walks the repo and returns `FileInfo` structs (relative + absolute paths) for files matching `config.Config.SourceExtensions`. Respects `config.Config.ExcludePatterns` and always excludes `docs/akb/` (the tool's own output directory). Used by `generate` to collect files for analysis.

- **`shouldExcludeDir`** — Unexported helper for directory exclusion logic. Matches patterns against directory name, full relative path, or path prefix. Hardcodes `docs/akb` exclusion.

- **`FileInfo`** — Simple struct holding `RelPath` and `AbsPath` for discovered source files.

## Design Notes

- Both public functions resolve the repo path to absolute internally, skip symlinks, and log via `slog.Debug`.
- Directory exclusion is config-driven (patterns from `.akb.yaml`) with a hardcoded exclusion for the tool's own output to prevent self-referential analysis.
- The test suite covers extension discovery, config-driven filtering, exclude patterns, and the `docs/akb/` hardcoded exclusion.
