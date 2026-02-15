package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/esnunes/akb/internal/analyzer"
	"github.com/esnunes/akb/internal/claude"
	"github.com/esnunes/akb/internal/config"
	"github.com/esnunes/akb/internal/manifest"
	"github.com/esnunes/akb/internal/walker"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			slog.Error("init failed", "error", err)
			os.Exit(1)
		}
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			slog.Error("generate failed", "error", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "akb: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: akb <command> [flags]

Commands:
  init       Discover source file types in a repository
  generate   Generate markdown knowledge base for source files
  help       Show this help message

Run 'akb <command> --help' for command-specific flags.
`)
}

func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	path := fs.String("path", ".", "Repository root path")
	verbose := fs.Bool("verbose", false, "Enable debug logging")
	fs.Parse(args)

	setupLogger(*verbose)

	return cmdInit(*path)
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	path := fs.String("path", ".", "Repository root path")
	workers := fs.Int("workers", 5, "Number of concurrent Claude CLI calls (1-20)")
	force := fs.Bool("force", false, "Regenerate all files, ignoring manifest")
	verbose := fs.Bool("verbose", false, "Enable debug logging")
	fs.Parse(args)

	setupLogger(*verbose)

	if *workers < 1 || *workers > 20 {
		return fmt.Errorf("--workers must be between 1 and 20, got %d", *workers)
	}

	return cmdGenerate(*path, *workers, *force)
}

func cmdInit(repoPath string) error {
	if err := claude.CheckInstalled(); err != nil {
		return err
	}

	if config.Exists(repoPath) {
		slog.Warn("config already exists; delete it to re-initialize", "path", config.Path(repoPath))
		return nil
	}

	slog.Info("discovering file extensions", "repo", repoPath)
	extensions, err := walker.DiscoverExtensions(repoPath)
	if err != nil {
		return fmt.Errorf("discover extensions: %w", err)
	}

	if len(extensions) == 0 {
		slog.Warn("no files found in repository")
		return nil
	}

	slog.Info("classifying extensions", "count", len(extensions))
	slog.Debug("extensions found", "extensions", extensions)

	ctx := context.Background()
	result, err := claude.ClassifyExtensions(ctx, extensions)
	if err != nil {
		return err
	}

	slog.Debug("classification result",
		"source", result.SourceExtensions,
		"non_source", result.NonSourceExtensions)

	cfg := &config.Config{
		SourceExtensions: result.SourceExtensions,
		ExcludePatterns: []string{
			".git/",
			"node_modules/",
			"vendor/",
			"docs/akb/",
		},
	}

	if err := config.Save(repoPath, cfg); err != nil {
		return err
	}

	slog.Info("config written", "path", config.Path(repoPath), "source_extensions", len(cfg.SourceExtensions))
	return nil
}

func cmdGenerate(repoPath string, workers int, force bool) error {
	if err := claude.CheckInstalled(); err != nil {
		return err
	}

	if !config.Exists(repoPath) {
		return fmt.Errorf("no config found; run 'akb init' first")
	}

	cfg, err := config.Load(repoPath)
	if err != nil {
		return err
	}

	slog.Info("scanning files", "repo", repoPath, "extensions", cfg.SourceExtensions)
	files, err := walker.WalkSourceFiles(repoPath, cfg)
	if err != nil {
		return fmt.Errorf("walk source files: %w", err)
	}

	m, err := manifest.Load(repoPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	ctx := context.Background()
	result := analyzer.Run(ctx, repoPath, files, m, workers, force)

	// Clean stale files.
	removed := analyzer.CleanStale(repoPath, files, m)
	if removed > 0 {
		slog.Info("removed stale files", "count", removed)
	}

	// Save manifest.
	if err := manifest.Save(repoPath, m); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	slog.Info("generate complete",
		"processed", result.Processed,
		"failed", result.Failed,
		"cached", result.Cached)

	if result.Failed > 0 {
		for _, fe := range result.Errors {
			slog.Error("file failed", "path", fe.RelPath, "error", fe.Err)
		}
		return fmt.Errorf("%d files failed to process", result.Failed)
	}

	return nil
}
