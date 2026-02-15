# walker_test.go

Test suite for the `walker` package, validating file discovery (`DiscoverExtensions`) and source file walking (`WalkSourceFiles`) including extension filtering, directory exclusion patterns (`.git/`, `node_modules/`, `vendor/`, `docs/akb/`), and config-driven file selection.

## Functions

### TestDiscoverExtensions

`func TestDiscoverExtensions(t *testing.T)`

Tests that `DiscoverExtensions` correctly identifies unique file extensions from a directory tree. Creates a temp directory with `.go`, `.py`, `.yaml`, `.ts`, and `Makefile` files, plus files in `.git/` and `node_modules/`. Asserts that source extensions are discovered while `.git` and `node_modules` contents are excluded.

- **Parameters:** `t *testing.T` — standard test context
- **Dependencies:** `DiscoverExtensions` from the `walker` package
- **Key assertions:** Extensions `.go`, `.py`, `.yaml`, `.ts`, and `Makefile` are present; `.js` from `node_modules/` is excluded

### TestWalkSourceFiles

`func TestWalkSourceFiles(t *testing.T)`

Tests that `WalkSourceFiles` returns only files matching configured source extensions while respecting exclude patterns. Uses a `config.Config` with `SourceExtensions: [".go"]` and `ExcludePatterns: [".git/", "vendor/"]`.

- **Parameters:** `t *testing.T` — standard test context
- **Dependencies:** `WalkSourceFiles` from `walker` package, `config.Config` from `github.com/esnunes/akb/internal/config`
- **Key assertions:** Finds `main.go` and `lib/util.go`; excludes `vendor/dep.go` (exclude pattern) and `config.yaml` (wrong extension); exactly 2 results

### TestDiscoverExtensionsIncludesDocs

`func TestDiscoverExtensionsIncludesDocs(t *testing.T)`

Tests that `DiscoverExtensions` walks into `docs/` directories (they are not excluded during discovery). Verifies `.go`, `.md`, and `.js` extensions are all found.

- **Parameters:** `t *testing.T` — standard test context
- **Dependencies:** `DiscoverExtensions`
- **Key assertions:** `docs/` directory is not skipped; all three extensions (`.go`, `.md`, `.js`) are present

### TestExcludeDocsAkb

`func TestExcludeDocsAkb(t *testing.T)`

Tests that `WalkSourceFiles` always excludes the `docs/akb/` directory (the tool's own output directory) even when no explicit exclude pattern covers it. Creates `docs/akb/main.go.md` alongside `main.go`.

- **Parameters:** `t *testing.T` — standard test context
- **Dependencies:** `WalkSourceFiles`, `config.Config`
- **Key assertions:** `docs/akb/main.go.md` is excluded from results; `main.go` is included
