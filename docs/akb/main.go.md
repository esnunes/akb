# main.go

Entry point for the `akb` CLI tool. Parses subcommands (`init`, `generate`, `help`), configures logging, and dispatches to command implementations that orchestrate file discovery, extension classification, and knowledge base generation.

## Functions

### main

`func main()`

CLI entry point. Requires at least one argument (subcommand). Dispatches to `runInit`, `runGenerate`, or `printUsage` based on `os.Args[1]`. Exits with code 1 on unknown commands or subcommand errors.

### printUsage

`func printUsage()`

Prints CLI usage/help text to stderr, listing available commands (`init`, `generate`, `help`).

### setupLogger

`func setupLogger(verbose bool)`

Configures the default `slog` logger on stderr. Sets level to `slog.LevelDebug` when `verbose` is true, otherwise `slog.LevelInfo`.

### runInit

`func runInit(args []string) error`

Parses flags for the `init` subcommand (`-path`, `-verbose`) and calls `cmdInit`. Uses `flag.NewFlagSet` with `flag.ExitOnError`.

- **Parameters:** `args` — CLI arguments after the `init` subcommand.
- **Returns:** error from `cmdInit`.

### runGenerate

`func runGenerate(args []string) error`

Parses flags for the `generate` subcommand (`-path`, `-verbose`, `-workers`, `-force`) and calls `cmdGenerate`. Validates that `workers` is between 1 and 20.

- **Parameters:** `args` — CLI arguments after the `generate` subcommand.
- **Returns:** error from flag validation or `cmdGenerate`.

### cmdInit

`func cmdInit(repoPath string) error`

Implements the `init` command workflow:
1. Checks Claude CLI is installed (`claude.CheckInstalled`).
2. Skips if config already exists (`config.Exists`).
3. Discovers file extensions via `walker.DiscoverExtensions`.
4. Classifies extensions as source/non-source via `claude.ClassifyExtensions`.
5. Saves config with source extensions and default exclude patterns (`.git/`, `node_modules/`, `vendor/`, `docs/akb/`).

- **Parameters:** `repoPath` — root path of the repository to analyze.
- **Returns:** error on discovery, classification, or config save failure.
- **Dependencies:** `claude`, `config`, `walker` packages.

### cmdGenerate

`func cmdGenerate(repoPath string, workers int, force bool) error`

Implements the `generate` command workflow:
1. Checks Claude CLI is installed.
2. Loads config (requires prior `init`).
3. Walks source files matching configured extensions via `walker.WalkSourceFiles`.
4. Loads the manifest for incremental processing (`manifest.Load`).
5. Runs concurrent analysis via `analyzer.Run` with the specified worker count.
6. Cleans stale output files via `analyzer.CleanStale`.
7. Saves updated manifest.
8. Reports processed/failed/cached counts; returns error if any files failed.

- **Parameters:** `repoPath` — repository root; `workers` — concurrency level (1–20); `force` — if true, regenerates all files ignoring manifest cache.
- **Returns:** error on missing config, walk failure, manifest I/O failure, or if any files failed processing.
- **Dependencies:** `analyzer`, `claude`, `config`, `manifest`, `walker` packages.
