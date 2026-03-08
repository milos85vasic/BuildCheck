# BuildCheck - API Reference

**Module:** `digital.vasic.buildcheck`
**Package:** `buildcheck`

## Constructor Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewDetector` | `NewDetector(store ManifestStore, opts ...Option) *Detector` | Creates a change detector with a manifest store. |
| `NewFileStore` | `NewFileStore(dir string) (*FileStore, error)` | Creates a file-based manifest store at the given directory. |
| `NewMemoryStore` | `NewMemoryStore() *MemoryStore` | Creates an in-memory manifest store. |
| `NewHashComputer` | `NewHashComputer() *DefaultHashComputer` | Creates the default SHA256 hash computer. |

## Interfaces

### ChangeDetector

Primary API for detecting changes and determining rebuild necessity.

```go
type ChangeDetector interface {
    DetectChanges(config ImageConfig) (*ChangeReport, error)
    ComputeSourceHash(config ImageConfig) (string, map[string]FileHash, error)
    NeedsRebuild(config ImageConfig) (bool, *ChangeReport, error)
}
```

| Method | Description |
|--------|-------------|
| `DetectChanges` | Compares current files against stored manifest, returns report |
| `ComputeSourceHash` | Computes aggregate hash without comparing to manifest |
| `NeedsRebuild` | Returns true if image should be rebuilt (DetectChanges + logic) |

### HashComputer

Computes SHA256 hashes for files and directories.

```go
type HashComputer interface {
    ComputeFileHash(path string) (FileHash, error)
    ComputeDirectoryHash(root string, ignorePaths []string) (map[string]FileHash, string, error)
}
```

| Method | Description |
|--------|-------------|
| `ComputeFileHash` | SHA256 of a single file's contents |
| `ComputeDirectoryHash` | SHA256 of all files in a directory; returns file map and aggregate hash |

### ManifestStore

Persists and retrieves build manifests.

```go
type ManifestStore interface {
    Load(imageName string) (*Manifest, error)
    Save(manifest *Manifest) error
    Delete(imageName string) error
    List() ([]string, error)
    Exists(imageName string) bool
}
```

| Method | Description |
|--------|-------------|
| `Load` | Retrieves a stored manifest by image name |
| `Save` | Persists a manifest (creates or updates) |
| `Delete` | Removes a manifest |
| `List` | Returns all stored image names |
| `Exists` | Checks if a manifest exists for an image |

## Core Types

### ImageConfig

```go
type ImageConfig struct {
    Name         string   `json:"name"`
    ContextPath  string   `json:"context_path"`
    Dockerfile   string   `json:"dockerfile,omitempty"`
    IgnorePaths  []string `json:"ignore_paths,omitempty"`
    IncludePaths []string `json:"include_paths,omitempty"`
    BuildCommand string   `json:"build_command,omitempty"`
    BuildArgs    []string `json:"build_args,omitempty"`
}
```

### Manifest

```go
type Manifest struct {
    Version     string                `json:"version"`
    ImageName   string                `json:"image_name"`
    SourceHash  string                `json:"source_hash"`
    FileHashes  map[string]FileHash   `json:"file_hashes"`
    CreatedAt   time.Time             `json:"created_at"`
    UpdatedAt   time.Time             `json:"updated_at"`
    LastBuildAt time.Time             `json:"last_build_at"`
    LastBuildTag string               `json:"last_build_tag"`
}
```

### FileHash

```go
type FileHash struct {
    Path    string    `json:"path"`
    Hash    string    `json:"hash"`
    Size    int64     `json:"size"`
    ModTime time.Time `json:"mod_time"`
}
```

### ChangeReport

```go
type ChangeReport struct {
    HasChanges bool          `json:"has_changes"`
    SourceHash string        `json:"source_hash"`
    Changes    []FileChange  `json:"changes"`
}
```

### FileChange

```go
type FileChange struct {
    Path    string     `json:"path"`
    Type    ChangeType `json:"type"`
    OldHash string     `json:"old_hash,omitempty"`
    NewHash string     `json:"new_hash,omitempty"`
}
```

## Enums

### ChangeType

| Constant | Value | Description |
|----------|-------|-------------|
| `ChangeTypeAdded` | `"added"` | File is new (not in previous manifest) |
| `ChangeTypeModified` | `"modified"` | File exists in both but hash differs |
| `ChangeTypeDeleted` | `"deleted"` | File was in previous manifest but is now missing |

## Functional Options

```go
type Option func(*Detector)
func WithLogger(logger *logrus.Logger) Option
func WithHashComputer(hc HashComputer) Option
```

## Detector Methods

Beyond the `ChangeDetector` interface, the concrete `Detector` also exposes:

| Method | Signature | Description |
|--------|-----------|-------------|
| `RecordBuild` | `RecordBuild(config ImageConfig, tag string) error` | Records a successful build for future comparisons |

## Error Handling

| Condition | Behavior |
|-----------|----------|
| No previous manifest | `NeedsRebuild` returns `true` (first build) |
| Permission denied on file | File skipped during directory hashing |
| Corrupt manifest JSON | `Load` returns an error |
| All errors | Wrapped with context via `fmt.Errorf("...: %w", err)` |
