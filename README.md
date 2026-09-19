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

A statistical benchmarking library for Go with saved baselines, CPU calibration, and conservative confidence intervals.

- **Analyze performance** with bias-corrected and accelerated bootstrap intervals for median timing ratios
- **Persist results** incrementally in Gob format for resilience and tracking
- **Compare runs** and reference implementations with confidence intervals
- **Format output** in clean, customizable tables
- **Configurable** thresholds, sampling and other options for precise control

This library applies a **bias-corrected and accelerated** (BCa) bootstrap interval to the median timing ratio. It resamples the raw measurements **100 000 times** by default, evaluates `log(variant/control)`, then adjusts the percentile endpoints with bias correction and the multi-sample jackknife acceleration. Working in log-ratio space makes improvements and regressions symmetric.

BCa can be overconfident with few observations or tied values. The reported interval is therefore widened to include non-interpolated [binomial order-statistic bounds for the two population medians](https://itl.nist.gov/div898/software/dataplot/refman1/auxillar/mediancl.htm). Each of the four tails receives one quarter of the error budget. Under independent, identically distributed observations, these bounds provide at least the requested coverage. This is deliberately conservative: at the default 99.9% confidence, fewer than 12 observations in either group cannot establish a change, regardless of the number of bootstrap resamples.

Timings must be finite and positive. An exact lower-tail [runs test](https://www.itl.nist.gov/div898/handbook/eda/section3/eda35d.htm) at 1% screens for clustering above and below the median, omitting ties. Detected clustering produces an `uncertain` result. This screen catches some drift and serial dependence; passing it does **not** prove independence. Confidence levels apply to individual comparisons, not to an entire benchmark suite or repeated CI runs.

The practical threshold is interpreted as a symmetric multiplicative timing ratio in log space: `WithThreshold(5)` requires the whole confidence interval to clear `log(1.05)` for regressions or `-log(1.05)` for improvements. Allocation indicators are simple median comparisons and are not confidence intervals.

### Comparing saved baselines under CPU load

Every run records the hostname, CPU model, OS/architecture, Go version, selected build flags, CPU count, GOMAXPROCS, live GC settings, sample duration, and calibration version. [gopsutil](https://github.com/shirou/gopsutil) supplies CPU model and utilization measurements. `CPUUsage` is the average system utilization during collection, including the benchmark itself; `-1` means unavailable.

A fixed CPU workload is sampled once per benchmark sample, alternating before and after the benchmark. Saved-baseline comparisons require matching environment metadata and a calibration confidence interval entirely within the tolerance set by `WithThreshold`, 5% by default. Calibration approximately doubles sampling time for a benchmark without a live reference. Raw benchmark timings and allocation counts exclude the calibration work.

The library keeps the measured timing ratio and displays `❔ <code>` whenever it cannot support a verdict. `Report.Inconclusive` returns the same one-word code:

| Code | Meaning |
|------|---------|
| `new` | No saved baseline exists yet. |
| `changed` | The machine, runtime, build settings, sample duration, or calibration context differs. |
| `invalid` | A timing, calibration, or benchmark input is invalid. |
| `uncertain` | The data cannot support a practical change or equivalence verdict. |
| `filtered` | The benchmark was filtered out. |

When the entire interval is within the practical tolerance, the output is `🟰 similar`.

Inconclusive comparisons have `Report.Significant == false` and a nonempty `Report.Inconclusive`. They preserve a usable saved baseline. Legacy files and inadequate baselines can be refreshed by a writable run; an inadequate baseline is replaced only when the new samples pass the quality checks. For an intentional environment change, use a new baseline filename. Saved reference files receive the same checks. Live references still run in alternating order and use conservative median bounds.

The returned `Report` describes the comparison with the previous saved result. A first run prints `❔ new` and returns `Inconclusive: "new"`; a filtered benchmark returns `Inconclusive: "filtered"`.

Calibration is a comparability check, not a correction factor. An integer CPU workload cannot account for every cache, memory, I/O, thermal, or scheduling effect. CPU utilization is diagnostic only. Keep baseline and current runs as comparable as possible; use a dedicated runner when small regressions must be distinguished reliably. Increasing bootstrap resamples cannot repair biased or dependent measurements.


**Use When**

* ✅ You want confidence-aware performance comparisons between Go implementations
* ✅ You want conservative confidence intervals with explicit inconclusive results
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

To compare against another run saved as `.gob`, omit the reference function and use `bench.WithReference("reference.gob")` alongside `bench.WithFile("results.gob")`.

### Asserting Benchmarks in CI

Use `bench.Assert` inside your tests to automatically fail when a benchmark regresses compared to the previously recorded results. Assertions run in dry-run mode by default and are skipped when tests are executed with the `-short` flag.

Only a supported regression fails the test. An inconclusive result is printed and returned without failing; this is **not** evidence that performance passed. CI callers that require a conclusive result can check `Report.Inconclusive` and apply their own retry or failure policy. Establish the baseline with a writable `bench.Run` first.

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
| `WithReference` | Enables the reference comparison column. Pass a saved `.gob` or JSON filename to compare against stored results, or pass a reference implementation to `b.Run`. |
| `WithDryRun` | Prevents the library from writing results to disk. This option is useful for quick experiments or CI jobs where you just want to see the formatted output without updating any files. |
| `WithConfidence` | Sets the confidence level (in percent) for significance testing. Higher values make it harder for a difference to be considered statistically significant. |
| `WithThreshold` | Sets the minimum practical timing-ratio change and the allowed CPU calibration variation, in percent. Defaults to 5%. Zero requires exact calibration equivalence and is generally unsuitable for saved-baseline comparisons. |
| `WithBootstrap` | Sets how many bootstrap resamples are used for comparisons. Increase this when using very high confidence levels; lower it for faster exploratory runs. |
| `WithSeed` | Mixes a user-provided seed into the deterministic bootstrap RNG. The default remains reproducible based on sample counts and bootstrap count. |

## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
