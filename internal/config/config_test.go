package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		SourceExtensions: []string{".go", ".py", ".ts"},
		ExcludePatterns:  []string{".git/", "vendor/"},
	}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.SourceExtensions) != 3 {
		t.Errorf("got %d extensions, want 3", len(loaded.SourceExtensions))
	}

	if len(loaded.ExcludePatterns) != 2 {
		t.Errorf("got %d patterns, want 2", len(loaded.ExcludePatterns))
	}

	// Verify file has header comment.
	data, _ := os.ReadFile(Path(dir))
	if got := string(data[:11]); got != "# Generated" {
		t.Errorf("config file should start with header comment, got %q", got)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error loading missing config")
	}
}

func TestLoadEmptyExtensions(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		SourceExtensions: []string{},
		ExcludePatterns:  []string{},
	}
	Save(dir, cfg)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for empty source_extensions")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()

	if Exists(dir) {
		t.Fatal("should not exist yet")
	}

	Save(dir, &Config{SourceExtensions: []string{".go"}})

	if !Exists(dir) {
		t.Fatal("should exist after save")
	}
}

func TestPath(t *testing.T) {
	got := Path("/repo")
	want := filepath.Join("/repo", "docs", "akb", ".config.yaml")
	if got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}
