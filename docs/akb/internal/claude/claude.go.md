# internal/claude/claude.go

Wrapper around the `claude` CLI subprocess that provides extension classification, file analysis, and folder summarization via LLM calls, with retry logic and JSON envelope parsing.

## Constants

### callTimeout

`callTimeout = 120 * time.Second`

Maximum duration for a single `claude` CLI invocation before the context is cancelled.

### maxRetries

`maxRetries = 2`

Number of retry attempts after the initial call fails (total attempts = 3).

## Types

### ClassificationResult

```go
type ClassificationResult struct {
    SourceExtensions    []string `json:"source_extensions"`
    NonSourceExtensions []string `json:"non_source_extensions"`
}
```

Holds the LLM's response when classifying file extensions into source code vs. non-source categories. Used by the `init` subcommand.

### cliEnvelope

```go
type cliEnvelope struct {
    Result string `json:"result"`
}
```

Internal type representing the JSON envelope that Claude CLI wraps responses in when using `--output-format json` or `--output-format text`. The `Result` field contains the actual LLM output as a string.

## Functions

### CheckInstalled

`func CheckInstalled() error`

Verifies that the `claude` CLI binary is available in `PATH` using `exec.LookPath`. Returns a descriptive error with installation link if not found.

### ClassifyExtensions

`func ClassifyExtensions(ctx context.Context, extensions []string) (*ClassificationResult, error)`

Sends a list of file extensions to Claude CLI and returns a `ClassificationResult` classifying them as source or non-source. Uses `--output-format json`. Handles the full pipeline: call with retry, extract result from CLI envelope, strip markdown code fences, and unmarshal JSON.

- **Parameters:** `ctx` — context for cancellation; `extensions` — list of file extensions (e.g., `.go`, `.yaml`)
- **Returns:** parsed `*ClassificationResult` or error
- **Calls:** `callWithRetry`, `extractResult`, `stripCodeFences`

### AnalyzeFile

`func AnalyzeFile(ctx context.Context, relPath string, content string) (string, error)`

Sends a source file's content to Claude CLI for analysis and returns structured markdown documentation. Uses `--output-format text`. Falls back to raw output if envelope extraction fails.

- **Parameters:** `ctx` — context; `relPath` — relative file path for the prompt; `content` — full file source code
- **Returns:** markdown string or error
- **Calls:** `callWithRetry`, `extractResult`

### SummarizeFolder

`func SummarizeFolder(ctx context.Context, folderPath string, childrenContent string) (string, error)`

Sends aggregated per-file and subfolder documentation to Claude CLI and returns a high-level markdown summary of the folder's purpose and architecture. Uses `--output-format text`. Falls back to raw output if envelope extraction fails.

- **Parameters:** `ctx` — context; `folderPath` — folder path for the prompt; `childrenContent` — concatenated documentation of children
- **Returns:** markdown summary string or error
- **Calls:** `callWithRetry`, `extractResult`

### extractResult

`func extractResult(output string) (string, error)`

Parses the Claude CLI JSON envelope and returns the inner `result` string. Returns an error if the output is not valid JSON or if the `result` field is empty.

- **Parameters:** `output` — raw CLI stdout
- **Returns:** inner result string or error

### stripCodeFences

`func stripCodeFences(s string) string`

Removes markdown code fences (`` ```json ... ``` `` or `` ``` ... ``` ``) wrapping content. Uses the package-level `codeFenceRe` regex. Returns the input unchanged if no fences are found.

### callWithRetry

`func callWithRetry(ctx context.Context, prompt, outputFormat string) (string, error)`

Wraps `call` with exponential backoff retry logic (2^attempt seconds delay). Respects context cancellation between retries. Makes up to `maxRetries + 1` total attempts.

- **Parameters:** `ctx` — context; `prompt` — LLM prompt text; `outputFormat` — `"json"` or `"text"`
- **Returns:** trimmed CLI output or error after all retries exhausted

### call

`func call(ctx context.Context, prompt, outputFormat string) (string, error)`

Executes the `claude` CLI as a subprocess with `-p <prompt> --output-format <format>`. Applies `callTimeout` via context, filters the `CLAUDECODE` environment variable to avoid conflicts when run inside Claude Code, and captures stderr on failure for diagnostics.

- **Parameters:** `ctx` — context; `prompt` — LLM prompt; `outputFormat` — CLI output format flag
- **Returns:** trimmed stdout string or error with exit code and stderr details
- **Side effects:** spawns subprocess, reads environment variables

### filterEnv

`func filterEnv(env []string, name string) []string`

Returns a copy of the environment variable slice with the named variable removed. Used to strip `CLAUDECODE` from the subprocess environment.

- **Parameters:** `env` — slice of `KEY=VALUE` strings; `name` — variable name to remove
- **Returns:** filtered copy of the slice

## Variables

### codeFenceRe

`` var codeFenceRe = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\n?(.*?)\\s*```\\s*$") ``

Compiled regex matching markdown code fences with optional `json` language tag. Used by `stripCodeFences`.
