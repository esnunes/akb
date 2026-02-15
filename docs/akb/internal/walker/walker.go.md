# walker.go

File in `internal/walker` that provides filesystem traversal utilities for discovering file extensions and collecting source files in a repository, with support for excluding directories and symlinks.

## Types

### FileInfo

```go
type FileInfo struct {
    RelPath string // path relative to repo root
    AbsPath string // absolute path
}
```

Holds metadata about a source file found during walking. Used as the return type of `WalkSourceFiles`.

## Functions

### DiscoverExtensions

```go
func DiscoverExtensions(repoPath string) ([]string, error)
```

Walks the repository tree and returns a deduplicated list of file extensions (e.g. `.go`, `.js`). Extensionless files are tracked by filename instead (e.g. `Makefile`, `Dockerfile`). Skips `.git`, `node_modules`, `vendor` directories and symlinks. Used by the `init` subcommand to discover what file types exist before classifying them.

- **Parameters:** `repoPath` — path to the repository root (resolved to absolute internally).
- **Returns:** slice of unique extension/filename strings; error if the walk fails.
- **Side effects:** emits `slog.Debug` messages during traversal.

### WalkSourceFiles

```go
func WalkSourceFiles(repoPath string, cfg *config.Config) ([]FileInfo, error)
```

Walks the repository and returns `FileInfo` entries for files whose extension matches `cfg.SourceExtensions`. Directories matching `cfg.ExcludePatterns` are skipped, as are symlinks. Used by the `generate` subcommand to collect files for analysis.

- **Parameters:** `repoPath` — repository root; `cfg` — parsed `config.Config` providing source extensions and exclude patterns.
- **Returns:** slice of `FileInfo`; error on walk failure.
- **Dependencies:** `config.Config`, `shouldExcludeDir`.

### shouldExcludeDir

```go
func shouldExcludeDir(relPath, name string, patterns []string) bool
```

Unexported helper that determines whether a directory should be excluded from walking. Always excludes the output directory `docs/akb`. For each pattern in `patterns`, matches against the directory name, full relative path, or as a path prefix.

- **Parameters:** `relPath` — path relative to repo root; `name` — directory base name; `patterns` — exclusion patterns from config.
- **Returns:** `true` if the directory should be skipped.
