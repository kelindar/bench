// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"gonum.org/v1/gonum/stat"
)

const (
	// Default sampling configuration
	defaultSamples    = 100
	defaultDuration   = 10 * time.Millisecond
	defaultTableFmt   = "%-20s %-12s %-12s %-12s %-18s %-18s\n"
	defaultFilename   = "bench.gob"
	defaultConfidence = 99.9
	boostramSamples   = 10000
)

// Result represents a single benchmark result
type Result struct {
	Name      string    `json:"name"`
	Samples   []float64 `json:"samples"`
	Allocs    []float64 `json:"-"`
	Timestamp int64     `json:"timestamp"`
}

// B manages benchmarks and handles persistence
type B struct {
	config
	t testing.TB
}

// Run executes benchmarks with the given configuration
func Run(fn func(*B), opts ...Option) {
	cfg := config{
		filename:   defaultFilename,
		samples:    defaultSamples,
		duration:   defaultDuration,
		tableFmt:   defaultTableFmt,
		confidence: defaultConfidence,
		codec:      gobCodec{},
	}

	// Apply flags first so user options can override
	initFlags(&cfg)

	for _, opt := range opts {
		opt(&cfg)
	}

	runner := &B{config: cfg}
	runner.printHeader()
	fn(runner)
}

// printHeader prints the table header
func (r *B) printHeader() {
	if r.showRef {
		fmt.Printf(r.tableFmt, "name", "time/op", "ops/s", "allocs/op", "vs prev", "vs ref")
		fmt.Printf(r.tableFmt, "--------------------", "------------", "------------", "------------", "------------------", "------------------")
	} else {
		fmt.Printf("%-20s %-12s %-12s %-12s %-18s\n", "name", "time/op", "ops/s", "allocs/op", "vs prev")
		fmt.Printf("%-20s %-12s %-12s %-12s %-18s\n", "--------------------", "------------", "------------", "------------", "------------------")
	}
}

// shouldRun checks if a benchmark matches the filter
func (r *B) shouldRun(name string) bool {
	if r.filter == "" {
		return true
	}
	return strings.HasPrefix(name, r.filter)
}

// benchmark runs a function repeatedly and returns performance samples
func (r *B) benchmark(fn func(op int) int) (timing []float64, allocs []float64) {
	timing = make([]float64, 0, r.samples)
	allocs = make([]float64, 0, r.samples)

	for i := 0; i < r.samples; i++ {
		// Force GC to get clean allocation measurements
		runtime.GC()
		runtime.GC()

		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		ops := 0
		for time.Since(start) < r.duration {
			ops += fn(ops)
		}
		elapsed := time.Since(start)

		runtime.ReadMemStats(&m2)

		nsPerOp := float64(elapsed.Nanoseconds()) / float64(ops)
		allocsPerOp := float64(m2.Mallocs-m1.Mallocs) / float64(ops)

		timing = append(timing, nsPerOp)
		allocs = append(allocs, allocsPerOp)
	}
	return timing, allocs
}

// Run executes a benchmark with optional reference comparison
func (r *B) Run(name string, ourFn func(i int), refFn ...func(i int)) Report {
	var refWrapped func(int) int
	if len(refFn) > 0 && refFn[0] != nil {
		rf := refFn[0]
		refWrapped = func(i int) int { rf(i); return 1 }
	}

	return r.run(name, func(i int) int { ourFn(i); return 1 }, refWrapped)
}

// RunN executes a benchmark where each iteration may return the number of
// operations performed. This allows amortizing expensive setup or batching.
func (r *B) RunN(name string, ourFn func(i int) int, refFn ...func(i int) int) Report {
	var refWrapped func(int) int
	if len(refFn) > 0 {
		refWrapped = refFn[0]
	}

	return r.run(name, ourFn, refWrapped)
}

func (r *B) run(name string, ourFn func(int) int, refFn func(int) int) (report Report) {
	if !r.shouldRun(name) {
		return
	}

	// Load previous results for delta comparison
	prevResults := r.loadResults()

	// Benchmark our implementation
	ourSamples, ourAllocs := r.benchmark(ourFn)
	nsPerOp := stat.Mean(ourSamples, nil)
	opsPerSec := 1e9 / nsPerOp

	// Calculate average allocations per operation
	var totalAllocs float64
	for _, v := range ourAllocs {
		totalAllocs += v
	}
	avgAllocsPerOp := totalAllocs / float64(len(ourSamples))

	// Create result
	result := Result{
		Name:      name,
		Samples:   ourSamples,
		Timestamp: time.Now().Unix(),
	}

	// Calculate delta vs previous run
	prevResult, exists := prevResults[name]
	vsPrev := "new"
	if exists {
		report = bca(prevResult.Samples, ourSamples, r.confidence/100.0, boostramSamples)
		vsPrev = r.formatComparison(report)
		if r.t != nil && report.Significant && report.Delta > 0 {
			r.t.Errorf("%s has a performance regression of %s", name, vsPrev)
		}
	}

	// Calculate vs reference if provided
	vsRef := ""
	if refFn != nil {
		refSamples, _ := r.benchmark(refFn)
		report := bca(refSamples, ourSamples, r.confidence/100.0, boostramSamples)
		vsRef = r.formatComparison(report)
	}

	// Format and display result
	fmt.Printf(r.tableFmt, name,
		formatTime(nsPerOp),
		formatOps(opsPerSec),
		formatAllocs(avgAllocsPerOp),
		vsPrev,
		vsRef)

	// Save result incrementally
	r.saveResult(result)
	return
}

// Assert runs benchmarks in dry-run mode and fails the test if performance regresses.
// It is skipped when testing is run with -short.
func Assert(t testing.TB, fn func(*B), opts ...Option) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping benchmark assertion in short mode")
	}

	cfg := config{
		filename:   defaultFilename,
		samples:    defaultSamples,
		duration:   defaultDuration,
		tableFmt:   defaultTableFmt,
		confidence: defaultConfidence,
		codec:      gobCodec{},
		dryRun:     true,
	}

	initFlags(&cfg)
	for _, opt := range opts {
		opt(&cfg)
	}

	runner := &B{config: cfg, t: t}
	runner.printHeader()
	fn(runner)
}
