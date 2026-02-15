package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	callTimeout = 120 * time.Second
	maxRetries  = 2
)

// CheckInstalled verifies that the claude CLI is available in PATH.
func CheckInstalled() error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH; install it from https://docs.anthropic.com/en/docs/claude-code/overview")
	}
	slog.Debug("claude CLI found", "path", path)
	return nil
}

// ClassificationResult holds the LLM's response for extension classification.
type ClassificationResult struct {
	SourceExtensions    []string `json:"source_extensions"`
	NonSourceExtensions []string `json:"non_source_extensions"`
}

// cliEnvelope is the JSON envelope that Claude CLI wraps responses in
// when using --output-format json.
type cliEnvelope struct {
	Result string `json:"result"`
}

// ClassifyExtensions sends a list of file extensions to Claude CLI and returns
// the classification of which are source code.
func ClassifyExtensions(ctx context.Context, extensions []string) (*ClassificationResult, error) {
	prompt := fmt.Sprintf(`Given the following list of file extensions found in a source code repository, return a JSON object with two arrays:

1. "source_extensions": extensions that are programming language source code files (code that defines functions, classes, methods, structs, etc.)
2. "non_source_extensions": everything else (configs, data, docs, binaries, etc.)

Be inclusive — if an extension could be source code, include it in source_extensions.

Do NOT wrap your response in markdown code fences. Return only the raw JSON object.

Extensions found:
%s`, strings.Join(extensions, "\n"))

	slog.Debug("sending classification prompt", "extension_count", len(extensions), "prompt_length", len(prompt))

	output, err := callWithRetry(ctx, prompt, "json")
	if err != nil {
		return nil, fmt.Errorf("classify extensions: %w", err)
	}

	slog.Debug("classification raw response", "output", output)

	// Claude CLI --output-format json wraps the LLM response in an envelope:
	// {"type":"result","result":"<LLM text>", ...}
	// Extract the inner result field.
	inner, err := extractResult(output)
	if err != nil {
		return nil, fmt.Errorf("extract result from CLI response: %w", err)
	}

	slog.Debug("classification inner result", "inner", inner)

	// The LLM may wrap JSON in markdown code fences; strip them.
	inner = stripCodeFences(inner)

	slog.Debug("classification stripped", "json", inner)

	var result ClassificationResult
	if err := json.Unmarshal([]byte(inner), &result); err != nil {
		return nil, fmt.Errorf("parse classification response: %w\nraw inner: %s", err, inner)
	}

	slog.Debug("classification parsed",
		"source_count", len(result.SourceExtensions),
		"non_source_count", len(result.NonSourceExtensions))

	return &result, nil
}

// AnalyzeFile sends a source file to Claude CLI for analysis and returns
// the generated markdown.
func AnalyzeFile(ctx context.Context, relPath string, content string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following source code file and generate a structured markdown knowledge base entry. This will be used as context by AI coding agents.

File path: %s

For each symbol (function, method, class, struct, interface, type, constant, variable) in the file, document:
- Name and signature
- Brief description of what it does
- Parameters and return values (if applicable)
- Notable dependencies, side effects, or relationships

Start with a brief file-level summary (1-2 sentences).

Use this exact format:

# <filename>

<file summary>

## Functions

### <function_name>

`+"`"+`<signature>`+"`"+`

<description>

## Types

### <type_name>

`+"`"+`<definition>`+"`"+`

<description>

(Use appropriate sections: Functions, Types, Methods, Constants, Variables, etc. Omit empty sections.)

Source code:
`+"```"+`
%s
`+"```", relPath, content)

	slog.Debug("analyzing file", "path", relPath, "content_length", len(content), "prompt_length", len(prompt))

	output, err := callWithRetry(ctx, prompt, "text")
	if err != nil {
		return "", fmt.Errorf("analyze file %s: %w", relPath, err)
	}

	// Claude CLI --output-format text also wraps in a JSON envelope.
	// Extract the inner result.
	inner, err := extractResult(output)
	if err != nil {
		// If extraction fails, the output might already be plain text.
		slog.Debug("could not extract envelope, using raw output", "error", err)
		inner = output
	}

	slog.Debug("analysis complete", "path", relPath, "output_length", len(inner))
	return inner, nil
}

// extractResult parses the Claude CLI JSON envelope and returns the inner
// result string. If the output is not a JSON envelope, returns an error.
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

// stripCodeFences removes markdown code fences (```json ... ``` or ``` ... ```)
// from around content.
var codeFenceRe = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\n?(.*?)\\s*```\\s*$")

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if m := codeFenceRe.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return s
}

func callWithRetry(ctx context.Context, prompt, outputFormat string) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			slog.Debug("retrying claude CLI call", "attempt", attempt+1, "delay", delay)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		output, err := call(ctx, prompt, outputFormat)
		if err == nil {
			return output, nil
		}
		lastErr = err
		slog.Debug("claude CLI call failed", "attempt", attempt+1, "error", err)
	}

	return "", fmt.Errorf("after %d attempts: %w", maxRetries+1, lastErr)
}

func call(ctx context.Context, prompt, outputFormat string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	args := []string{"-p", prompt, "--output-format", outputFormat}
	cmd := exec.CommandContext(ctx, "claude", args...)

	// Remove CLAUDECODE env var so the subprocess doesn't refuse to start
	// when akb is run from within a Claude Code session.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	slog.Debug("executing claude CLI", "output_format", outputFormat, "timeout", callTimeout)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			slog.Debug("claude CLI stderr", "stderr", stderr, "exit_code", exitErr.ExitCode())
			return "", fmt.Errorf("claude CLI exited with code %d: %s", exitErr.ExitCode(), stderr)
		}
		return "", fmt.Errorf("run claude CLI: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// filterEnv returns a copy of env with the named variable removed.
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
