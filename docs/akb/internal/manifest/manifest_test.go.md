# manifest_test.go

Test suite for the `manifest` package, validating Save/Load round-tripping, atomic file writes, SHA-256 file hashing, change detection, and manifest path construction.

## Functions

### TestSaveAndLoad

`func TestSaveAndLoad(t *testing.T)`

Verifies that a `Manifest` can be saved to a temp directory and loaded back with identical entries. Asserts correct entry count and hash value for a known key.

- **Dependencies:** `Save`, `Load`, `Manifest`

### TestLoadMissing

`func TestLoadMissing(t *testing.T)`

Verifies that `Load` returns an empty manifest (not an error) when no manifest file exists on disk.

- **Dependencies:** `Load`

### TestAtomicSave

`func TestAtomicSave(t *testing.T)`

Verifies atomic write behavior of `Save` by checking that no `.tmp` file remains after a successful save. Uses `Path` to derive the expected temp file location.

- **Dependencies:** `Save`, `Path`, `os.Stat`

### TestHashFile

`func TestHashFile(t *testing.T)`

Validates `HashFile` output format: checks for `sha256:` prefix, correct total length (7 + 64 hex chars), and determinism (same content produces same hash).

- **Dependencies:** `HashFile`, `os.WriteFile`

### TestChanged

`func TestChanged(t *testing.T)`

Tests the `Manifest.Changed` method for three cases: same hash (not changed), different hash (changed), and new file not in manifest (changed).

- **Dependencies:** `Manifest.Changed`

### TestPath

`func TestPath(t *testing.T)`

Asserts that `Path("/repo")` returns `"/repo/docs/akb/.manifest.json"`.

- **Dependencies:** `Path`, `filepath.Join`
