# internal/config/

Manages akb's YAML configuration file (`docs/akb/.config.yaml`), providing load, save, validation, and path resolution for the settings that control which files get analyzed during knowledge base generation.

## Key Components

- **`Config` struct** — Holds `SourceExtensions` (file types to analyze) and `ExcludePatterns` (globs to skip), serialized as YAML.
- **`Path(repoPath)`** — Resolves the canonical config file location: `<repo>/docs/akb/.config.yaml`.
- **`Exists(repoPath)`** — Checks whether the config file is present on disk.
- **`Load(repoPath)`** — Reads, parses, and validates the config; rejects empty `SourceExtensions` with remediation guidance.
- **`Save(repoPath, cfg)`** — Writes the config as YAML with a header comment, creating directories as needed.

## Role in the Project

This package is the bridge between the `init` subcommand (which generates the config by classifying file extensions via Claude CLI) and the `generate` subcommand (which reads the config to determine what to analyze). All other packages reference `config.Load` or `config.Path` to discover project settings — it is the single source of truth for analysis scope.

## Testing

`config_test.go` covers save/load round-tripping, missing file errors, empty-extensions validation, existence checks, and path construction. Tests use temp directories for isolation.
