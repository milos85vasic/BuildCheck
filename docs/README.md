# BuildCheck Documentation

## Architecture Overview

BuildCheck (`digital.vasic.buildcheck`) is a content-based change detection library for container image builds. It determines whether container images need rebuilding by computing SHA256 hashes of source files and comparing them against previously recorded manifests.

### Component Diagram

```
                        +---------------------+
                        |    ChangeDetector    |
                        |   (detector.go)      |
                        +----------+----------+
                               /         \
                              /           \
               +-------------+--+   +------+-----------+
               |  HashComputer  |   |  ManifestStore   |
               |   (hash.go)    |   |   (store.go)     |
               +-------+--------+   +------+-----------+
                       |                    |
            SHA256 of files        File or Memory backend
                       |                    |
                       v                    v
               +-------+--------+   +------+-----------+
               |    FileHash    |   |    Manifest      |
               |  (types.go)    |   |   (types.go)     |
               +----------------+   +------------------+
                       \                   /
                        \                 /
                     +---+---------------+---+
                     |    ChangeReport       |
                     |  - HasChanges         |
                     |  - Changes[]          |
                     |  - SourceHash         |
                     +----------------------+
```

### Package Structure

```
BuildCheck/
├── go.mod                  # Module: digital.vasic.buildcheck (Go 1.24+)
├── go.sum
├── README.md
├── CLAUDE.md
├── AGENTS.md
├── docs/
│   └── README.md           # This file
└── pkg/
    └── buildcheck/
        ├── types.go        # Core types: ImageConfig, Manifest, FileHash, ChangeReport
        ├── hash.go         # SHA256 hash computation for files and directories
        ├── store.go        # ManifestStore implementations (FileStore, MemoryStore)
        ├── detector.go     # ChangeDetector implementation and rebuild logic
        ├── options.go      # Functional options for configuring components
        └── *_test.go       # Unit and benchmark tests
```

### Core Interfaces

BuildCheck is built around three interfaces that separate concerns:

**HashComputer** -- Computes SHA256 hashes for individual files and entire directories:

```go
type HashComputer interface {
    ComputeFileHash(path string) (FileHash, error)
    ComputeDirectoryHash(root string, ignorePaths []string) (map[string]FileHash, string, error)
}
```

**ManifestStore** -- Persists and retrieves build manifests:

```go
type ManifestStore interface {
    Load(imageName string) (*Manifest, error)
    Save(manifest *Manifest) error
    Delete(imageName string) error
    List() ([]string, error)
    Exists(imageName string) bool
}
```

**ChangeDetector** -- The primary API for detecting changes and determining rebuild necessity:

```go
type ChangeDetector interface {
    DetectChanges(config ImageConfig) (*ChangeReport, error)
    ComputeSourceHash(config ImageConfig) (string, map[string]FileHash, error)
    NeedsRebuild(config ImageConfig) (bool, *ChangeReport, error)
}
```

### Change Detection Algorithm

1. **Load previous manifest** for the image name from the `ManifestStore`.
2. **Walk the build context directory**, computing SHA256 for each file (respecting `IgnorePaths`).
3. **Compare file hashes** against the stored manifest:
   - Files present in the old manifest but missing now are marked `ChangeTypeDeleted`.
   - Files present now but absent from the old manifest are marked `ChangeTypeAdded`.
   - Files present in both but with differing hashes are marked `ChangeTypeModified`.
4. **Compute an aggregate directory hash** (SHA256 of sorted `path:hash;` pairs) for deterministic comparison.
5. **Generate a `ChangeReport`** containing the aggregate hash, the list of individual file changes, and a boolean `HasChanges` flag.

If no previous manifest exists (first build), the detector treats all files as added and reports that a rebuild is needed.

### Hash Computation Details

- **File hash**: SHA256 digest of the raw file contents.
- **Directory hash**: SHA256 digest of concatenated `path:hash;` strings, sorted alphabetically by path. This ensures deterministic results regardless of filesystem traversal order.
- **Ignored paths**: Directories like `.git`, `node_modules`, and `vendor` can be excluded via `IgnorePaths` in `ImageConfig`.

---

## How to Use BuildCheck

### Installation

```bash
go get digital.vasic.buildcheck
```

### Basic Usage

```go
package main

import (
    "fmt"
    "digital.vasic.buildcheck/pkg/buildcheck"
)

func main() {
    // 1. Create a manifest store (file-based for persistence)
    store, err := buildcheck.NewFileStore("./manifests")
    if err != nil {
        panic(err)
    }

    // 2. Create a change detector
    detector := buildcheck.NewDetector(store)

    // 3. Define the image configuration
    config := buildcheck.ImageConfig{
        Name:        "my-api",
        ContextPath: "./cmd/api",
        IgnorePaths: []string{".git", "node_modules", "vendor"},
    }

    // 4. Check if a rebuild is needed
    needsRebuild, report, err := detector.NeedsRebuild(config)
    if err != nil {
        panic(err)
    }

    if needsRebuild {
        fmt.Printf("Rebuild needed: %d file(s) changed\n", len(report.Changes))
        for _, change := range report.Changes {
            fmt.Printf("  %s: %s\n", change.Type, change.Path)
        }

        // ... perform the container build ...

        // 5. Record the build so the next check has a baseline
        detector.RecordBuild(config, "v1.0.0")
    } else {
        fmt.Println("No changes detected, skipping rebuild")
    }
}
```

### Checking Multiple Images

