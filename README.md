<p align="center">
<img width="300" height="100" src=".github/logo.png" border="0" alt="kelindar/bench">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/bench" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/bench"><img src="https://pkg.go.dev/badge/github.com/kelindar/bench" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/bench"><img src="https://goreportcard.com/badge/github.com/kelindar/bench" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://coveralls.io/github/kelindar/bench"><img src="https://coveralls.io/repos/github/kelindar/bench/badge.svg" alt="Coverage"></a>
</p>

## Bench: Statistical Benchmarking for Go

A **small, statistical benchmarking library** for Go, designed for robust, repeatable, and insightful performance analysis using BCa-style bootstrap inference.

- **Analyze performance** with bias-corrected and accelerated bootstrap intervals for median timing ratios
- **Persist results** incrementally in Gob format for resilience and tracking
- **Compare runs** and reference implementations with confidence intervals
- **Format output** in clean, customizable tables
- **Configurable** thresholds, sampling and other options for precise control

This library applies a **bias-corrected and accelerated** (BCa) bootstrap interval to the median timing ratio between independent sample sets. It resamples the raw measurements **100 000 times** (by default), evaluates `log(variant/control)`, then adjusts the percentile endpoints with bias correction and the multi-sample jackknife acceleration. Working in log-ratio space makes improvements and regressions symmetric and avoids absolute-duration thresholds that behave differently for fast and slow benchmarks.

Good practice is **25+ independent timings**; smaller n inflates the acceleration estimate and can widen intervals. Similarly, very heavy-tailed timing data can erode coverage and may need trimming or more samples. Benchmarks should be collected under stable conditions because CPU frequency changes, thermal drift, background load, cache state, and GC behavior can bias the samples before the bootstrap sees them. Reference comparisons are sampled in alternating order to reduce simple run-order drift.

The practical threshold is interpreted as a symmetric multiplicative timing ratio in log space: `WithThreshold(5)` requires the whole confidence interval to clear `log(1.05)` for regressions or `-log(1.05)` for improvements. Allocation indicators are simple median comparisons and are not confidence intervals.


**Use When**

* ✅ You want confidence-aware performance comparisons between Go implementations
* ✅ You need publication-quality statistical analysis with confidence intervals
* ✅ You need incremental, resilient result saving (e.g., for CI or long runs)
* ✅ You want to compare against previous or reference runs with clear significance
* ✅ You prefer clean, readable output and easy filtering
* ✅ You need to assert benchmarks in CI to avoid performance regressions

**Not For**

* ❌ Micro-benchmarks where Go's built-in `testing.B` is sufficient
* ❌ Long-term, distributed, or multi-process benchmarking
* ❌ Profiling memory/cpu in detail (use pprof for that)

### Example Output

```
name                 time/op      ops/s        allocs/op    vs prev             
-------------------- ------------ ------------ ------------ ------------------ 
find                 479.7 µs     2.1K         ✅ 0         ✅ +65%
sort                 47.4 ns      21.1M        🟰 1         🟰 similar
```

## Quick Start

```go
package main

import "github.com/kelindar/bench"

func main() {
    bench.Run(func(b *bench.B) {
        // Simple benchmark
        b.Run("benchmark name", func(i int) {
            // code to benchmark
        })

        // Benchmark with reference comparison
        b.Run("benchmark vs ref",
            func(i int) { /* our implementation */ },
            func(i int) { /* reference implementation */ })
    },
    bench.WithFile("results.json"),   // optional: set results file
    bench.WithFilter("set"),          // optional: only run benchmarks starting with "set"
    bench.WithConfidence(95.0),       // optional: set confidence level (default 99.9%)
    bench.WithThreshold(10.0),        // optional: require at least 10% practical change
    bench.WithBootstrap(50_000),      // optional: set bootstrap resamples
    bench.WithSeed(42),               // optional: mix in a deterministic bootstrap seed
    // Add more options as needed
    )
}
```

### Asserting Benchmarks in CI

Use `bench.Assert` inside your tests to automatically fail when a benchmark regresses compared to the previously recorded results. Assertions run in dry-run mode by default and are skipped when tests are executed with the `-short` flag.

```go
func TestPerformance(t *testing.T) {
    bench.Assert(t, func(b *bench.B) {
        b.Run("my-bench", func(i int) {
            // code to benchmark
        })
    }, bench.WithFile("baseline.json"))
}
```

## Options

The benchmark runner can be customized with a set of option functions. The table below explains what each option does and how you might use it.

| Option | Description |
|--------|-------------|
| `WithFile` | Use this to pick the file where benchmark results are stored. When the filename ends with `.gob`, the data is written in a compact binary format; otherwise JSON is used. Saving results lets you track performance over time or share them between machines. |
| `WithFilter` | Runs only the benchmarks whose names start with the provided prefix. This is handy when your suite has many benchmarks and you only want to focus on a subset without changing your code. |
| `WithSamples` | Sets how many samples should be collected for each benchmark. More samples give more stable statistics but also make the run take longer, so adjust the number depending on how precise you need the measurements to be. |
| `WithDuration` | Controls how long each sample runs. Increase the duration when the code under test is very fast or when you want less variation between runs. |
| `WithReference` | Enables the reference comparison column in the output. Provide a reference implementation when calling `b.Run` and Bench will show how your code performs against that reference, making regressions easy to spot. |
| `WithDryRun` | Prevents the library from writing results to disk. This option is useful for quick experiments or CI jobs where you just want to see the formatted output without updating any files. |
| `WithConfidence` | Sets the confidence level (in percent) for significance testing. Higher values make it harder for a difference to be considered statistically significant. |
| `WithThreshold` | Sets the minimum practical timing-ratio change (in percent) required before a statistically significant interval is reported as an improvement or regression. Raising this value is useful when unchanged code still shows run-to-run movement from machine noise. |
| `WithBootstrap` | Sets how many bootstrap resamples are used for comparisons. Increase this when using very high confidence levels; lower it for faster exploratory runs. |
| `WithSeed` | Mixes a user-provided seed into the deterministic bootstrap RNG. The default remains reproducible based on sample counts and bootstrap count. |

## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
