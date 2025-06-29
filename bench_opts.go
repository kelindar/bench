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
	codec      codec
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

// WithConfidence sets the confidence level for statistical significance tests
func WithConfidence(level float64) Option {
	return func(c *config) {
		c.confidence = level
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

	c.filter = *prefix
	c.dryRun = *dry
}
