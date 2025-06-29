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

A **lightweight, statistical benchmarking library** for Go, designed for robust, repeatable, and insightful performance analysis using state-of-the-art BCa bootstrap inference. Bench makes it easy to:

- **Analyze performance** with BCa (Bias-Corrected accelerated) bootstrap for rigorous statistical significance
- **Persist results** incrementally in Gob format for resilience and tracking
- **Compare runs** and reference implementations with confidence intervals
- **Format output** in clean, customizable tables
- **Filter benchmarks** by name prefix for focused runs
- **Configure sampling** for precise control

**Use When**

* ✅ You want statistically rigorous performance comparisons between Go implementations
* ✅ You need publication-quality statistical analysis with confidence intervals
* ✅ You need incremental, resilient result saving (e.g., for CI or long runs)
* ✅ You want to compare against previous or reference runs with clear significance
* ✅ You prefer clean, readable output and easy filtering

**Not For**

* ❌ Micro-benchmarks where Go's built-in `testing.B` is sufficient
* ❌ Long-term, distributed, or multi-process benchmarking
* ❌ Profiling memory/cpu in detail (use pprof for that)

### Example Output

```
name              time/op     ops/s      allocs/op  vs prev            vs ref
----------------- ----------- ---------- ---------- ------------------ ------------------
and 1.0K (seq)    920.0 ns    1.1M       4          ✅ +7% (CI: +2% to +12%)   ❌ -18% (CI: -25% to -11%)
and 1.0K (rnd)    665.9 ns    1.5M       4          🟰 similar         ❌ -11% (CI: -18% to -4%)
and 1.0K (sps)    1.3 µs      754.5K     19         🟰 similar         🟰 -2% (CI: -8% to +4%)
and 1.0K (dns)    172.0 ns    5.8M       4          ❌ -7% (CI: -12% to -2%)   ❌ -18% (CI: -25% to -11%)
and 10.0M (seq)   191.3 µs    5.2K       156        🟰 +5% (CI: -1% to +11%)   ✅ +45% (CI: +38% to +52%)
and 10.0M (rnd)   274.1 µs    3.6K       176        ✅ +29% (CI: +22% to +36%)  ✅ +2% (CI: -5% to +9%)
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
    // Add more options as needed
    )
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
| `WithConfidence` | Sets the confidence level for the bootstrap confidence intervals (default 99.9%). Use 95.0 for 95% confidence intervals, which is common in many fields. |

## Statistical Method: BCa Bootstrap

Bench uses **BCa (Bias-Corrected accelerated) bootstrap inference** for all statistical comparisons:

- **Non-parametric method** using 10,000 bootstrap resamples by default
- **Bias-corrected and accelerated** confidence intervals
- **State-of-the-art method** used in recent benchmarking literature
- **Publication-quality results** suitable for rigorous analysis
- Shows results like `✅ +15% (CI: +8% to +23%)`
- **Significant if confidence interval excludes 0**

**Why BCa Bootstrap:**
- ✅ No assumptions about data distribution (works with any performance data)
- ✅ Provides intuitive confidence intervals instead of p-values
- ✅ Handles small or unequal sample sizes gracefully
- ✅ Automatically corrects for bias and skewness in the data
- ✅ Recommended by modern statistical literature for benchmarking

**Example:**
```go
bench.Run(func(b *bench.B) {
    b.Run("algorithm_v1", func(i int) { /* implementation */ })
    b.Run("algorithm_v2", func(i int) { /* implementation */ })
}, bench.WithConfidence(95.0)) // 95% confidence intervals
```

## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
