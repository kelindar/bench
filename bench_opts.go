package bench

import "time"

// Option configures the benchmark runner
// It mutates the internal config used by Run.
type Option func(*config)

// config holds runtime configuration for benchmarks.
type config struct {
	filename string
	filter   string
	samples  int
	duration time.Duration
	tableFmt string
	showRef  bool
	dryRun   bool
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

// WithDryRun disables writing benchmark results to disk
func WithDryRun() Option {
	return func(c *config) {
		c.dryRun = true
	}
}
