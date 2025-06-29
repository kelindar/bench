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

A **small, statistical benchmarking library** for Go, designed for robust, repeatable, and insightful performance analysis using state-of-the-art BCa bootstrap inference. 

- **Analyze performance** with BCa (Bias-Corrected and Accelerated) bootstrap for rigorous statistical significance
- **Persist results** incrementally in Gob format for resilience and tracking
- **Compare runs** and reference implementations with confidence intervals
- **Format output** in clean, customizable tables
- **Configurable** thresholds, sampling and other options for precise control

This library applies the **bias-corrected and accelerated** (BCa) bootstrap to every set of timings. It shuffles the raw measurements **10 000 times** (by default), then adjusts the percentile endpoints with the bias and the acceleration. BCa is non-parametric and enjoys second-order accuracy, so its confidence interval keeps nominal coverage without assuming any particular distribution and stays stable in the presence of moderate skew. 

However, good practice is **25+ independent timings**; smaller n inflates the acceleration estimate and can widen intervals. Similarly, very heavy-tailed timing data can erode coverage and may need trimming or more samples.


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
name                 time/op      ops/s        allocs/op    vs prev             
-------------------- ------------ ------------ ------------ ------------------ 
find                 479.7 µs     2.1K         0            ✅ +65% [-33%,-24%]
sort                 47.4 ns      21.1M        1            🟰 similar
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
| `WithConfidence` | Sets the confidence level (in percent) for significance testing. Higher values make it harder for a difference to be considered statistically significant. |

## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
