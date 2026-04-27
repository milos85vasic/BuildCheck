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

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->



<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.
