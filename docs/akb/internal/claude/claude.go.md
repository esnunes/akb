# internal/claude/claude.go

Wrapper around the `claude` CLI subprocess that provides extension classification and file analysis capabilities, handling JSON envelope parsing, markdown code fence stripping, retries with exponential backoff, and environment sanitization.

## Constants

### callTimeout

`callTimeout = 120 * time.Second`

Maximum duration for a single `claude` CLI invocation before the context is cancelled.

### maxRetries

`maxRetries = 2`

Number of additional attempts after an initial failure (3 total attempts).

## Functions

### CheckInstalled

`func CheckInstalled() error`

Verifies that the `claude` CLI binary is available in `PATH` using `exec.LookPath`. Returns a descriptive error with install URL if not found.

### ClassifyExtensions

`func ClassifyExtensions(ctx context.Context, extensions []string) (*ClassificationResult, error)`

Sends a list of file extensions to Claude CLI (`--output-format json`) and returns a parsed `ClassificationResult` categorizing them as source or non-source. Handles the full response pipeline: retry, envelope extraction, code fence stripping, and JSON unmarshalling.

- **Parameters:** `ctx` — parent context; `extensions` — list of file extensions (e.g., `".go"`, `".yaml"`)
- **Returns:** parsed classification or error
- **Calls:** `callWithRetry`, `extractResult`, `stripCodeFences`

### AnalyzeFile

`func AnalyzeFile(ctx context.Context, relPath string, content string) (string, error)`

Sends a source file's content to Claude CLI (`--output-format text`) and returns generated markdown documentation. Falls back to raw output if the JSON envelope extraction fails.

- **Parameters:** `ctx` — parent context; `relPath` — relative file path for the prompt; `content` — full file source code
- **Returns:** markdown string or error
- **Calls:** `callWithRetry`, `extractResult`

### extractResult

`func extractResult(output string) (string, error)`

Parses the Claude CLI JSON envelope (`{"type":"result","result":"..."}`) and returns the inner `result` string. Returns an error if the output is not valid JSON or the result field is empty.

### stripCodeFences

`func stripCodeFences(s string) string`

Removes markdown code fences (`` ```json ... ``` `` or `` ``` ... ``` ``) wrapping content. Uses the package-level `codeFenceRe` regex. Returns the input unchanged if no fences are found.

### callWithRetry

`func callWithRetry(ctx context.Context, prompt, outputFormat string) (string, error)`

Wraps `call` with retry logic — up to `maxRetries` additional attempts with exponential backoff (2^attempt seconds). Respects context cancellation between retries.

- **Parameters:** `prompt` — the full prompt text; `outputFormat` — `"json"` or `"text"`
- **Returns:** trimmed CLI output or error after all attempts exhausted

### call

`func call(ctx context.Context, prompt, outputFormat string) (string, error)`

Executes `claude -p <prompt> --output-format <outputFormat>` as a subprocess with a `callTimeout` deadline. Filters `CLAUDECODE` from the environment to prevent conflicts when run inside a Claude Code session. Captures stderr on failure for diagnostics.

- **Side effects:** spawns a subprocess
- **Calls:** `filterEnv`

### filterEnv

`func filterEnv(env []string, name string) []string`

Returns a copy of the environment variable slice with the named variable removed. Matches by prefix (`name=`).

## Types

### ClassificationResult

```go
type ClassificationResult struct {
    SourceExtensions    []string `json:"source_extensions"`
    NonSourceExtensions []string `json:"non_source_extensions"`
}
```

Holds the LLM's classification of file extensions into source code vs. non-source categories. Populated by `ClassifyExtensions`.

### cliEnvelope

```go
type cliEnvelope struct {
    Result string `json:"result"`
}
```

Internal type representing the JSON envelope that Claude CLI wraps responses in when using `--output-format json`. Used by `extractResult`.

## Variables

### codeFenceRe

`` var codeFenceRe = regexp.MustCompile(`(?s)^\s*` + "```" + `(?:json)?\s*\n?(.*?)\s*` + "```" + `\s*$`) ``

Compiled regex matching markdown code fences (with optional `json` language tag) around content. Used by `stripCodeFences`.
