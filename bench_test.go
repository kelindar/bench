// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("options", func(t *testing.T) {
		cfg := config{}
		WithFile("foo.json")(&cfg)
		WithFilter("bar")(&cfg)
		WithSamples(42)(&cfg)
		WithDuration(123 * time.Millisecond)(&cfg)
		WithReference("ref.gob")(&cfg)
		WithDryRun()(&cfg)
		WithConfidence(95.5)(&cfg)
		WithThreshold(7.5)(&cfg)
		WithBootstrap(1234)(&cfg)
		WithSeed(99)(&cfg)

		assert.Equal(t, "foo.json", cfg.filename)
		assert.Equal(t, "bar", cfg.filter)
		assert.Equal(t, 42, cfg.samples)
		assert.Equal(t, 123*time.Millisecond, cfg.duration)
		assert.True(t, cfg.showRef)
		assert.Equal(t, "ref.gob", cfg.referenceFilename)
		assert.True(t, cfg.dryRun)
		_, ok := cfg.codec.(jsonCodec)
		assert.True(t, ok)
		assert.InDelta(t, 95.5, cfg.confidence, 0.001)
		assert.InDelta(t, 7.5, cfg.threshold, 0.001)
		assert.Equal(t, 1234, cfg.bootstrap)
		assert.Equal(t, uint64(99), cfg.seed)
	})

	t.Run("invalid options", func(t *testing.T) {
		cfg := config{}

		WithSamples(0)(&cfg)
		WithDuration(0)(&cfg)
		WithConfidence(math.NaN())(&cfg)
		WithThreshold(-1)(&cfg)
		WithBootstrap(0)(&cfg)

		assert.Equal(t, minSamples, cfg.samples)
		assert.Equal(t, defaultDuration, cfg.duration)
		assert.Equal(t, defaultConfidence, cfg.confidence)
		assert.Equal(t, 0.0, cfg.threshold)
		assert.Equal(t, defaultBootstrap, cfg.bootstrap)
	})

	t.Run("normalize", func(t *testing.T) {
		cfg := config{
			samples:    -1,
			duration:   -1,
			confidence: math.Inf(1),
			threshold:  -1,
			bootstrap:  -1,
		}

		cfg.normalize()

		assert.Equal(t, defaultFilename, cfg.filename)
		assert.Equal(t, minSamples, cfg.samples)
		assert.Equal(t, defaultDuration, cfg.duration)
		assert.Equal(t, defaultTableFmt, cfg.tableFmt)
		assert.Equal(t, defaultConfidence, cfg.confidence)
		assert.Equal(t, 0.0, cfg.threshold)
		assert.Equal(t, defaultBootstrap, cfg.bootstrap)
		_, ok := cfg.codec.(gobCodec)
		assert.True(t, ok)
	})

	t.Run("filter", func(t *testing.T) {
		b := &B{config: config{filter: "foo"}}
		assert.True(t, b.shouldRun("foobar"))
		assert.False(t, b.shouldRun("bar"))
		b.filter = ""
		assert.True(t, b.shouldRun("anything"))
	})

	t.Run("flags preserve config", func(t *testing.T) {
		oldArgs := os.Args
		t.Cleanup(func() {
			os.Args = oldArgs
		})
		os.Args = []string{"test"}

		cfg := config{dryRun: true, filter: "keep"}
		initFlags(&cfg)

		assert.True(t, cfg.dryRun)
		assert.Equal(t, "keep", cfg.filter)
	})
}

