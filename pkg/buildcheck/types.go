package buildcheck

import (
	"crypto/sha256"
	"encoding/json"
	"time"
)

type ChangeType int

const (
	ChangeTypeNone ChangeType = iota
	ChangeTypeAdded
	ChangeTypeModified
	ChangeTypeDeleted
)

func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeNone:
		return "none"
	case ChangeTypeAdded:
		return "added"
	case ChangeTypeModified:
		return "modified"
	case ChangeTypeDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

type FileHash struct {
	Path    string    `json:"path"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

type Manifest struct {
	Version      string              `json:"version"`
	ImageName    string              `json:"image_name"`
	SourceHash   string              `json:"source_hash"`
	FileHashes   map[string]FileHash `json:"file_hashes"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	BuildCommand string              `json:"build_command,omitempty"`
	BuildArgs    []string            `json:"build_args,omitempty"`
	LastBuildAt  time.Time           `json:"last_build_at,omitempty"`
	LastBuildTag string              `json:"last_build_tag,omitempty"`
}

type Change struct {
	Type    ChangeType `json:"type"`
	Path    string     `json:"path"`
	OldHash string     `json:"old_hash,omitempty"`
	NewHash string     `json:"new_hash,omitempty"`
}

type ChangeReport struct {
	ImageName   string    `json:"image_name"`
	HasChanges  bool      `json:"has_changes"`
	Changes     []Change  `json:"changes"`
	OldManifest *Manifest `json:"old_manifest,omitempty"`
	NewManifest *Manifest `json:"new_manifest,omitempty"`
	ComputedAt  time.Time `json:"computed_at"`
	SourceHash  string    `json:"source_hash"`
}

type ImageConfig struct {
	Name         string
	ContextPath  string
	Dockerfile   string
	IgnorePaths  []string
	IncludePaths []string
	BuildCommand string
	BuildArgs    []string
}

type HashComputer interface {
	ComputeFileHash(path string) (FileHash, error)
	ComputeDirectoryHash(root string, ignorePaths []string) (map[string]FileHash, string, error)
}

type ManifestStore interface {
	Load(imageName string) (*Manifest, error)
	Save(manifest *Manifest) error
	Delete(imageName string) error
	List() ([]string, error)
	Exists(imageName string) bool
}

type ChangeDetector interface {
	DetectChanges(config ImageConfig) (*ChangeReport, error)
	ComputeSourceHash(config ImageConfig) (string, map[string]FileHash, error)
	NeedsRebuild(config ImageConfig) (bool, *ChangeReport, error)
}

func (m *Manifest) ComputeAggregateHash() string {
	if len(m.FileHashes) == 0 {
		return ""
	}

	hasher := sha256.New()
	for path, fh := range m.FileHashes {
		hasher.Write([]byte(path + ":" + fh.Hash + ";"))
	}
	return formatHash(hasher.Sum(nil))
}

func (m *Manifest) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func ManifestFromJSON(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