```go
images := []buildcheck.ImageConfig{
    {Name: "api",       ContextPath: "./cmd/api"},
    {Name: "worker",    ContextPath: "./cmd/worker"},
    {Name: "scheduler", ContextPath: "./cmd/scheduler"},
}

var rebuildQueue []buildcheck.ImageConfig
for _, img := range images {
    needsRebuild, _, err := detector.NeedsRebuild(img)
    if err != nil {
        log.Printf("Error checking %s: %v", img.Name, err)
        continue
    }
    if needsRebuild {
        rebuildQueue = append(rebuildQueue, img)
    }
}
```

### Computing Source Hash Without Comparison

If you only need the current source hash (e.g., for tagging or logging):

```go
hash, fileHashes, err := detector.ComputeSourceHash(config)
if err != nil {
    panic(err)
}
fmt.Printf("Aggregate hash: %s (%d files)\n", hash, len(fileHashes))
```

---

## Configuration Options

### ImageConfig Fields

| Field          | Type       | Required | Description                                           |
|----------------|------------|----------|-------------------------------------------------------|
| `Name`         | `string`   | Yes      | Unique image identifier used for manifest lookup      |
| `ContextPath`  | `string`   | Yes      | Root directory to scan for source file changes        |
| `Dockerfile`   | `string`   | No       | Path to Dockerfile (stored as metadata in manifest)   |
| `IgnorePaths`  | `[]string` | No       | Directories/files to exclude from hashing             |
| `IncludePaths` | `[]string` | No       | If set, only these paths are scanned                  |
| `BuildCommand` | `string`   | No       | Build command (metadata only, not executed)            |
| `BuildArgs`    | `[]string` | No       | Build arguments (metadata only)                       |

### Storage Backends

**FileStore** -- Persists manifests as JSON files on disk. Uses `sync.RWMutex` for thread safety and atomic writes (temp file + rename) for crash safety.

```go
store, err := buildcheck.NewFileStore("/path/to/manifests")
```

Each manifest is stored as `<manifests-dir>/<image-name>.json`.

**MemoryStore** -- In-memory storage for testing or ephemeral use cases. Data does not survive process restarts.

```go
store := buildcheck.NewMemoryStore()
```

### Manifest JSON Format

```json
{
  "version": "1.0.0",
  "image_name": "my-api",
  "source_hash": "a1b2c3d4e5f6...",
  "file_hashes": {
    "main.go": {
      "path": "main.go",
      "hash": "x9y8z7w6v5u4...",
      "size": 1234,
      "mod_time": "2024-01-01T00:00:00Z"
    }
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-02T00:00:00Z",
  "last_build_at": "2024-01-02T00:00:00Z",
  "last_build_tag": "v1.2.3"
}
```

---

## Integration with HelixAgent

BuildCheck integrates with the HelixAgent release build system and the Containers module (`digital.vasic.containers`) to optimize container image rebuilds.

### Role in the Build Pipeline

HelixAgent builds 7 applications (`helixagent`, `api`, `grpc-server`, `cognee-mock`, `sanity-check`, `mcp-bridge`, `generate-constitution`) across 5 platforms. Without change detection, every build would rebuild all images. BuildCheck provides the following optimizations:

1. **Skip unchanged images** -- Before building a container image, the build system calls `NeedsRebuild()` with the image's source context. If no files have changed since the last recorded build, the image build is skipped entirely.

2. **Deterministic hash tagging** -- The `SourceHash` from `ChangeReport` can be used as an image tag or part of `build-info.json`, tying each image to a specific source state.

3. **Audit trail** -- `RecordBuild()` stores the build tag and timestamp alongside the file hashes, creating a historical record of what source code produced each image version.

4. **Incremental build strategies** -- The detailed `Changes` list (added/modified/deleted files) enables smart rebuild decisions. For example, changes only in documentation files might not warrant a container rebuild.

### Connection to Containers Module

The Containers module (`digital.vasic.containers`) handles runtime detection, compose orchestration, and health checking. BuildCheck complements it by handling the pre-build decision: _should_ the image be rebuilt before the Containers module deploys it.

The typical flow is:

```
Source Code Change
       |
       v
  BuildCheck: NeedsRebuild()?
       |
   yes / no
      |      \
      v       v
  Build     Skip
  Image     Build
      |       |
      v       v
  Containers Module: Deploy & Health Check
```

### Connection to Release Build System

The HelixAgent release build system (`scripts/build/`) already performs SHA256-based change detection for release binaries (see `make release` and `compute_source_hash` in `version-manager.sh`). BuildCheck provides the same capability as a reusable Go library, enabling programmatic use from Go code rather than shell scripts.

---

## Testing

```bash
# Run all tests
go test ./...

# With race detection
go test -race ./...

# Benchmarks
go test -bench=. ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Concurrency Safety

- `FileStore` protects its internal cache with `sync.RWMutex`.
- `MemoryStore` protects its map with `sync.RWMutex`.
- `Detector` is stateless and safe for concurrent use (all state lives in the store).
- File writes use the atomic temp-file-and-rename pattern to prevent corruption.

### Error Handling

- `os.IsNotExist` errors when loading a manifest indicate a first-time build (no previous state). This is not an error condition; `NeedsRebuild()` returns `true`.
- `os.IsPermission` errors cause individual files to be skipped during directory hashing.
- All errors are wrapped with context using `fmt.Errorf("...: %w", err)`.

---

## Dependencies

| Dependency                    | Purpose                     |
|-------------------------------|-----------------------------|
| `github.com/google/uuid`     | Unique IDs for temp files   |
| `github.com/stretchr/testify`| Test assertions             |

BuildCheck has minimal dependencies by design to keep it lightweight and broadly reusable.
