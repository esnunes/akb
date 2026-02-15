# manifest.go

Manages a JSON-based manifest that tracks SHA-256 content hashes for repository files, enabling incremental processing by detecting which files have changed since the last run.

## Types

### Manifest

`type Manifest map[string]string`

Maps relative file paths to their SHA-256 content hashes (prefixed with `sha256:`). Used to track file changes between `generate` runs.

## Functions

### Path

`func Path(repoPath string) string`

Returns the absolute path to the manifest file at `<repoPath>/docs/akb/.manifest.json`.

- **Parameters:** `repoPath` — root path of the repository
- **Returns:** full path to the manifest JSON file

### Load

`func Load(repoPath string) (Manifest, error)`

Reads and parses the manifest file from disk. Returns an empty `Manifest` if the file does not exist (non-error case), allowing first-run scenarios.

- **Parameters:** `repoPath` — root path of the repository
- **Returns:** parsed `Manifest`, or error if the file exists but cannot be read/parsed
- **Dependencies:** `Path()` for file location, `encoding/json` for deserialization

### Save

`func Save(repoPath string, m Manifest) error`

Writes the manifest to disk atomically using a temp file + `os.Rename` pattern. Creates the parent directory if needed. Output is pretty-printed JSON with a trailing newline.

- **Parameters:** `repoPath` — root path of the repository; `m` — manifest to persist
- **Returns:** error if any step (mkdir, marshal, write, rename) fails
- **Side effects:** creates `docs/akb/` directory if absent; cleans up temp file on rename failure

### HashFile

`func HashFile(path string) (string, error)`

Computes the SHA-256 hash of a file's contents, returned as a `sha256:`-prefixed hex string.

- **Parameters:** `path` — absolute or relative path to the file to hash
- **Returns:** hash string (e.g. `sha256:abc123...`), or error if the file cannot be read

## Methods

### Manifest.Changed

`func (m Manifest) Changed(relPath, hash string) bool`

Returns `true` if the file is new (not in the manifest) or its hash differs from the stored value. Used to skip re-processing unchanged files during `generate`.

- **Parameters:** `relPath` — relative file path as manifest key; `hash` — current hash to compare
- **Returns:** `true` if the file needs reprocessing
