package buildcheck

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		name     string
		ct       ChangeType
		expected string
	}{
		{"none", ChangeTypeNone, "none"},
		{"added", ChangeTypeAdded, "added"},
		{"modified", ChangeTypeModified, "modified"},
		{"deleted", ChangeTypeDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ct.String())
		})
	}
}

func TestManifest_ComputeAggregateHash(t *testing.T) {
	t.Run("empty manifest", func(t *testing.T) {
		m := &Manifest{FileHashes: make(map[string]FileHash)}
		assert.Equal(t, "", m.ComputeAggregateHash())
	})

	t.Run("with file hashes", func(t *testing.T) {
		m := &Manifest{
			FileHashes: map[string]FileHash{
				"main.go": {Path: "main.go", Hash: "abc123"},
				"util.go": {Path: "util.go", Hash: "def456"},
			},
		}
		hash := m.ComputeAggregateHash()
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64)
	})

	t.Run("deterministic hash", func(t *testing.T) {
		m := &Manifest{
			FileHashes: map[string]FileHash{
				"main.go": {Path: "main.go", Hash: "abc123"},
			},
		}
		hash1 := m.ComputeAggregateHash()
		hash2 := m.ComputeAggregateHash()
		assert.Equal(t, hash1, hash2)
	})
}

func TestManifest_ToJSON(t *testing.T) {
	m := &Manifest{
		Version:    "1.0.0",
		ImageName:  "test-image",
		SourceHash: "abc123",
		FileHashes: map[string]FileHash{
			"main.go": {Path: "main.go", Hash: "xyz789"},
		},
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := m.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-image")
	assert.Contains(t, string(data), "abc123")
}

func TestManifestFromJSON(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		jsonData := `{
			"version": "1.0.0",
			"image_name": "test-image",
			"source_hash": "abc123",
			"file_hashes": {
				"main.go": {"path": "main.go", "hash": "xyz789"}
			}
		}`

		m, err := ManifestFromJSON([]byte(jsonData))
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", m.Version)
		assert.Equal(t, "test-image", m.ImageName)
		assert.Equal(t, "abc123", m.SourceHash)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := ManifestFromJSON([]byte("invalid"))
		assert.Error(t, err)
	})
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	t.Run("save and load", func(t *testing.T) {
		m := &Manifest{
			ImageName:  "test-image",
			SourceHash: "abc123",
		}

		err := store.Save(m)
		require.NoError(t, err)

		loaded, err := store.Load("test-image")
		require.NoError(t, err)
		assert.Equal(t, "abc123", loaded.SourceHash)
		assert.False(t, loaded.CreatedAt.IsZero())
	})

	t.Run("load non-existent", func(t *testing.T) {
		loaded, err := store.Load("non-existent")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("delete", func(t *testing.T) {
		m := &Manifest{ImageName: "to-delete"}
		store.Save(m)

		err := store.Delete("to-delete")
		require.NoError(t, err)

		assert.False(t, store.Exists("to-delete"))
	})

	t.Run("list", func(t *testing.T) {
		store := NewMemoryStore()
		store.Save(&Manifest{ImageName: "image1"})
		store.Save(&Manifest{ImageName: "image2"})

		list, err := store.List()
		require.NoError(t, err)
		assert.Len(t, list, 2)
	})
}

func TestFileStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "buildcheck-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewFileStore(tmpDir)
	require.NoError(t, err)

	t.Run("save and load", func(t *testing.T) {
		m := &Manifest{
			ImageName:  "test-image",
			SourceHash: "abc123",
		}

		err := store.Save(m)
		require.NoError(t, err)

		loaded, err := store.Load("test-image")
		require.NoError(t, err)
		assert.Equal(t, "abc123", loaded.SourceHash)
	})

	t.Run("exists", func(t *testing.T) {
		assert.True(t, store.Exists("test-image"))
		assert.False(t, store.Exists("non-existent"))
	})

	t.Run("list", func(t *testing.T) {
		list, err := store.List()
		require.NoError(t, err)
		assert.Contains(t, list, "test-image")
	})

	t.Run("delete", func(t *testing.T) {
		err := store.Delete("test-image")
		require.NoError(t, err)
		assert.False(t, store.Exists("test-image"))
	})
}