func TestRun(t *testing.T) {
	t.Run("filtering", func(t *testing.T) {
		file := "test_bench2.json"
		defer os.Remove(file)
		var ran, ranRef bool
		Run(func(b *B) {
			b.Run("foo", func(i int) { ran = true })
			report := b.Run("bar", func(i int) {}, func(i int) { ranRef = true })
			assert.Equal(t, "filtered", report.Inconclusive)
		}, WithFile(file), WithFilter("foo"))
		assert.True(t, ran, "filtered benchmark did not run")
		assert.False(t, ranRef, "filtered out benchmark ran")
	})

	t.Run("reference without previous", func(t *testing.T) {
		file := "test_bench3.json"
		defer os.Remove(file)
		Run(func(b *B) {
			b.Run("bench", func(i int) {}, func(i int) {})
		}, WithFile(file), WithReference())
		_, err := os.Stat(file)
		assert.NoError(t, err, "results file not created")
	})

	t.Run("gob reference", func(t *testing.T) {
		referenceFile := "test_reference.gob"
		resultFile := "test_reference_current.gob"
		defer os.Remove(referenceFile)
		defer os.Remove(resultFile)

		err := gobCodec{}.save(referenceFile, map[string]Result{
			"bench": {Samples: []float64{0, 0}},
		})
		assert.NoError(t, err)

		oldStdout := os.Stdout
		reader, writer, err := os.Pipe()
		assert.NoError(t, err)
		os.Stdout = writer
		t.Cleanup(func() {
			os.Stdout = oldStdout
			reader.Close()
			writer.Close()
		})

		Run(func(b *B) {
			b.Run("bench", func(i int) {})
		}, WithFile(resultFile), WithReference(referenceFile), WithSamples(2), WithDuration(time.Nanosecond), WithBootstrap(10))

		assert.NoError(t, writer.Close())
		os.Stdout = oldStdout
		output, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(output), "⚠️ no calibration"), "legacy reference must be inconclusive: %s", output)
	})

	t.Run("dry run", func(t *testing.T) {
		file := "test_bench_dry.json"
		defer os.Remove(file)
		Run(func(b *B) {
			b.Run("bench", func(i int) {})
		}, WithFile(file), WithDryRun())
		_, err := os.Stat(file)
		assert.Error(t, err, "results file should not be created")
	})

	t.Run("bca bootstrap", func(t *testing.T) {
		file := "test_bca_bootstrap.json"
		defer os.Remove(file)

		// Test that benchmark execution works with BCa bootstrap (always enabled)
		Run(func(b *B) {
			report := b.Run("test_bca", func(i int) {
				time.Sleep(time.Microsecond) // Simulate some work
			})
			assert.Equal(t, "no baseline", report.Inconclusive)
		}, WithFile(file), WithSamples(10))

		// Verify results file was created
		_, err := os.Stat(file)
		assert.NoError(t, err, "results file should be created")

		loaded := jsonCodec{}.load(file)
		assert.Len(t, loaded["test_bca"].Allocs, 10, "allocation samples should be saved with timing samples")
	})

	t.Run("positive ops", func(t *testing.T) {
		file := "test_runn_invalid.json"
		defer os.Remove(file)

		assert.PanicsWithValue(t, "bench: RunN function must return a positive operation count", func() {
			Run(func(b *B) {
				b.RunN("bad", func(i int) int {
					return 0
				})
			}, WithFile(file), WithSamples(2), WithDuration(time.Nanosecond))
		})
	})

	t.Run("ops overflow", func(t *testing.T) {
		maxInt := int(^uint(0) >> 1)

		assert.PanicsWithValue(t, "bench: RunN operation count overflow", func() {
			addOps(maxInt, 1)
		})
	})

	t.Run("assert", func(t *testing.T) {
		file := "test_assert.json"
		defer os.Remove(file)

		// baseline run to create previous results
		Run(func(b *B) {
			b.Run("bench", func(i int) { time.Sleep(time.Millisecond) })
		}, WithFile(file), WithSamples(5), WithDuration(time.Millisecond))

		before, err := os.Stat(file)
		assert.NoError(t, err)

		// Assert should pass with identical performance and not modify file
		Assert(t, func(b *B) {
			b.Run("bench", func(i int) { time.Sleep(time.Millisecond) })
		}, WithFile(file), WithSamples(5), WithDuration(time.Millisecond))

		after, err := os.Stat(file)
		assert.NoError(t, err)
		assert.Equal(t, before.ModTime(), after.ModTime(), "file should not be modified")
	})
}

func TestBCaBootstrap(t *testing.T) {
	// Create test data with known difference
	control := []float64{10.0, 12.0, 11.0, 13.0, 9.0, 11.5, 10.5, 12.5}
	experiment := []float64{6.0, 7.0, 5.5, 6.5, 5.0, 6.0, 7.5, 6.2}

	// Run BCa bootstrap with 95% confidence
	result := bca(control, experiment, 0.95, 1000, defaultThreshold)

	// Check that we get reasonable results
	assert.True(t, result.Delta < 0, "Expected negative delta (experiment faster)")
	assert.True(t, result.CI[0] < result.CI[1], "Lower CI should be less than upper CI")
	assert.Equal(t, 0.95, result.Confidence, "Confidence level should match")
	assert.Equal(t, 1000, result.Samples, "Bootstrap samples should match")

	// The difference should be significant given the clear separation
	assert.True(t, result.Significant, "Difference should be significant")

	// Test with identical data (should not be significant)
	identical := []float64{10.0, 10.0, 10.0, 10.0}
	result2 := bca(identical, identical, 0.95, 1000, defaultThreshold)
	assert.False(t, result2.Significant, "Identical data should not be significant")
	assert.InDelta(t, 0.0, result2.Delta, 0.001, "Delta should be near zero for identical data")
}

