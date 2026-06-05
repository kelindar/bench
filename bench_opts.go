// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"flag"
	"os"
	"strings"
	"time"
)

// Option configures the benchmark runner
// It mutates the internal config used by Run.
type Option func(*config)

// config holds runtime configuration for benchmarks.
type config struct {
	filename   string
	filter     string
	samples    int
	duration   time.Duration
	tableFmt   string
	showRef    bool
	dryRun     bool
	confidence float64
	threshold  float64
	bootstrap  int
	seed       uint64
	codec      codec
}

func (c *config) normalize() {
	if c.filename == "" {
		c.filename = defaultFilename
	}
	if c.samples < minSamples {
		c.samples = minSamples
	}
	if c.duration <= 0 {
		c.duration = defaultDuration
	}
	if c.tableFmt == "" {
		c.tableFmt = defaultTableFmt
	}
	if !isFinite(c.confidence) || c.confidence <= 0 || c.confidence >= 100 {
		c.confidence = defaultConfidence
	}
	if !isFinite(c.threshold) || c.threshold < 0 {
		c.threshold = 0
	}
	if c.bootstrap <= 0 {
		c.bootstrap = defaultBootstrap
	}
	if c.codec == nil {
		if strings.HasSuffix(c.filename, ".gob") {
			c.codec = gobCodec{}
		} else {
			c.codec = jsonCodec{}
		}
	}
}

// WithFile sets the filename for benchmark results
func WithFile(filename string) Option {
	return func(c *config) {
		c.filename = filename
		if strings.HasSuffix(filename, ".gob") {
			c.codec = gobCodec{}
		} else {
			c.codec = jsonCodec{}
		}
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
		if n < minSamples {
			n = minSamples
		}
		c.samples = n
	}
}

// WithDuration sets the duration for each sample
func WithDuration(d time.Duration) Option {
	return func(c *config) {
		if d <= 0 {
			d = defaultDuration
		}
		c.duration = d
	}
}

// WithReference enables reference comparison column
func WithReference() Option {
	return func(c *config) {
		c.showRef = true
	}
}

// WithDryRun disables writing benchmark results to disk
func WithDryRun() Option {
	return func(c *config) {
		c.dryRun = true
	}
}

// WithConfidence sets the confidence level for statistical significance tests
func WithConfidence(level float64) Option {
	return func(c *config) {
		if !isFinite(level) || level <= 0 || level >= 100 {
			level = defaultConfidence
		}
		c.confidence = level
	}
}

// WithThreshold sets the minimum practical change, in percent, required before
// a statistically significant interval is reported as an improvement/regression.
func WithThreshold(percent float64) Option {
	return func(c *config) {
		if !isFinite(percent) || percent < 0 {
			percent = 0
		}
		c.threshold = percent
	}
}

// WithBootstrap sets the number of bootstrap resamples used for comparisons.
func WithBootstrap(n int) Option {
	return func(c *config) {
		if n <= 0 {
			n = defaultBootstrap
		}
		c.bootstrap = n
	}
}

// WithSeed sets an additional seed mixed into the deterministic bootstrap RNG.
func WithSeed(seed uint64) Option {
	return func(c *config) {
		c.seed = seed
	}
}

// initFlags parses command-line flags and applies them to the config. It
// recognizes "-bench" to filter benchmarks by prefix and "-n" for dry runs.
func initFlags(c *config) {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	prefix := fs.String("bench", "", "Run only benchmarks with this prefix")
	dry := fs.Bool("n", false, "dry run - do not update bench.gob")

	// Parse only the flags we care about from os.Args
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

	if *prefix != "" {
		c.filter = *prefix
	}
	if *dry {
		c.dryRun = true
	}
}
