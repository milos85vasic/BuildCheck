# CLAUDE.md - BuildCheck Module


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Source-hash change detection → NeedsRebuild decision
cd BuildCheck && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestChangeDetector_NeedsRebuild' ./...
```
Expect: PASS; SHA-256 hashing of source tree yields a stable rebuild decision; manifest round-trip produces identical output.


## Overview

`digital.vasic.buildcheck` is a content-based change detection library for container image builds. It uses SHA256 hashing to track source file changes and determine if container images need rebuilding.

## Module

- **Name:** `digital.vasic.buildcheck`
- **Go Version:** 1.24+
- **Purpose:** Detect source code changes to optimize container rebuilds

## Architecture

```
┌─────────────────────────────────────────────┐
│              ChangeDetector                  │
│  ┌──────────────┐  ┌──────────────────┐    │
│  │ HashComputer │  │  ManifestStore   │    │
│  │  (SHA256)    │  │ (File/Memory)    │    │
│  └──────┬───────┘  └────────┬─────────┘    │
│         │                   │               │
│         └─────────┬─────────┘               │
│                   │                         │
│           ┌───────▼───────┐                 │
│           │ ChangeReport  │                 │
│           │  - HasChanges │                 │
│           │  - Changes[]  │                 │
│           │  - SourceHash │                 │
│           └───────────────┘                 │
└─────────────────────────────────────────────┘
```

## Package Structure

```
BuildCheck/
├── go.mod
├── README.md
├── CLAUDE.md
├── AGENTS.md
└── pkg/
    └── buildcheck/
        ├── types.go      # Core types and interfaces
        ├── hash.go       # SHA256 hash computation
        ├── store.go      # Manifest storage (File/Memory)
        ├── detector.go   # Change detection logic
        ├── options.go    # Functional options
        └── *_test.go     # Unit tests
```

## Key Interfaces

### HashComputer

```go
type HashComputer interface {
    ComputeFileHash(path string) (FileHash, error)
    ComputeDirectoryHash(root string, ignorePaths []string) (map[string]FileHash, string, error)
}
```

### ManifestStore

```go
type ManifestStore interface {
    Load(imageName string) (*Manifest, error)
    Save(manifest *Manifest) error
    Delete(imageName string) error
    List() ([]string, error)
    Exists(imageName string) bool
}
```

### ChangeDetector

```go
type ChangeDetector interface {
    DetectChanges(config ImageConfig) (*ChangeReport, error)
    ComputeSourceHash(config ImageConfig) (string, map[string]FileHash, error)
    NeedsRebuild(config ImageConfig) (bool, *ChangeReport, error)
}
```

## Usage Patterns

### Basic Rebuild Check

```go
store, _ := buildcheck.NewFileStore("./manifests")
detector := buildcheck.NewDetector(store)

config := buildcheck.ImageConfig{
    Name:        "my-api",
    ContextPath: "./cmd/api",
    IgnorePaths: []string{".git", "node_modules"},
}

needsRebuild, report, _ := detector.NeedsRebuild(config)
if needsRebuild {
    // Build container image
    buildImage(config)
    
    // Record successful build
    detector.RecordBuild(config, "v1.2.3")
}
```

### Multiple Images

```go
images := []buildcheck.ImageConfig{
    {Name: "api", ContextPath: "./cmd/api"},
    {Name: "worker", ContextPath: "./cmd/worker"},
    {Name: "scheduler", ContextPath: "./cmd/scheduler"},
}

for _, img := range images {
    if needsRebuild, _, _ := detector.NeedsRebuild(img); needsRebuild {
        rebuildQueue = append(rebuildQueue, img)
    }
}
```

## Change Detection Algorithm

1. **Load Previous Manifest**: Retrieve stored manifest for image
2. **Compute Current Hashes**: Walk directory, compute SHA256 for each file
3. **Compare Hashes**: 
   - Files in old but not new → `ChangeTypeDeleted`
   - Files in new but not old → `ChangeTypeAdded`
   - Files in both with different hash → `ChangeTypeModified`
4. **Generate Report**: Aggregate hash + individual changes

## Hash Computation

- **Algorithm**: SHA256
- **File Hash**: SHA256 of file contents
- **Directory Hash**: SHA256 of concatenated `path:hash;` strings
- **Deterministic**: Same content always produces same hash

## Storage Format

Manifest JSON structure:

```json
{
  "version": "1.0.0",
  "image_name": "my-api",
  "source_hash": "a1b2c3...",
  "file_hashes": {
    "main.go": {
      "path": "main.go",
      "hash": "x9y8z7...",
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

## Testing

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Integration with HelixAgent

BuildCheck integrates with the Containers module to:
1. Detect changes before container builds
2. Skip unnecessary rebuilds
3. Record build metadata for audit trails
4. Support incremental build strategies

## Error Handling

- `os.IsNotExist`: First-time build (no manifest)
- `os.IsPermission`: Skip files/directories without read access
- All errors wrapped with context using `fmt.Errorf`

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | root only |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.
