package buildcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type FileStore struct {
	baseDir string
	mu      sync.RWMutex
	cache   map[string]*Manifest
}

func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	return &FileStore{
		baseDir: baseDir,
		cache:   make(map[string]*Manifest),
	}, nil
}

func (s *FileStore) manifestPath(imageName string) string {
	safeName := filepath.Base(imageName)
	return filepath.Join(s.baseDir, safeName+".json")
}

func (s *FileStore) Load(imageName string) (*Manifest, error) {
	s.mu.RLock()
	if m, ok := s.cache[imageName]; ok {
		s.mu.RUnlock()
		return m, nil
	}
	s.mu.RUnlock()

	path := s.manifestPath(imageName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	m, err := ManifestFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	s.mu.Lock()
	s.cache[imageName] = m
	s.mu.Unlock()

	return m, nil
}

func (s *FileStore) Save(manifest *Manifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest cannot be nil")
	}

	if manifest.ImageName == "" {
		return fmt.Errorf("manifest must have an image name")
	}

	now := time.Now()
	if manifest.CreatedAt.IsZero() {
		manifest.CreatedAt = now
	}
	manifest.UpdatedAt = now

	if manifest.Version == "" {
		manifest.Version = "1.0.0"
	}

	data, err := manifest.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	path := s.manifestPath(manifest.ImageName)
	tmpPath := path + ".tmp." + uuid.New().String()[:8]

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename manifest: %w", err)
	}

	s.mu.Lock()
	s.cache[manifest.ImageName] = manifest
	s.mu.Unlock()

	return nil
}

func (s *FileStore) Delete(imageName string) error {
	path := s.manifestPath(imageName)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}

	s.mu.Lock()
	delete(s.cache, imageName)
	s.mu.Unlock()

	return nil
}

func (s *FileStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()[:len(entry.Name())-5]
			names = append(names, name)
		}
	}

	return names, nil
}

func (s *FileStore) Exists(imageName string) bool {
	s.mu.RLock()
	if _, ok := s.cache[imageName]; ok {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()

	_, err := os.Stat(s.manifestPath(imageName))
	return err == nil
}

type MemoryStore struct {
	mu        sync.RWMutex
	manifests map[string]*Manifest
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		manifests: make(map[string]*Manifest),
	}
}

func (s *MemoryStore) Load(imageName string) (*Manifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.manifests[imageName], nil
}

func (s *MemoryStore) Save(manifest *Manifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest cannot be nil")
	}

	now := time.Now()
	if manifest.CreatedAt.IsZero() {
		manifest.CreatedAt = now
	}
	manifest.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()
	s.manifests[manifest.ImageName] = manifest
	return nil
}

func (s *MemoryStore) Delete(imageName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.manifests, imageName)
	return nil
}

func (s *MemoryStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.manifests))
	for name := range s.manifests {
		names = append(names, name)
	}
	return names, nil
}

func (s *MemoryStore) Exists(imageName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.manifests[imageName]
	return ok
}

func (s *MemoryStore) ToJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s.manifests, "", "  ")
}
