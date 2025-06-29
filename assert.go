package bench

import (
	"testing"
)

// Assert runs benchmarks in dry-run mode and fails the test if performance regresses.
// It is skipped when testing is run with -short.
func Assert(t *testing.T, fn func(*B), opts ...Option) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping benchmark assertion in short mode")
	}

	cfg := config{
		filename:   DefaultFilename,
		samples:    DefaultSamples,
		duration:   DefaultDuration,
		tableFmt:   DefaultTableFmt,
		confidence: DefaultConfidence,
		codec:      gobCodec{},
	}

	initFlags(&cfg)
	cfg.dryRun = true

	for _, opt := range opts {
		opt(&cfg)
	}

	runner := &B{config: cfg, t: t}
	runner.printHeader()
	fn(runner)
}
