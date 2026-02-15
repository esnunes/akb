# claude_test.go

Unit tests for the `claude` package's helper functions `extractResult` and `stripCodeFences`, which handle parsing Claude CLI JSON envelope responses and stripping markdown code fences from LLM output.

## Functions

### TestExtractResult

`func TestExtractResult(t *testing.T)`

Table-driven test for `extractResult`. Validates four cases:
- Extracting the `result` field from a valid Claude CLI JSON envelope
- Handling escaped JSON embedded within the `result` field
- Returning an error for non-JSON input
- Returning an error when the `result` field is empty

Exercises the critical parsing path: Claude CLI's `--output-format json` wraps responses in `{"type":"result","result":"..."}` and the `result` string must be extracted before further processing.

### TestStripCodeFences

`func TestStripCodeFences(t *testing.T)`

Table-driven test for `stripCodeFences`. Validates five cases:
- Stripping `` ```json `` fenced blocks
- Stripping plain `` ``` `` fenced blocks (no language tag)
- Passing through content that has no fences
- Handling leading whitespace/newlines around fenced blocks
- Preserving multiline content within fences

Covers the scenario where the LLM wraps its JSON response in markdown code fences, which must be removed before `json.Unmarshal`.
