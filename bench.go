// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	// Default sampling configuration
	minSamples        = 2
	defaultSamples    = 100
	defaultDuration   = 10 * time.Millisecond
	defaultTableFmt   = "%-20s %-12s %-12s %-12s %-18s %-18s\n"
	defaultFilename   = "bench.gob"
	defaultConfidence = 99.9
	defaultThreshold  = 5.0
	defaultBootstrap  = 100000
)

func defaultConfig() config {
	return config{
		filename:   defaultFilename,
		samples:    defaultSamples,
		duration:   defaultDuration,
		tableFmt:   defaultTableFmt,
		confidence: defaultConfidence,
		threshold:  defaultThreshold,
		bootstrap:  defaultBootstrap,
		codec:      gobCodec{},
	}
}

// Result represents a single benchmark result
type Result struct {
	Name        string      `json:"name"`
	Samples     []float64   `json:"samples"`
	Allocs      []float64   `json:"allocs"`
	Timestamp   int64       `json:"timestamp"`
	Environment Environment `json:"environment"`
	Calibration []float64   `json:"calibration,omitempty"` // CPU calibration timings, in collection order
	CPUUsage    float64     `json:"cpuUsage"`              // Average system utilization in percent; -1 if unavailable
}

// B manages benchmarks and handles persistence
type B struct {
	config
	t testing.TB
}

// Run executes benchmarks with the given configuration
func Run(fn func(*B), opts ...Option) {
	cfg := defaultConfig()

	// Apply flags first so user options can override
	initFlags(&cfg)

	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.normalize()

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

func (r *B) benchmarkPair(ourFn, refFn func(op int) int) (ourTiming, ourAllocs, refTiming, calibrations []float64) {
	ourTiming = make([]float64, 0, r.samples)
	ourAllocs = make([]float64, 0, r.samples)
	calibrations = make([]float64, 0, r.samples)
	if refFn != nil {
		refTiming = make([]float64, 0, r.samples)
	}

	for i := 0; i < r.samples; i++ {
		var ourNS, ourAlloc, refNS, calibrationNS float64
		if i%2 == 0 {
			calibrationNS, _ = r.sample(calibration)
			ourNS, ourAlloc = r.sample(ourFn)
			if refFn != nil {
				refNS, _ = r.sample(refFn)
			}
		} else {
			if refFn != nil {
				refNS, _ = r.sample(refFn)
			}
			ourNS, ourAlloc = r.sample(ourFn)
			calibrationNS, _ = r.sample(calibration)
		}

		ourTiming = append(ourTiming, ourNS)
		ourAllocs = append(ourAllocs, ourAlloc)
		calibrations = append(calibrations, calibrationNS)
		if refFn != nil {
			refTiming = append(refTiming, refNS)
		}
	}
	return
}

func (r *B) sample(fn func(op int) int) (nsPerOp, allocsPerOp float64) {
	// Force GC to get clean allocation measurements.
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()
	ops := 0
	for {
		ops = addOps(ops, fn(ops))
		if time.Since(start) >= r.duration {
			break
		}
	}
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m2)

	return float64(elapsed.Nanoseconds()) / float64(ops),
		float64(m2.Mallocs-m1.Mallocs) / float64(ops)
}

func addOps(total, n int) int {
	if n <= 0 {
		panic("bench: RunN function must return a positive operation count")
	}

	maxInt := int(^uint(0) >> 1)
	if n > maxInt-total {
		panic("bench: RunN operation count overflow")
	}

	return total + n
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
		return Report{Inconclusive: "filtered"}
	}

	// Load previous results for delta comparison
	prevResults := r.loadResults()

	environment := captureEnvironment(r.duration)
	cpuBefore := cpuSnapshot()
	ourSamples, ourAllocs, refSamples, calibrations := r.benchmarkPair(ourFn, refFn)
	usage := cpuUsage(cpuBefore, cpuSnapshot())

	// Create result
	result := Result{
		Name:        name,
		Samples:     ourSamples,
		Allocs:      ourAllocs,
		Timestamp:   time.Now().Unix(),
		Environment: environment,
		Calibration: calibrations,
		CPUUsage:    usage,
	}
	return r.record(result, prevResults, refSamples)
}

