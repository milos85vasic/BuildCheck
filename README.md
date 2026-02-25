# BuildCheck

**Module:** `digital.vasic.buildcheck`

Content-based change detection for container image builds. Detects source code changes using SHA256 hashes to determine if container images need rebuilding.

## Features

- **Content-Hashed Change Detection**: SHA256-based file and directory hashing
- **Manifest Storage**: Persistent storage with atomic writes (file or memory backend)
- **Change Reports**: Detailed reports of added, modified, and deleted files
- **Rebuild Detection**: Simple API to check if rebuilds are needed
- **Ignore Paths**: Exclude directories (e.g., `node_modules`, `.git`)

## Installation

```bash
go get digital.vasic.buildcheck
```

## Quick Start

```go
package main

import (
    "fmt"
    "digital.vasic.buildcheck/pkg/buildcheck"
)

func main() {
    // Create a manifest store
    store, _ := buildcheck.NewFileStore("./manifests")
    
    // Create a change detector
    detector := buildcheck.NewDetector(store)
    
    // Configure the image
    config := buildcheck.ImageConfig{
        Name:        "my-app",
        ContextPath: "./src",
        IgnorePaths: []string{"node_modules", ".git"},
    }
    
    // Check if rebuild is needed
    needsRebuild, report, _ := detector.NeedsRebuild(config)
    
    if needsRebuild {
        fmt.Printf("Changes detected: %d files changed\n", len(report.Changes))
        
        // After building, record the build
        detector.RecordBuild(config, "v1.0.0")
    }
}
```

## API

### Types

```go
type ImageConfig struct {
    Name         string   // Image name
    ContextPath  string   // Build context path
    Dockerfile   string   // Dockerfile path (optional)
    IgnorePaths  []string // Paths to ignore
    IncludePaths []string // Paths to include (optional)
    BuildCommand string   // Build command (metadata)
    BuildArgs    []string // Build arguments (metadata)
}

type ChangeReport struct {
    ImageName  string    // Image name
    HasChanges bool      // Whether changes detected
    Changes    []Change  // List of changes
    SourceHash string    // Aggregate source hash
}
```

### Interfaces

```go
type ChangeDetector interface {
    DetectChanges(config ImageConfig) (*ChangeReport, error)
    ComputeSourceHash(config ImageConfig) (string, map[string]FileHash, error)
    NeedsRebuild(config ImageConfig) (bool, *ChangeReport, error)
}

type ManifestStore interface {
    Load(imageName string) (*Manifest, error)
    Save(manifest *Manifest) error
    Delete(imageName string) error
    List() ([]string, error)
    Exists(imageName string) bool
}
```

## Storage Backends

### FileStore

Persists manifests to JSON files on disk.

```go
store, _ := buildcheck.NewFileStore("/path/to/manifests")
```

### MemoryStore

In-memory storage for testing or ephemeral use.

```go
store := buildcheck.NewMemoryStore()
```

## Change Types

- `ChangeTypeNone` - No change
- `ChangeTypeAdded` - File added
- `ChangeTypeModified` - File modified
- `ChangeTypeDeleted` - File deleted

## Testing

```bash
go test ./...
go test -bench=. ./...
```

## License

Proprietary - vasic-digital
