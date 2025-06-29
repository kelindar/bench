package bench

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/codahale/tinystat"
)

const (
	// Default sampling configuration
	DefaultSamples  = 100
	DefaultDuration = 10 * time.Millisecond
	DefaultTableFmt = "%-20s %-12s %-12s %-12s %-18s %-18s\n"
	DefaultFilename = "bench.json"
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
}

// Run executes benchmarks with the given configuration
func Run(fn func(*B), opts ...Option) {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	prefix := fs.String("bench", "", "Run only benchmarks with this prefix")
	dry := fs.Bool("n", false, "dry run - do not update bench.json")

	// Parse only our known flags from os.Args
	args := []string{}
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		if strings.HasPrefix(a, "-bench") || a == "-bench" || strings.HasPrefix(a, "-n") || a == "-n" {
			args = append(args, a)
			if !strings.Contains(a, "=") && i+1 < len(os.Args) {
				i++
				args = append(args, os.Args[i])
			}
		}
	}
	_ = fs.Parse(args)

	cfg := config{
		filename: DefaultFilename,
		samples:  DefaultSamples,
		duration: DefaultDuration,
		tableFmt: DefaultTableFmt,
		filter:   *prefix,
		dryRun:   *dry,
		codec:    jsonCodec{},
	}

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
func (r *B) benchmark(fn func(b *B, op int)) (samples []float64, allocs []float64) {
	samples = make([]float64, 0, r.samples)
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
			fn(r, ops)
			ops++
		}
		elapsed := time.Since(start)

		runtime.ReadMemStats(&m2)

		opsPerSec := float64(ops) / elapsed.Seconds()
		allocsPerOp := float64(m2.Mallocs-m1.Mallocs) / float64(ops)

		samples = append(samples, opsPerSec)
		allocs = append(allocs, allocsPerOp)
	}
	return samples, allocs
}

// formatAllocs formats number of allocations per operation
func (r *B) formatAllocs(allocsPerOp float64) string {
	switch {
	case allocsPerOp >= 1000:
		return fmt.Sprintf("%.1fK", allocsPerOp/1000)
	case allocsPerOp >= 1:
		return fmt.Sprintf("%.0f", allocsPerOp)
	default:
		return "0"
	}
}

// formatComparison formats statistical comparison between two sample sets
func (r *B) formatComparison(ourSamples, otherSamples []float64) string {
	if len(otherSamples) == 0 {
		return "new"
	}

	our := tinystat.Summarize(ourSamples)
	other := tinystat.Summarize(otherSamples)
	if other.Mean == 0 {
		if our.Mean > 0 {
			return "✅ +inf%"
		}
		return "~ similar"
	}

	speedup := our.Mean / other.Mean
	changePercent := (speedup - 1) * 100
	diff := tinystat.Compare(our, other, 99.9)

	// For non-significant changes close to zero, show "similar"
	if !diff.Significant() && changePercent >= -2 && changePercent <= 2 {
		return "~ similar"
	}

	var sign string
	if changePercent > 0 {
		sign = "+"
	} else {
		sign = ""
	}

	if !diff.Significant() {
		return fmt.Sprintf("~ %s%.0f%% (p=%.3f)", sign, changePercent, diff.PValue)
	}

	if speedup > 1 {
		return fmt.Sprintf("✅ %s%.0f%% (p=%.3f)", sign, changePercent, diff.PValue)
	}

	return fmt.Sprintf("❌ %s%.0f%% (p=%.3f)", sign, changePercent, diff.PValue)
}

// formatTime formats nanoseconds per operation
func (r *B) formatTime(nsPerOp float64) string {
	switch {
	case nsPerOp >= 1000000:
		return fmt.Sprintf("%.1f ms", nsPerOp/1000000)
	case nsPerOp >= 1000:
		return fmt.Sprintf("%.1f µs", nsPerOp/1000)
	default:
		return fmt.Sprintf("%.1f ns", nsPerOp)
	}
}

// formatOps formats operations per second
func (r *B) formatOps(opsPerSec float64) string {
	if opsPerSec >= 1000000 {
		return fmt.Sprintf("%.1fM", opsPerSec/1000000)
	}
	if opsPerSec >= 1000 {
		return fmt.Sprintf("%.1fK", opsPerSec/1000)
	}
	return fmt.Sprintf("%.0f", opsPerSec)
}

// Run executes a benchmark with optional reference comparison
func (r *B) Run(name string, ourFn func(b *B, op int), refFn ...func(b *B, op int)) {
	if !r.shouldRun(name) {
		return
	}

	// Load previous results for delta comparison
	prevResults := r.loadResults()

	// Benchmark our implementation
	ourSamples, ourAllocs := r.benchmark(ourFn)
	ourMean := tinystat.Summarize(ourSamples).Mean
	nsPerOp := 1e9 / ourMean

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
	delta := "new"
	if exists {
		delta = r.formatComparison(ourSamples, prevResult.Samples)
	}

	// Calculate vs reference if provided
	vsRef := ""
	if len(refFn) > 0 && refFn[0] != nil {
		refSamples, _ := r.benchmark(func(b *B, op int) { refFn[0](b, op) })
		vsRef = r.formatComparison(ourSamples, refSamples)
	}

	// Format and display result
	if r.showRef {
		fmt.Printf(r.tableFmt,
			name,
			r.formatTime(nsPerOp),
			r.formatOps(ourMean),
			r.formatAllocs(avgAllocsPerOp),
			delta,
			vsRef)
	} else {
		fmt.Printf("%-20s %-12s %-12s %-12s %-18s\n",
			name,
			r.formatTime(nsPerOp),
			r.formatOps(ourMean),
			r.formatAllocs(avgAllocsPerOp),
			delta)
	}

	// Save result incrementally
	r.saveResult(result)
}
