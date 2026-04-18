// SPDX-License-Identifier: Apache-2.0
//
// Package-level stress tests for digital.vasic.buildcheck.
//
// These tests serve as the canonical P3 stress-test template for other
// extracted modules: they use ONLY in-package helpers (no external
// services), respect the Constitution's resource-limit contract
// (GOMAXPROCS=2, no global goroutine blow-ups), and validate that the
// core MemoryStore + Detector APIs are safe under sustained concurrent
// use.
//
// Run with:
//   GOMAXPROCS=2 nice -n 19 ionice -c 3 go test -race -run '^TestStress' \
//       ./pkg/buildcheck/ -p 1 -count=1 -timeout 120s
package buildcheck

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Per Constitution rule CONST-022, stress tests must fit inside
	// 30–40% of host resources. The concurrency levels below are sized
	// for GOMAXPROCS=2 and single-process -p 1 execution.
	stressGoroutines   = 8
	stressIterations   = 200
	stressMaxWallClock = 30 * time.Second
)

// TestStress_MemoryStore_ConcurrentSaveLoad asserts MemoryStore remains
// consistent under a sustained mix of Save / Load / Exists / Delete
// calls across goroutines. It is the safety net for future refactors
// of the in-memory store.
func TestStress_MemoryStore_ConcurrentSaveLoad(t *testing.T) {
	store := NewMemoryStore()

	var wg sync.WaitGroup
	var errors atomic.Int64
	deadline := time.Now().Add(stressMaxWallClock)

	for g := 0; g < stressGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < stressIterations; j++ {
				if time.Now().After(deadline) {
					return
				}
				name := fmt.Sprintf("img-%d-%d", id, j%16)
				m := &Manifest{
					Version:   "1.0.0",
					ImageName: name,
					SourceHash: fmt.Sprintf("h-%d-%d", id, j),
					FileHashes: map[string]FileHash{
						"main.go": {Path: "main.go", Hash: "xxx"},
					},
				}
				if err := store.Save(m); err != nil {
					errors.Add(1)
					continue
				}
				if _, err := store.Load(name); err != nil {
					errors.Add(1)
				}
				_ = store.Exists(name)
				if j%7 == 0 {
					_ = store.Delete(name)
				}
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, int64(0), errors.Load(), "stress ops should not produce errors")

	list, err := store.List()
	require.NoError(t, err)
	// List should be bounded by the working-set size (16 unique names per
	// goroutine but some are deleted). The exact count is schedule-dependent;
	// the assertion here is just "not unbounded growth".
	assert.LessOrEqual(t, len(list), stressGoroutines*16,
		"MemoryStore grew beyond expected working set")
}

// TestStress_Detector_ConcurrentDetectChanges asserts Detector is safe
// under parallel DetectChanges + RecordBuild calls against the same
// in-memory store.
func TestStress_Detector_ConcurrentDetectChanges(t *testing.T) {
	store := NewMemoryStore()
	// Seed with one manifest so DetectChanges has something to compare.
	require.NoError(t, store.Save(&Manifest{
		Version:    "1.0.0",
		ImageName:  "seeded",
		SourceHash: "abc",
		FileHashes: map[string]FileHash{"f.go": {Path: "f.go", Hash: "abc"}},
	}))

	detector := NewDetector(store)

	tmp := t.TempDir()
	// Write a fixed-content file so ComputeSourceHash is deterministic.
	fp := tmp + "/main.go"
	require.NoError(t, writeFile(fp, "package main\nfunc main(){}\n"))

	startGoroutines := runtime.NumGoroutine()
	var wg sync.WaitGroup
	var errCount atomic.Int64
	deadline := time.Now().Add(stressMaxWallClock)

	for g := 0; g < stressGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < stressIterations; j++ {
				if time.Now().After(deadline) {
					return
				}
				cfg := ImageConfig{
					Name:        fmt.Sprintf("stress-%d", id%4),
					ContextPath: tmp,
				}
				if _, err := detector.DetectChanges(cfg); err != nil {
					errCount.Add(1)
				}
				if j%5 == 0 {
					_ = detector.RecordBuild(cfg, fmt.Sprintf("v%d.%d", id, j))
				}
			}
		}(g)
	}
	wg.Wait()

	assert.Equal(t, int64(0), errCount.Load(), "DetectChanges should not error on well-formed inputs")

	// Allow runtime to reap worker goroutines.
	time.Sleep(50 * time.Millisecond)
	runtime.Gosched()
	endGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, endGoroutines-startGoroutines, 2,
		"goroutine leak: worker count grew by %d", endGoroutines-startGoroutines)
}

// BenchmarkStress_MemoryStore_Save / Load establish a performance
// baseline so future regressions can be gated at ±25% per CONST-022
// resource budget.
func BenchmarkStress_MemoryStore_Save(b *testing.B) {
	store := NewMemoryStore()
	m := &Manifest{
		Version:    "1.0.0",
		ImageName:  "bench",
		SourceHash: "sha",
		FileHashes: map[string]FileHash{"a.go": {Path: "a.go", Hash: "1"}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ImageName = fmt.Sprintf("bench-%d", i%64)
		_ = store.Save(m)
	}
}

func BenchmarkStress_MemoryStore_Load(b *testing.B) {
	store := NewMemoryStore()
	for i := 0; i < 64; i++ {
		_ = store.Save(&Manifest{
			Version:    "1.0.0",
			ImageName:  fmt.Sprintf("bench-%d", i),
			SourceHash: "sha",
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Load(fmt.Sprintf("bench-%d", i%64))
	}
}

// writeFile is a tiny helper so the stress test does not pull in
// testing/testutil. Inline to keep the template self-contained.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