func TestSHA256Hasher_ComputeFileHash(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "hash-test-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("test content for hashing")
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	tmpFile.Close()

	hasher := NewSHA256Hasher()
	fh, err := hasher.ComputeFileHash(tmpFile.Name())
	require.NoError(t, err)

	assert.NotEmpty(t, fh.Hash)
	assert.Len(t, fh.Hash, 64)
	assert.Equal(t, int64(len(content)), fh.Size)
	assert.False(t, fh.IsDir)
}

func TestSHA256Hasher_ComputeDirectoryHash(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hash-dir-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package util"), 0644)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	require.NoError(t, err)

	hasher := NewSHA256Hasher()
	fileHashes, aggHash, err := hasher.ComputeDirectoryHash(tmpDir, nil)
	require.NoError(t, err)

	assert.NotEmpty(t, aggHash)
	assert.Len(t, aggHash, 64)
	assert.Contains(t, fileHashes, "main.go")
	assert.Contains(t, fileHashes, "util.go")
	assert.Contains(t, fileHashes, "subdir")
}

func TestSHA256Hasher_ComputeDirectoryHash_IgnorePaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hash-ignore-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "node_modules", "pkg.js"), []byte("module.exports"), 0644)
	require.NoError(t, err)

	hasher := NewSHA256Hasher()
	fileHashes, _, err := hasher.ComputeDirectoryHash(tmpDir, []string{"node_modules"})
	require.NoError(t, err)

	assert.Contains(t, fileHashes, "main.go")
	assert.NotContains(t, fileHashes, "node_modules")
	assert.NotContains(t, fileHashes, "node_modules/pkg.js")
}

func TestDetector_DetectChanges(t *testing.T) {
	store := NewMemoryStore()
	detector := NewDetector(store)

	tmpDir, err := os.MkdirTemp("", "detect-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	require.NoError(t, err)

	config := ImageConfig{
		Name:        "test-image",
		ContextPath: tmpDir,
	}

	t.Run("first detection - no manifest", func(t *testing.T) {
		report, err := detector.DetectChanges(config)
		require.NoError(t, err)

		assert.True(t, report.HasChanges)
		assert.NotNil(t, report.NewManifest)
		assert.Nil(t, report.OldManifest)
		assert.NotEmpty(t, report.SourceHash)
	})

	t.Run("record build", func(t *testing.T) {
		err := detector.RecordBuild(config, "v1.0.0")
		require.NoError(t, err)

		manifest, err := store.Load("test-image")
		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", manifest.LastBuildTag)
	})

	t.Run("no changes after build", func(t *testing.T) {
		needsRebuild, _, err := detector.NeedsRebuild(config)
		require.NoError(t, err)
		assert.False(t, needsRebuild)
	})

	t.Run("detect modification", func(t *testing.T) {
		time.Sleep(10 * time.Millisecond)
		err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main // modified"), 0644)
		require.NoError(t, err)

		needsRebuild, report, err := detector.NeedsRebuild(config)
		require.NoError(t, err)
		assert.True(t, needsRebuild)
		assert.True(t, report.HasChanges)

		found := false
		for _, c := range report.Changes {
			if c.Path == "main.go" && c.Type == ChangeTypeModified {
				found = true
				break
			}
		}
		assert.True(t, found, "expected modified change for main.go")
	})

	t.Run("detect deletion", func(t *testing.T) {
		detector.RecordBuild(config, "v1.0.1")

		err := os.Remove(filepath.Join(tmpDir, "main.go"))
		require.NoError(t, err)

		_, report, err := detector.NeedsRebuild(config)
		require.NoError(t, err)

		found := false
		for _, c := range report.Changes {
			if c.Path == "main.go" && c.Type == ChangeTypeDeleted {
				found = true
				break
			}
		}
		assert.True(t, found, "expected deleted change for main.go")
	})

	t.Run("detect addition", func(t *testing.T) {
		store := NewMemoryStore()
		detector := NewDetector(store)

		err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
		require.NoError(t, err)

		detector.RecordBuild(config, "v1.0.0")

		err = os.WriteFile(filepath.Join(tmpDir, "new.go"), []byte("package new"), 0644)
		require.NoError(t, err)

		_, report, err := detector.NeedsRebuild(config)
		require.NoError(t, err)

		found := false
		for _, c := range report.Changes {
			if c.Path == "new.go" && c.Type == ChangeTypeAdded {
				found = true
				break
			}
		}
		assert.True(t, found, "expected added change for new.go")
	})
}
