# main.go

Entry point for the `akb` CLI tool. Parses subcommands (`init`, `generate`, `help`), configures logging, and orchestrates the two main workflows: discovering/classifying file extensions and generating markdown knowledge base files.

## Functions

### main

`func main()`

Top-level entry point. Dispatches to `runInit`, `runGenerate`, or `printUsage` based on `os.Args[1]`. Exits with code 1 on unknown commands or subcommand errors.

### printUsage

`func printUsage()`

Prints CLI usage information to stderr, listing available commands (`init`, `generate`, `help`).

### setupLogger

`func setupLogger(verbose bool)`

Configures the default `slog` logger. Sets level to `slog.LevelDebug` when `verbose` is true, otherwise `slog.LevelInfo`. Output goes to stderr via `slog.NewTextHandler`.

### runInit

`func runInit(args []string) error`

Parses flags for the `init` subcommand (`-path`, `-verbose`) using `flag.NewFlagSet`, sets up logging, and delegates to `cmdInit`.

- **Parameters:** `args` — command-line arguments after "init"
- **Returns:** error from flag parsing or `cmdInit`

### runGenerate

`func runGenerate(args []string) error`

Parses flags for the `generate` subcommand (`-path`, `-workers`, `-force`, `-verbose`) using `flag.NewFlagSet`, sets up logging, validates worker count (1–20), and delegates to `cmdGenerate`.

- **Parameters:** `args` — command-line arguments after "generate"
- **Returns:** error from flag parsing, validation, or `cmdGenerate`

### cmdInit

`func cmdInit(repoPath string) error`

Orchestrates the `init` workflow:
1. Checks Claude CLI is installed (`claude.CheckInstalled`)
2. Skips if config already exists (`config.Exists`)
3. Discovers file extensions via `walker.DiscoverExtensions`
4. Classifies extensions as source/non-source via `claude.ClassifyExtensions`
5. Writes a new `config.Config` with source extensions and default exclude patterns

- **Parameters:** `repoPath` — repository root directory
- **Returns:** error on failure, nil on success
- **Side effects:** creates config file on disk
- **Dependencies:** `claude`, `config`, `walker` packages

### cmdGenerate

`func cmdGenerate(repoPath string, workers int, force bool) error`

Orchestrates the `generate` workflow:
1. Checks Claude CLI is installed and config exists
2. Loads config and scans source files via `walker.WalkSourceFiles`
3. Loads manifest for incremental processing (`manifest.Load`)
4. Runs concurrent file analysis via `analyzer.Run`
5. Generates folder summaries via `summarizer.Run`
6. Cleans stale output files via `analyzer.CleanStale` and `summarizer.CleanStale`
7. Saves updated manifest (`manifest.Save`)
8. Returns error if any files failed processing

- **Parameters:** `repoPath` — repository root; `workers` — concurrency level (1–20); `force` — if true, regenerate all files ignoring manifest cache
- **Returns:** error on failure (including partial failures with count), nil on success
- **Side effects:** writes/removes markdown files in `docs/akb/`, updates manifest file
- **Dependencies:** `analyzer`, `claude`, `config`, `manifest`, `summarizer`, `walker` packages
