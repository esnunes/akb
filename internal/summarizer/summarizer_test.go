package summarizer

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/esnunes/akb/internal/walker"
)

func TestCollectFolders(t *testing.T) {
	tests := []struct {
		name  string
		files []walker.FileInfo
		want  []string
	}{
		{
			name: "single file at root",
			files: []walker.FileInfo{
				{RelPath: "main.go"},
			},
			want: []string{"."},
		},
		{
			name: "files in nested folders",
			files: []walker.FileInfo{
				{RelPath: "main.go"},
				{RelPath: filepath.Join("internal", "config", "config.go")},
				{RelPath: filepath.Join("internal", "walker", "walker.go")},
			},
			want: []string{
				".",
				"internal",
				filepath.Join("internal", "config"),
				filepath.Join("internal", "walker"),
			},
		},
		{
			name: "deeply nested",
			files: []walker.FileInfo{
				{RelPath: filepath.Join("a", "b", "c", "d.go")},
			},
			want: []string{
				".",
				"a",
				filepath.Join("a", "b"),
				filepath.Join("a", "b", "c"),
			},
		},
		{
			name:  "empty file list",
			files: []walker.FileInfo{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectFolders(tt.files)
			sort.Strings(got)
			sort.Strings(tt.want)

			if len(got) != len(tt.want) {
				t.Fatalf("collectFolders() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("collectFolders()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGroupByDepth(t *testing.T) {
	folders := []string{
		".",
		"internal",
		filepath.Join("internal", "config"),
		filepath.Join("internal", "walker"),
		filepath.Join("internal", "walker", "deep"),
	}

	groups := groupByDepth(folders)

	// Should be 4 groups: depth 3, 2, 1, 0.
	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d: %v", len(groups), groups)
	}

	// First group (deepest) should contain internal/walker/deep.
	if len(groups[0]) != 1 || groups[0][0] != filepath.Join("internal", "walker", "deep") {
		t.Errorf("group[0] = %v, want [internal/walker/deep]", groups[0])
	}

	// Second group should contain internal/config and internal/walker.
	sort.Strings(groups[1])
	wantG1 := []string{filepath.Join("internal", "config"), filepath.Join("internal", "walker")}
	sort.Strings(wantG1)
	if len(groups[1]) != 2 || groups[1][0] != wantG1[0] || groups[1][1] != wantG1[1] {
		t.Errorf("group[1] = %v, want %v", groups[1], wantG1)
	}

	// Third group should contain internal.
	if len(groups[2]) != 1 || groups[2][0] != "internal" {
		t.Errorf("group[2] = %v, want [internal]", groups[2])
	}

	// Fourth group (shallowest) should contain ".".
	if len(groups[3]) != 1 || groups[3][0] != "." {
		t.Errorf("group[3] = %v, want [.]", groups[3])
	}
}

func TestOutputPath(t *testing.T) {
	absRoot := "/project"

	tests := []struct {
		name   string
		folder string
		want   string
	}{
		{
			name:   "root folder",
			folder: ".",
			want:   filepath.Join(absRoot, "docs", "akb", "README.md"),
		},
		{
			name:   "top-level folder",
			folder: "internal",
			want:   filepath.Join(absRoot, "docs", "akb", "internal.md"),
		},
		{
			name:   "nested folder",
			folder: filepath.Join("internal", "config"),
			want:   filepath.Join(absRoot, "docs", "akb", "internal", "config.md"),
		},
		{
			name:   "deeply nested folder",
			folder: filepath.Join("internal", "walker", "deep"),
			want:   filepath.Join(absRoot, "docs", "akb", "internal", "walker", "deep.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outputPath(absRoot, tt.folder)
			if got != tt.want {
				t.Errorf("outputPath(%q) = %q, want %q", tt.folder, got, tt.want)
			}
		})
	}
}

func TestBuildDirtySet(t *testing.T) {
	absRoot := t.TempDir()

	// Create a summary file for internal/config so it's NOT dirty.
	configSummary := outputPath(absRoot, filepath.Join("internal", "config"))
	os.MkdirAll(filepath.Dir(configSummary), 0755)
	os.WriteFile(configSummary, []byte("# config summary"), 0644)

	// Create root summary.
	rootSummary := outputPath(absRoot, ".")
	os.WriteFile(rootSummary, []byte("# root summary"), 0644)

	// Create internal summary.
	internalSummary := outputPath(absRoot, "internal")
	os.WriteFile(internalSummary, []byte("# internal summary"), 0644)

	folders := []string{
		".",
		"internal",
		filepath.Join("internal", "config"),
		filepath.Join("internal", "walker"),
	}

	t.Run("processed file marks folder and ancestors dirty", func(t *testing.T) {
		processedFiles := []string{filepath.Join("internal", "walker", "walker.go")}
		dirty := buildDirtySet(folders, processedFiles, absRoot, false)

		if !dirty[filepath.Join("internal", "walker")] {
			t.Error("internal/walker should be dirty (contains processed file)")
		}
		if !dirty["internal"] {
			t.Error("internal should be dirty (ancestor of dirty folder)")
		}
		if !dirty["."] {
			t.Error(". should be dirty (ancestor of dirty folder)")
		}
		if dirty[filepath.Join("internal", "config")] {
			t.Error("internal/config should NOT be dirty (no processed files, summary exists)")
		}
	})

	t.Run("missing summary marks folder dirty", func(t *testing.T) {
		// internal/walker has no summary file on disk.
		processedFiles := []string{}
		dirty := buildDirtySet(folders, processedFiles, absRoot, false)

		if !dirty[filepath.Join("internal", "walker")] {
			t.Error("internal/walker should be dirty (summary missing)")
		}
		if !dirty["internal"] {
			t.Error("internal should be dirty (ancestor of missing summary)")
		}
		if !dirty["."] {
			t.Error(". should be dirty (ancestor of missing summary)")
		}
		if dirty[filepath.Join("internal", "config")] {
			t.Error("internal/config should NOT be dirty (summary exists)")
		}
	})

	t.Run("force marks all dirty", func(t *testing.T) {
		dirty := buildDirtySet(folders, []string{}, absRoot, true)

		for _, f := range folders {
			if !dirty[f] {
				t.Errorf("%q should be dirty with --force", f)
			}
		}
	})

	t.Run("nothing processed and all summaries exist", func(t *testing.T) {
		// Create the missing walker summary.
		walkerSummary := outputPath(absRoot, filepath.Join("internal", "walker"))
		os.MkdirAll(filepath.Dir(walkerSummary), 0755)
		os.WriteFile(walkerSummary, []byte("# walker summary"), 0644)

		dirty := buildDirtySet(folders, []string{}, absRoot, false)

		for _, f := range folders {
			if dirty[f] {
				t.Errorf("%q should NOT be dirty (nothing changed, summary exists)", f)
			}
		}

		// Cleanup for other subtests.
		os.Remove(walkerSummary)
	})
}

func TestDepth(t *testing.T) {
	tests := []struct {
		folder string
		want   int
	}{
		{".", 0},
		{"internal", 1},
		{filepath.Join("internal", "config"), 2},
		{filepath.Join("a", "b", "c"), 3},
	}

	for _, tt := range tests {
		t.Run(tt.folder, func(t *testing.T) {
			got := depth(tt.folder)
			if got != tt.want {
				t.Errorf("depth(%q) = %d, want %d", tt.folder, got, tt.want)
			}
		})
	}
}

func TestManifestKey(t *testing.T) {
	if got := manifestKey("internal/config"); got != "dir:internal/config" {
		t.Errorf("manifestKey() = %q, want %q", got, "dir:internal/config")
	}

	folder, ok := fromManifestKey("dir:internal/config")
	if !ok || folder != "internal/config" {
		t.Errorf("fromManifestKey(dir:internal/config) = (%q, %v), want (internal/config, true)", folder, ok)
	}

	_, ok = fromManifestKey("internal/config.go")
	if ok {
		t.Error("fromManifestKey should return false for non-dir keys")
	}
}