// record compares collected measurements, reports assertions, and updates a
// usable baseline. Input results are read-only; persistence uses the codec.
func (r *B) record(result Result, previous map[string]Result, refSamples []float64) (report Report) {
	name := result.Name
	nsPerOp := median(result.Samples)
	opsPerSec := 1e9 / nsPerOp
	avgAllocsPerOp := median(result.Allocs)

	// Calculate delta vs previous run
	prevResult, exists := previous[name]
	vsPrev := "new"
	allocsChange := allocUnknown
	report.Inconclusive = "no baseline"
	if exists {
		report = r.compare(prevResult, result)
		vsPrev = r.formatComparison(report)
		allocsChange = compareAllocs(prevResult.Allocs, result.Allocs)
		if r.t != nil && report.Significant && report.Delta > 0 {
			r.t.Errorf("%s has a performance regression of %s", name, vsPrev)
		}
	}

	// Calculate vs reference if provided
	vsRef := ""
	if refSamples != nil {
		report := bcaWithSeed(refSamples, result.Samples, r.confidence/100.0, r.bootstrap, r.threshold, r.seed)
		vsRef = r.formatComparison(report)
	} else if refResult, ok := r.loadReferenceResults()[name]; ok {
		vsRef = r.formatComparison(r.compare(refResult, result))
	}

	// Format and display result
	fmt.Printf(r.tableFmt, name,
		formatTime(nsPerOp),
		formatOps(opsPerSec),
		formatAllocsWithChange(avgAllocsPerOp, allocsChange),
		vsPrev,
		vsRef)

	// Keep a comparable baseline when a run is inconclusive. Legacy files get
	// one fresh baseline with calibration metadata on the next writable run.
	if !exists || report.Inconclusive == "" || report.Inconclusive == "no calibration" ||
		(report.Inconclusive == "baseline incomplete" && r.usable(result)) {
		r.saveResult(result)
	}
	return
}

// compare retains the measured effect but withholds a verdict unless the saved
// and current runs have compatible environments and equivalent CPU calibration.
func (r *B) compare(previous, current Result) Report {
	report := bcaWithSeed(previous.Samples, current.Samples, r.confidence/100, r.bootstrap, r.threshold, r.seed)
	reason := ""
	switch {
	case len(previous.Calibration) == 0 || len(current.Calibration) == 0:
		reason = "no calibration"
	case !previous.Environment.valid() || !current.Environment.valid():
		reason = "unknown setup"
	case previous.Environment != current.Environment:
		reason = "setup changed"
	case len(previous.Calibration) != len(previous.Samples) || len(current.Calibration) != len(current.Samples):
		reason = "invalid calibration"
	case !r.usable(previous):
		reason = "baseline incomplete"
	default:
		calibration := bcaWithSeed(previous.Calibration, current.Calibration, r.confidence/100, r.bootstrap, r.threshold, r.seed)
		margin := 1 + r.threshold/100
		if calibration.Inconclusive != "" || calibration.RatioCI[0] < 1/margin || calibration.RatioCI[1] > margin {
			reason = "CPU unstable"
		}
	}
	if reason != "" {
		report.Significant = false
		report.Inconclusive = reason
	}
	return report
}

// usable permits replacing an inadequate baseline once a run has enough valid,
// unclustered observations. It does not establish cross-run equivalence.
func (r *B) usable(result Result) bool {
	if !result.Environment.valid() || len(result.Calibration) != len(result.Samples) ||
		!validSamples(result.Samples) || !validSamples(result.Calibration) ||
		clustered(result.Samples, median(result.Samples)) || clustered(result.Calibration, median(result.Calibration)) {
		return false
	}
	lower, _ := medianInterval(result.Samples, (1-r.confidence/100)/4)
	return lower > 0
}

// Assert runs benchmarks in dry-run mode and fails the test on a supported
// regression. Inconclusive comparisons are reported without failing the test;
// callers can inspect Report.Inconclusive to enforce a stricter policy.
// It is skipped when testing is run with -short.
func Assert(t testing.TB, fn func(*B), opts ...Option) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping benchmark assertion in short mode")
	}

	cfg := defaultConfig()
	cfg.dryRun = true

	initFlags(&cfg)
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.normalize()

	runner := &B{config: cfg, t: t}
	runner.printHeader()
	fn(runner)
}
