# config_test.go

Test suite for the `config` package, verifying YAML config file save/load round-tripping, missing file handling, validation of empty extensions, file existence checks, and config path construction.

## Functions

### TestSaveAndLoad

`func TestSaveAndLoad(t *testing.T)`

Round-trip test: creates a `Config` with sample extensions and exclude patterns, saves it to a temp directory via `Save`, reloads it via `Load`, and asserts the loaded values match. Also verifies the saved file starts with a `# Generated` header comment by reading raw bytes from `Path(dir)`.

- Depends on: `Save`, `Load`, `Path`, `Config`

### TestLoadMissing

`func TestLoadMissing(t *testing.T)`

Verifies that `Load` returns an error when no config file exists in the given directory.

- Depends on: `Load`

### TestLoadEmptyExtensions

`func TestLoadEmptyExtensions(t *testing.T)`

Verifies that `Load` returns a validation error when `SourceExtensions` is empty. Saves a config with empty slices first, then asserts `Load` rejects it.

- Depends on: `Save`, `Load`, `Config`

### TestExists

`func TestExists(t *testing.T)`

Verifies `Exists` returns `false` before saving and `true` after saving a config file.

- Depends on: `Exists`, `Save`, `Config`

### TestPath

`func TestPath(t *testing.T)`

Verifies `Path("/repo")` returns `"/repo/docs/akb/.config.yaml"`. Confirms the config file location convention used throughout the project.

- Depends on: `Path`
