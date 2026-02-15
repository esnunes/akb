# summarizer_test.go

Test suite for the `summarizer` package's helper functions, covering folder collection, depth-based grouping, output path generation, dirty-set computation, depth calculation, and manifest key encoding/decoding.

## Functions

### TestCollectFolders

`func TestCollectFolders(t *testing.T)`

Table-driven test for `collectFolders`. Verifies that given a list of `walker.FileInfo`, the function returns all unique ancestor folder paths (including `"."` for root). Covers single files at root, nested folders, deeply nested paths, and empty input.

**Dependencies:** `walker.FileInfo`, `collectFolders`

### TestGroupByDepth

`func TestGroupByDepth(t *testing.T)`

Tests `groupByDepth` with a set of folders at varying depths (0–3). Asserts that the returned groups are ordered deepest-first and contain the correct folders at each depth level.

**Dependencies:** `groupByDepth`

### TestOutputPath

`func TestOutputPath(t *testing.T)`

Table-driven test for `outputPath`. Verifies the mapping from a folder relative path to its summary file location under `docs/akb/`. Special case: root folder `"."` maps to `README.md`; other folders become `<folder>.md` files.

**Dependencies:** `outputPath`

### TestBuildDirtySet

`func TestBuildDirtySet(t *testing.T)`

Tests `buildDirtySet` across four scenarios:
- **Processed file marks folder and ancestors dirty** — a file in `internal/walker` dirties that folder plus `internal` and `"."`, but not `internal/config` (which has an existing summary).
- **Missing summary marks folder dirty** — folders without summary files on disk are dirty, propagating to ancestors.
- **Force marks all dirty** — when force flag is `true`, every folder is dirty regardless of state.
- **Nothing processed and all summaries exist** — no folders are dirty when all summaries are present and no files were processed.

Uses `t.TempDir()` to create temporary summary files on disk. Cleans up walker summary at the end to avoid leaking state between subtests.

**Dependencies:** `buildDirtySet`, `outputPath`, `os.MkdirAll`, `os.WriteFile`, `os.Remove`

### TestDepth

`func TestDepth(t *testing.T)`

Table-driven test for `depth`. Verifies that `"."` has depth 0, single-segment paths have depth 1, and multi-segment paths count separators correctly.

**Dependencies:** `depth`

### TestManifestKey

`func TestManifestKey(t *testing.T)`

Tests the `manifestKey` / `fromManifestKey` round-trip encoding. Verifies that folder paths are prefixed with `"dir:"` and that `fromManifestKey` correctly parses valid keys and rejects non-dir keys.

**Dependencies:** `manifestKey`, `fromManifestKey`
