package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manifest maps relative file paths to their SHA-256 content hashes.
type Manifest map[string]string

func Path(repoPath string) string {
	return filepath.Join(repoPath, "docs", "akb", ".manifest.json")
}

func Load(repoPath string) (Manifest, error) {
	data, err := os.ReadFile(Path(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return make(Manifest), nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return m, nil
}

// Save writes the manifest atomically using a temp file + rename.
func Save(repoPath string, m Manifest) error {
	p := Path(repoPath)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write manifest tmp: %w", err)
	}

	if err := os.Rename(tmp, p); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename manifest: %w", err)
	}

	return nil
}

// HashFile computes the SHA-256 hash of a file's content.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file for hashing: %w", err)
	}

	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

// Changed returns true if the file's hash differs from what's in the manifest.
func (m Manifest) Changed(relPath, hash string) bool {
	existing, ok := m[relPath]
	return !ok || existing != hash
}