func TestCompare(t *testing.T) {
	env := Environment{Hostname: "test", GOOS: "test", GOARCH: "test", GoVersion: "test",
		CPUModel: "test", NumCPU: 8, GOMAXPROCS: 8, DurationNS: 10000000, CalibrationVersion: calibrationVersion,
		Build: "test", GOGC: "100", GOMEMLIMIT: "9223372036854775807"}
	for _, name := range []string{"same", "regression", "improvement", "busy CPU", "noisy CPU", "legacy", "changed setup", "unknown setup", "invalid calibration", "bad baseline"} {
		t.Run(name, func(t *testing.T) {
			previous := Result{Environment: env}
			current := Result{Environment: env}
			for i := 0; i < 100; i++ {
				previous.Samples = append(previous.Samples, 100+float64((i*3)%7))
				current.Samples = append(current.Samples, 100+float64((i*3)%7))
				previous.Calibration = append(previous.Calibration, 10+float64((i*3)%7)/100)
				current.Calibration = append(current.Calibration, 10+float64((i*3)%7)/100)
			}
			wantReason := ""
			switch name {
			case "regression", "busy CPU", "noisy CPU", "legacy", "changed setup", "unknown setup", "invalid calibration":
				for i := range current.Samples {
					current.Samples[i] *= 1.25
				}
			case "improvement":
				for i := range current.Samples {
					current.Samples[i] *= 0.8
				}
			}
			switch name {
			case "busy CPU":
				for i := range current.Calibration {
					current.Calibration[i] *= 1.25
				}
				wantReason = "CPU unstable"
			case "noisy CPU":
				for i := range current.Calibration {
					current.Calibration[i] *= 0.5 + float64(i%2)
				}
				wantReason = "CPU unstable"
			case "legacy":
				previous.Calibration = nil
				wantReason = "no calibration"
			case "changed setup":
				current.Environment.GOMAXPROCS++
				wantReason = "setup changed"
			case "unknown setup":
				current.Environment.CPUModel = ""
				wantReason = "unknown setup"
			case "invalid calibration":
				current.Calibration = current.Calibration[:1]
				wantReason = "invalid calibration"
			case "bad baseline":
				previous.Samples = previous.Samples[:2]
				previous.Calibration = previous.Calibration[:2]
				wantReason = "baseline incomplete"
			}
			cfg := defaultConfig()
			cfg.bootstrap = 1000
			runner := &B{config: cfg}
			report := runner.compare(previous, current)
			assert.Equal(t, wantReason, report.Inconclusive)
			assert.Equal(t, name == "regression" || name == "improvement", report.Significant)
			if name == "busy CPU" {
				assert.InDelta(t, 1.25, report.Ratio, 1e-12, "retain the raw slowdown, do not normalize it away")
			}
			if name == "bad baseline" {
				assert.False(t, runner.usable(previous))
				assert.True(t, runner.usable(current), "a fresh run can repair an inadequate baseline")
			}
		})
	}
}

func TestBaseline(t *testing.T) {
	file := filepath.Join(t.TempDir(), "baseline.json")
	previous := Result{Name: "bench", Samples: []float64{1, 2}, Calibration: []float64{1, 2}, Timestamp: 123}
	require.NoError(t, (jsonCodec{}).save(file, map[string]Result{"bench": previous}))
	Run(func(b *B) {
		report := b.Run("bench", func(int) {})
		assert.Equal(t, "unknown setup", report.Inconclusive)
		assert.False(t, report.Significant)
	}, WithFile(file), WithSamples(2), WithDuration(time.Nanosecond), WithBootstrap(10))
	assert.Equal(t, previous, (jsonCodec{}).load(file)["bench"], "an inconclusive run must not replace the baseline")
}
