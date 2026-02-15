# Folder Summaries for akb

**Date:** 2026-02-15
**Status:** Brainstorm

## What We're Building

After akb generates per-file markdown documentation, it should also generate one markdown summary per folder. Each folder summary provides a higher-level view of all files within it, including information about subfolders.

Folder summaries are **LLM-generated** (via Claude CLI) and built **bottom-up** — leaf folders are summarized first, then parent folders incorporate child folder summaries alongside their own direct file documentation to produce progressively higher-level overviews.

## Why This Approach

- **LLM-generated** rather than mechanical aggregation: produces genuinely useful high-level descriptions that explain *purpose* and *architecture*, not just file listings.
- **Bottom-up ordering**: parent summaries can reference synthesized child summaries, giving them a rich understanding of their entire subtree without needing to process every leaf file directly.
- **Full content as input**: the complete per-file markdowns and child summaries are sent to Claude CLI, preserving all detail for accurate synthesis.

## Key Decisions

### 1. Output file naming

- Subfolders: `docs/akb/<parent>/<folder-name>.md`
  - Example: `internal/config/` -> `docs/akb/internal/config.md`
  - Example: `internal/` -> `docs/akb/internal.md`
- Root folder: `docs/akb/README.md`
- No naming collision with per-file docs since those use `<filename>.<ext>.md` (e.g., `config.go.md` vs `config.md`).

### 2. Processing order — bottom-up by depth level

- Determine all folders containing source files (directly or transitively).
- Group folders by depth (deepest first).
- Process each depth level concurrently (same semaphore + WaitGroup pattern as file analysis).
- Within each level, all folders at that depth can be processed in parallel since they don't depend on each other.
- Move to the next level up only after the current level completes.

### 3. LLM input — full content of children

- For each folder, gather:
  - Full content of per-file `.md` files for direct child source files.
  - Full content of child folder summary `.md` files (already generated in a previous depth level).
- Send all gathered content to Claude CLI with a prompt asking for a high-level synthesis.

### 4. Caching via manifest

- Folder summaries are tracked in the existing manifest (`.manifest.json`).
- Use a distinct key format to avoid collision with file entries: `dir:<relative-path>` (e.g., `dir:internal/config`).
- A folder summary needs regeneration when:
  - Any file within it (recursively) was reprocessed in this run.
  - Any child folder summary was regenerated in this run.
  - The summary file doesn't exist on disk.
  - The `--force` flag is set.
- The "changed" signal propagates upward: if a leaf folder is regenerated, all ancestor folders are also regenerated.

### 5. New `internal/summarizer` package

- Clean separation from the existing `internal/analyzer` package.
- Follows the project's `internal/` package layout convention.
- Exposes a `Run(ctx, repoPath, files, manifest, workers, force)` function matching the analyzer pattern.
- Called as a new stage in `cmdGenerate` after `analyzer.Run` and before `analyzer.CleanStale`.

### 6. Stale cleanup

- When a folder no longer contains any source files (all files removed), its summary should be deleted.
- Extend or complement `analyzer.CleanStale` to handle folder summary files.

## Open Questions

None — all resolved.

## Resolved Questions

### 1. Large folder prompt size

**Decision:** No limit. Trust Claude CLI's context window to handle it. Most project folders won't exceed it, and this keeps the implementation simple.

### 2. Folder summary output structure

**Decision:** Free-form. Let the LLM decide the best structure for each folder based on its contents. More natural and adaptive than enforcing a rigid template.
