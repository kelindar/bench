package bench

import (
	"encoding/json"
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

// Option configures the benchmark runner
type Option func(*config)

type config struct {
	filename string
	filter   string
	samples  int
	duration time.Duration
	tableFmt string
	showRef  bool
}

// WithFile sets the filename for benchmark results
func WithFile(filename string) Option {
	return func(c *config) {
		c.filename = filename
	}
}

// WithFilter sets a prefix filter for benchmark names
func WithFilter(prefix string) Option {
	return func(c *config) {
		c.filter = prefix
	}
}

// WithSamples sets the number of samples to collect per benchmark
func WithSamples(n int) Option {
	return func(c *config) {
		c.samples = n
	}
}

// WithDuration sets the duration for each sample
func WithDuration(d time.Duration) Option {
	return func(c *config) {
		c.duration = d
	}
}

// WithReference enables reference comparison column
func WithReference() Option {
	return func(c *config) {
		c.showRef = true
	}
}

// B manages benchmarks and handles persistence
type B struct {
	config
}

// Run executes benchmarks with the given configuration
func Run(fn func(*B), opts ...Option) {
	cfg := config{
		filename: DefaultFilename,
		samples:  DefaultSamples,
		duration: DefaultDuration,
		tableFmt: DefaultTableFmt,
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
func (r *B) benchmark(fn func()) (samples []float64, allocs []float64) {
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
			fn()
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

// loadResults loads previous results from JSON file
func (r *B) loadResults() map[string]Result {
	data, err := os.ReadFile(r.filename)
	if err != nil {
		return make(map[string]Result)
	}

	var results map[string]Result
	if err := json.Unmarshal(data, &results); err != nil {
		return make(map[string]Result)
	}

	return results
}

// saveResult saves a single result incrementally
func (r *B) saveResult(result Result) {
	// Load current results to merge with
	current := r.loadResults()
	current[result.Name] = result

	// Save merged results
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		return
	}

	if err := os.WriteFile(r.filename, data, 0644); err != nil {
		fmt.Printf("Error writing results file: %v\n", err)
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
func (r *B) Run(name string, ourFn func(), refFn ...func()) {
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
		refSamples, _ := r.benchmark(refFn[0])
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
