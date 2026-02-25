package buildcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func formatHash(b []byte) string {
	return hex.EncodeToString(b)
}

type SHA256Hasher struct{}

func NewSHA256Hasher() *SHA256Hasher {
	return &SHA256Hasher{}
}

func (h *SHA256Hasher) ComputeFileHash(path string) (FileHash, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileHash{}, err
	}

	if info.IsDir() {
		return FileHash{
			Path:    path,
			IsDir:   true,
			ModTime: info.ModTime(),
		}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return FileHash{}, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return FileHash{}, err
	}

	return FileHash{
		Path:    path,
		Hash:    formatHash(hasher.Sum(nil)),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   false,
	}, nil
}

func (h *SHA256Hasher) ComputeDirectoryHash(root string, ignorePaths []string) (map[string]FileHash, string, error) {
	fileHashes := make(map[string]FileHash)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}

	ignoreMap := make(map[string]bool)
	for _, p := range ignorePaths {
		ignoreMap[filepath.Clean(p)] = true
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		for ignore := range ignoreMap {
			if strings.HasPrefix(relPath, ignore) || relPath == ignore {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			fileHashes[relPath] = FileHash{
				Path:    relPath,
				IsDir:   true,
				ModTime: info.ModTime(),
			}
			return nil
		}

		fh, err := h.ComputeFileHash(path)
		if err != nil {
			return err
		}
		fh.Path = relPath
		fileHashes[relPath] = fh
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	aggregateHasher := sha256.New()
	for relPath, fh := range fileHashes {
		if !fh.IsDir {
			aggregateHasher.Write([]byte(relPath + ":" + fh.Hash + ";"))
		}
	}

	return fileHashes, formatHash(aggregateHasher.Sum(nil)), nil
}
