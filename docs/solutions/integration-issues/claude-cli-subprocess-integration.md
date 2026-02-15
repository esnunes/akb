---
title: "Claude CLI Subprocess Integration: Envelope Parsing, Env Vars, and Directory Walking"
date: 2026-02-13
category: integration-issues
tags:
  - claude-cli
  - json-parsing
  - subprocess
  - environment-variables
  - directory-traversal
components:
  - internal/claude/claude.go
  - internal/walker/walker.go
severity: high
symptoms:
  - "akb init generates config with empty source_extensions despite Claude CLI correctly classifying extensions"
  - "Claude CLI exits with 'cannot be launched inside another Claude Code session' when akb runs within Claude Code"
  - "File extensions from docs/ directory not discovered during init phase"
---

# Claude CLI Subprocess Integration Issues

Three related bugs in the akb CLI's integration with Claude CLI as a subprocess.

## Bug 1: JSON Envelope Not Unwrapped

### Symptom

`akb init` writes `.config.yaml` with `source_extensions: []` even though Claude CLI correctly identifies source code extensions in its response.

### Root Cause

Claude CLI's `--output-format json` wraps the LLM response in a JSON envelope:

```json
{"type":"result","subtype":"success","result":"```json\n{\"source_extensions\":[\".go\"]}\n```","duration_ms":3189}
```

The code parsed the outer envelope directly into `ClassificationResult`. Go's `json.Unmarshal` silently ignores unknown fields, so it succeeded with zero-value (empty) arrays.

### Solution

Added `extractResult()` to unwrap the envelope and `stripCodeFences()` to remove markdown fences:

```go
type cliEnvelope struct {
    Result string `json:"result"`
}

func extractResult(output string) (string, error) {
    var env cliEnvelope
    if err := json.Unmarshal([]byte(output), &env); err != nil {
        return "", fmt.Errorf("not a CLI envelope: %w", err)
    }
    if env.Result == "" {
        return "", fmt.Errorf("CLI envelope has empty result field")
    }
    return env.Result, nil
}

var codeFenceRe = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\n?(.*?)\\s*```\\s*$")

func stripCodeFences(s string) string {
    s = strings.TrimSpace(s)
    if m := codeFenceRe.FindStringSubmatch(s); m != nil {
        return strings.TrimSpace(m[1])
    }
    return s
}
```

Applied to both `ClassifyExtensions` (JSON output) and `AnalyzeFile` (text output, with fallback to raw).

### Key Lesson

Go's `json.Unmarshal` silently produces zero-value structs when fields don't match. Always validate parsed results are non-empty, or use `json.Decoder` with `DisallowUnknownFields()` when strict parsing is needed.

---

## Bug 2: Nested Session Environment Variable

### Symptom

Claude CLI exits with: "Error: Claude Code cannot be launched inside another Claude Code session."

### Root Cause

When akb runs inside a Claude Code session, the `CLAUDECODE` environment variable is set. The subprocess inherits it, causing Claude CLI to detect a "nested session" and abort.

### Solution

Filter out `CLAUDECODE` when spawning the subprocess:

```go
cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

func filterEnv(env []string, name string) []string {
    prefix := name + "="
    filtered := make([]string, 0, len(env))
    for _, e := range env {
        if !strings.HasPrefix(e, prefix) {
            filtered = append(filtered, e)
        }
    }
    return filtered
}
```

### Key Lesson

When shelling out to CLI tools, be aware that parent environment variables are inherited. Tools may use env vars for session detection, and inherited values can cause unexpected failures. Explicitly manage the subprocess environment.

---

## Bug 3: Directory Exclusion Too Broad in Discovery

### Symptom

Extensions from files under `docs/` (e.g., `.html`, `.js`, `.css`) not found during `akb init`.

### Root Cause

`DiscoverExtensions` hardcoded `name == "docs"` in its skip list, preventing traversal of the entire `docs/` directory. Only `docs/akb` (the output directory) should be excluded, and only during the generate phase.

### Solution

Removed `"docs"` from the discovery skip list. Only `.git`, `node_modules`, and `vendor` are skipped during discovery. The `docs/akb` exclusion is handled separately by `shouldExcludeDir()` during `WalkSourceFiles` (generate phase only).

### Key Lesson

Keep phase-specific exclusions separate. Discovery should be maximally inclusive; generation should apply targeted exclusions. Don't hardcode exclusions that belong in configuration.

---

## Prevention Strategies

1. **Validate parsed results** — After `json.Unmarshal`, check that required fields are non-empty before using the result.
2. **Test against real CLI output** — Create test fixtures from actual Claude CLI responses, not just idealized formats.
3. **Manage subprocess environments explicitly** — Filter or whitelist env vars rather than blindly inheriting.
4. **Separate concerns across phases** — Discovery and generation have different exclusion requirements; don't conflate them.
5. **Add debug logging early** — `slog.Debug` at key decision points (directory skips, JSON parsing, subprocess invocation) makes issues immediately visible with `--verbose`.

## Related Documents

- [Plan: akb CLI Knowledge Base Generator](../../plans/2026-02-13-feat-akb-cli-knowledge-base-generator-plan.md)
- [Brainstorm: akb CLI](../../brainstorms/2026-02-13-akb-cli-brainstorm.md)
