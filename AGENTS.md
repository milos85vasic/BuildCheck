# AGENTS.md - BuildCheck Module

Guidance for AI coding agents working on the BuildCheck module.

## Project Overview

BuildCheck (`digital.vasic.buildcheck`) is a Go library for content-based change detection in container image builds. It uses SHA256 hashing to track source file changes and optimize rebuild decisions.

## Build Commands

```bash
go build ./...
go build -o bin/buildcheck ./...
```

## Testing

```bash
go test ./...                    # All tests
go test -v -race ./...           # Verbose with race detection
go test -coverprofile=coverage.out ./...  # Coverage
go test -bench=. ./...           # Benchmarks
```

## Code Style

- Standard Go conventions (`gofmt`, `goimports`)
- Imports grouped: stdlib, third-party, local
- Line length ≤ 100 characters
- No comments unless explicitly requested
- Error wrapping: `fmt.Errorf("context: %w", err)`

## Architecture

### Core Components

| Component | File | Purpose |
|-----------|------|---------|
| Types | `types.go` | Core types, interfaces, JSON marshaling |
| Hash | `hash.go` | SHA256 hash computation for files/directories |
| Store | `store.go` | Manifest storage (File/Memory backends) |
| Detector | `detector.go` | Change detection logic |
| Options | `options.go` | Functional options pattern |

### Key Types

- `Manifest` - Stored state for an image
- `FileHash` - Hash metadata for a single file
- `ChangeReport` - Result of change detection
- `ImageConfig` - Image configuration for detection

### Interfaces

- `HashComputer` - File/directory hashing
- `ManifestStore` - Persistence abstraction
- `ChangeDetector` - Main detection API

## Adding New Features

1. Define types in `types.go`
2. Implement interfaces in appropriate files
3. Add comprehensive tests in `*_test.go`
4. Update documentation (README.md, CLAUDE.md)

## Storage Backends

To add a new storage backend:

1. Implement `ManifestStore` interface
2. Add constructor function `NewXxxStore()`
3. Add tests for the new backend
4. Register in factory if needed

## Hash Algorithms

Currently using SHA256. To add a new algorithm:

1. Create new hasher implementing `HashComputer`
2. Add option to configure hasher
3. Ensure backward compatibility with existing manifests

## Error Handling

- Wrap errors with context
- Use `os.IsNotExist` for missing files/manifests
- Use `os.IsPermission` for access errors
- Never panic in library code

## Concurrency

- `FileStore` uses `sync.RWMutex` for cache protection
- `MemoryStore` uses `sync.RWMutex` for map protection
- `Detector` is stateless (safe for concurrent use)
- Atomic file writes using temp file + rename pattern

## Testing Patterns

- Use `testify` assertions
- Table-driven tests for multiple cases
- Use `t.TempDir()` for temporary directories
- Clean up resources with `defer`

## Git Conventions

- Branch naming: `feat/`, `fix/`, `chore/`, `docs/`
- Conventional Commits: `feat(detector): add xyz`
- Run `go fmt` and `go vet` before committing

## Dependencies

- `github.com/google/uuid` - Unique IDs for temp files
- `github.com/stretchr/testify` - Testing assertions

Minimal dependencies - avoid adding new ones unless necessary.
