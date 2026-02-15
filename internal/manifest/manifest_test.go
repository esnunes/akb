package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	m := Manifest{
		"src/foo.go": "sha256:abc123",
		"src/bar.py": "sha256:def456",
	}

	if err := Save(dir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("got %d entries, want 2", len(loaded))
	}

	if loaded["src/foo.go"] != "sha256:abc123" {
		t.Errorf("unexpected hash for src/foo.go: %q", loaded["src/foo.go"])
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load missing should not error: %v", err)
	}

	if len(m) != 0 {
		t.Errorf("expected empty manifest, got %d entries", len(m))
	}
}

func TestAtomicSave(t *testing.T) {
	dir := t.TempDir()

	m := Manifest{"a.go": "sha256:111"}
	if err := Save(dir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify no temp file left behind.
	tmp := Path(dir) + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.go")
	os.WriteFile(p, []byte("package main\n"), 0644)

	hash, err := HashFile(p)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	if hash[:7] != "sha256:" {
		t.Errorf("hash should have sha256: prefix, got %q", hash)
	}

	if len(hash) != 7+64 { // "sha256:" + 64 hex chars
		t.Errorf("unexpected hash length: %d", len(hash))
	}

	// Same content should produce same hash.
	hash2, _ := HashFile(p)
	if hash != hash2 {
		t.Error("same file should produce same hash")
	}
}

func TestChanged(t *testing.T) {
	m := Manifest{
		"a.go": "sha256:abc",
	}

	if m.Changed("a.go", "sha256:abc") {
		t.Error("same hash should not be changed")
	}

	if !m.Changed("a.go", "sha256:xyz") {
		t.Error("different hash should be changed")
	}

	if !m.Changed("new.go", "sha256:abc") {
		t.Error("new file should be changed")
	}
}

func TestPath(t *testing.T) {
	got := Path("/repo")
	want := filepath.Join("/repo", "docs", "akb", ".manifest.json")
	if got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}
