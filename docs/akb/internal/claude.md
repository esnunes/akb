# internal/claude/

LLM integration layer that wraps the `claude` CLI subprocess to provide extension classification, file analysis, and folder summarization capabilities. Serves as the sole interface between the akb tool and Claude's language model.

## Key Responsibilities

- **Extension classification** (`ClassifyExtensions`): Asks the LLM to categorize file extensions as source vs. non-source code, returning structured JSON. Used by the `init` subcommand.
- **File analysis** (`AnalyzeFile`): Sends source file content to the LLM and returns structured markdown documentation. Used by the `generate` subcommand.
- **Folder summarization** (`SummarizeFolder`): Aggregates child documentation and produces a high-level markdown summary of a folder's purpose and architecture.

## Architecture

All three public functions follow the same pipeline:

1. `callWithRetry` — invokes the `claude` CLI subprocess with exponential backoff (up to 3 attempts)
2. `call` — executes `claude -p <prompt> --output-format <json|text>`, filtering the `CLAUDECODE` env var to avoid conflicts when running inside Claude Code
3. `extractResult` — parses the CLI's JSON envelope (`{"type":"result","result":"..."}`) to extract the actual LLM output
4. `stripCodeFences` — removes markdown code fences the LLM may wrap around JSON responses (classification path only)

JSON-format responses (`ClassifyExtensions`) go through the full pipeline including code fence stripping and JSON unmarshalling. Text-format responses (`AnalyzeFile`, `SummarizeFolder`) fall back to raw output if envelope extraction fails.

## Files

- **claude.go** — All production code: types, public API, CLI subprocess management, retry logic, and response parsing.
- **claude_test.go** — Table-driven unit tests for `extractResult` and `stripCodeFences`, covering valid envelopes, escaped JSON, non-JSON input, empty results, and various code fence formats.
