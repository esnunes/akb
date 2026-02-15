package walker

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/esnunes/akb/internal/config"
)

func TestDiscoverExtensions(t *testing.T) {
	dir := t.TempDir()

	// Create test files.
	files := map[string]string{
		"main.go":          "package main",
		"lib/util.py":      "def foo(): pass",
		"lib/helper.py":    "def bar(): pass",
		"config.yaml":      "key: value",
		"Makefile":         "all:",
		"src/app.ts":       "const x = 1",
		".git/HEAD":        "ref: refs/heads/main",
		"node_modules/x.js": "module.exports = {}",
	}

	for name, content := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}

	exts, err := DiscoverExtensions(dir)
	if err != nil {
		t.Fatalf("DiscoverExtensions: %v", err)
	}

	sort.Strings(exts)

	// Should include .go, .py, .yaml, .ts, Makefile
	// Should NOT include .git or node_modules contents.
	extSet := make(map[string]struct{})
	for _, e := range exts {
		extSet[e] = struct{}{}
	}

	for _, want := range []string{".go", ".py", ".yaml", ".ts", "Makefile"} {
		if _, ok := extSet[want]; !ok {
			t.Errorf("expected %q in extensions, got %v", want, exts)
		}
	}

	// .js from node_modules should be excluded.
	if _, ok := extSet[".js"]; ok {
		t.Errorf("should not include .js from node_modules")
	}
}

func TestWalkSourceFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"main.go":       "package main",
		"lib/util.go":   "package lib",
		"config.yaml":   "key: value",
		"vendor/dep.go": "package dep",
		"README.md":     "# readme",
	}

	for name, content := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}

	cfg := &config.Config{
		SourceExtensions: []string{".go"},
		ExcludePatterns:  []string{".git/", "vendor/"},
	}

	result, err := WalkSourceFiles(dir, cfg)
	if err != nil {
		t.Fatalf("WalkSourceFiles: %v", err)
	}

	// Should find main.go and lib/util.go, not vendor/dep.go or config.yaml.
	relPaths := make(map[string]struct{})
	for _, f := range result {
		relPaths[f.RelPath] = struct{}{}
	}

	if _, ok := relPaths["main.go"]; !ok {
		t.Error("expected main.go")
	}
	if _, ok := relPaths[filepath.Join("lib", "util.go")]; !ok {
		t.Error("expected lib/util.go")
	}
	if _, ok := relPaths[filepath.Join("vendor", "dep.go")]; ok {
		t.Error("should exclude vendor/dep.go")
	}
	if _, ok := relPaths["config.yaml"]; ok {
		t.Error("should exclude config.yaml (not a source extension)")
	}

	if len(result) != 2 {
		t.Errorf("expected 2 files, got %d", len(result))
	}
}

func TestDiscoverExtensionsIncludesDocs(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"main.go":         "package main",
		"docs/guide.md":   "# Guide",
		"docs/script.js":  "console.log('hello')",
	}

	for name, content := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}

	exts, err := DiscoverExtensions(dir)
	if err != nil {
		t.Fatalf("DiscoverExtensions: %v", err)
	}

	extSet := make(map[string]struct{})
	for _, e := range exts {
		extSet[e] = struct{}{}
	}

	// docs/ should NOT be skipped during discovery.
	for _, want := range []string{".go", ".md", ".js"} {
		if _, ok := extSet[want]; !ok {
			t.Errorf("expected %q in extensions (docs/ should be walked), got %v", want, exts)
		}
	}
}

func TestExcludeDocsAkb(t *testing.T) {
	dir := t.TempDir()

	// Create a file inside docs/akb that matches a source extension.
	p := filepath.Join(dir, "docs", "akb", "main.go.md")
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, []byte("# main.go"), 0644)

	// Also create a real source file.
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	cfg := &config.Config{
		SourceExtensions: []string{".go", ".md"},
		ExcludePatterns:  []string{},
	}

	result, err := WalkSourceFiles(dir, cfg)
	if err != nil {
		t.Fatalf("WalkSourceFiles: %v", err)
	}

	for _, f := range result {
		if f.RelPath == filepath.Join("docs", "akb", "main.go.md") {
			t.Error("should exclude files in docs/akb/")
		}
	}
}
