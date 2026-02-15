# internal/manifest/

Provides incremental processing support for `akb generate` by tracking SHA-256 content hashes of repository files in a JSON manifest, so only changed or new files are re-analyzed between runs.

## Key Components

- **Manifest type** (`map[string]string`) — maps relative file paths to `sha256:`-prefixed hex hashes, serving as the change-detection store.
- **Load/Save** — reads and writes `.manifest.json` at `<repo>/docs/akb/.manifest.json`, with `Save` using atomic temp-file + `os.Rename` for crash safety. `Load` gracefully handles first-run (missing file) by returning an empty manifest.
- **HashFile** — computes SHA-256 of a file's contents for comparison against stored hashes.
- **Changed** — compares a file's current hash against the manifest to determine if reprocessing is needed.

## Role in the Project

The `generate` subcommand uses this package to skip unchanged files, avoiding redundant Claude CLI calls. Before analyzing a file, it hashes the content and checks `Changed`; after a successful run, the updated manifest is saved so the next invocation picks up only what's new or modified.
