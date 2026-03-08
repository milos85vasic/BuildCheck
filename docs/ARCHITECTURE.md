# BuildCheck - Architecture

**Module:** `digital.vasic.buildcheck`

## Overview

BuildCheck is a content-based change detection library for container image
builds. It computes SHA256 hashes of source files and compares them against
previously recorded manifests to determine whether container images need
rebuilding.

## Component Diagram

```
                    +---------------------+
                    |    ChangeDetector    |
                    |   (detector.go)     |
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

## Change Detection Algorithm

1. **Load previous manifest** for the image name from the ManifestStore
2. **Walk the build context directory**, computing SHA256 for each file
   (respecting `IgnorePaths`)
3. **Compare file hashes** against the stored manifest:
   - Present in old but missing now: `ChangeTypeDeleted`
   - Present now but absent from old: `ChangeTypeAdded`
   - Present in both with differing hashes: `ChangeTypeModified`
4. **Compute aggregate directory hash** (SHA256 of sorted `path:hash;`
   pairs) for deterministic comparison
5. **Generate `ChangeReport`** with aggregate hash, individual changes,
   and `HasChanges` flag

If no previous manifest exists (first build), all files are treated as
added and a rebuild is needed.

## Hash Computation

- **File hash:** SHA256 digest of raw file contents
- **Directory hash:** SHA256 of concatenated `path:hash;` strings sorted
  alphabetically by path (deterministic regardless of filesystem order)
- **Ignored paths:** `.git`, `node_modules`, `vendor` configurable via
  `IgnorePaths` in `ImageConfig`

## Storage Backends

### FileStore

Persists manifests as JSON files on disk. Uses `sync.RWMutex` for
thread safety and atomic writes (temp file + rename) for crash safety.

Each manifest is stored at `<manifests-dir>/<image-name>.json`.

### MemoryStore

In-memory storage for testing or ephemeral use. Protected by
`sync.RWMutex`. Data does not survive process restarts.

## Functional Options

The `options.go` file provides the functional options pattern for
configuring components:

```go
detector := buildcheck.NewDetector(store, buildcheck.WithLogger(logger))
```

## Integration with HelixAgent

BuildCheck integrates with two systems:

1. **Containers Module** (`digital.vasic.containers`): Determines whether
   images need rebuilding before the Containers module deploys them
2. **Release Build System** (`scripts/build/`): Provides the same
   SHA256-based change detection as a Go library instead of shell scripts

### Build Pipeline Flow

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

## Concurrency Safety

- `FileStore` and `MemoryStore` protect internal state with `sync.RWMutex`
- `Detector` is stateless and safe for concurrent use
- File writes use atomic temp-file-and-rename pattern

## Package Structure

```
BuildCheck/
├── pkg/
│   └── buildcheck/
│       ├── types.go      # ImageConfig, Manifest, FileHash, ChangeReport
│       ├── hash.go       # SHA256 hash computation
│       ├── store.go      # FileStore, MemoryStore implementations
│       ├── detector.go   # ChangeDetector logic
│       ├── options.go    # Functional options
│       └── *_test.go     # Unit and benchmark tests
├── go.mod
├── CLAUDE.md
├── AGENTS.md
└── README.md
```
