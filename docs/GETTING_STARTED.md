# BuildCheck - Getting Started

**Module:** `digital.vasic.buildcheck`

## Installation

```go
import "digital.vasic.buildcheck/pkg/buildcheck"
```

## Quick Start: Check If an Image Needs Rebuilding

### 1. Create a Manifest Store

Choose between file-based (persistent) or memory-based (ephemeral):

```go
package main

import (
    "fmt"

    "digital.vasic.buildcheck/pkg/buildcheck"
)

func main() {
    // File-based store for persistent manifests
    store, err := buildcheck.NewFileStore("./manifests")
    if err != nil {
        panic(err)
    }

    // Or: in-memory store for testing
    // store := buildcheck.NewMemoryStore()
```

### 2. Create a Change Detector

```go
    detector := buildcheck.NewDetector(store)
```

### 3. Define Image Configuration

```go
    config := buildcheck.ImageConfig{
        Name:        "my-api",
        ContextPath: "./cmd/api",
        IgnorePaths: []string{".git", "node_modules", "vendor"},
    }
```

### 4. Check for Changes

```go
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

        // Record the build for future comparisons
        detector.RecordBuild(config, "v1.0.0")
    } else {
        fmt.Println("No changes detected, skipping rebuild")
    }
}
```

## Checking Multiple Images

```go
images := []buildcheck.ImageConfig{
    {Name: "api", ContextPath: "./cmd/api"},
    {Name: "worker", ContextPath: "./cmd/worker"},
    {Name: "scheduler", ContextPath: "./cmd/scheduler"},
}

var rebuildQueue []buildcheck.ImageConfig
for _, img := range images {
    needsRebuild, _, err := detector.NeedsRebuild(img)
    if err != nil {
        fmt.Printf("Error checking %s: %v\n", img.Name, err)
        continue
    }
    if needsRebuild {
        rebuildQueue = append(rebuildQueue, img)
    }
}

fmt.Printf("%d image(s) need rebuilding\n", len(rebuildQueue))
```

## Computing a Source Hash

Get the current source hash without comparing to a manifest:

```go
hash, fileHashes, err := detector.ComputeSourceHash(config)
if err != nil {
    panic(err)
}
fmt.Printf("Aggregate hash: %s (%d files)\n", hash, len(fileHashes))
```

This is useful for tagging images or logging source state.

## ImageConfig Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Name` | `string` | Yes | Unique image identifier for manifest lookup |
| `ContextPath` | `string` | Yes | Root directory to scan for changes |
| `Dockerfile` | `string` | No | Path to Dockerfile (metadata only) |
| `IgnorePaths` | `[]string` | No | Paths to exclude from hashing |
| `IncludePaths` | `[]string` | No | If set, only these paths are scanned |
| `BuildCommand` | `string` | No | Build command (metadata only) |
| `BuildArgs` | `[]string` | No | Build arguments (metadata only) |

## Storage Options

### FileStore

Manifests persist as JSON files on disk at
`<manifests-dir>/<image-name>.json`. Atomic writes prevent corruption.

```go
store, _ := buildcheck.NewFileStore("/path/to/manifests")
```

### MemoryStore

For testing or CI pipelines where persistence is not needed:

```go
store := buildcheck.NewMemoryStore()
```

## Running Tests

```bash
go test ./...                            # All tests
go test -race ./...                      # With race detection
go test -bench=. ./...                   # Benchmarks
go test -coverprofile=coverage.out ./... # Coverage
```

## Next Steps

- See [ARCHITECTURE.md](ARCHITECTURE.md) for the detection algorithm
- See [API_REFERENCE.md](API_REFERENCE.md) for all types and interfaces
