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

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

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
