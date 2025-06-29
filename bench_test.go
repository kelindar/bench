// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithOptions(t *testing.T) {
	cfg := config{}
	WithFile("foo.json")(&cfg)
	WithFilter("bar")(&cfg)
	WithSamples(42)(&cfg)
	WithDuration(123 * time.Millisecond)(&cfg)
	WithReference()(&cfg)
	WithDryRun()(&cfg)
	WithConfidence(95.5)(&cfg)

	assert.Equal(t, "foo.json", cfg.filename)
	assert.Equal(t, "bar", cfg.filter)
	assert.Equal(t, 42, cfg.samples)
	assert.Equal(t, 123*time.Millisecond, cfg.duration)
	assert.True(t, cfg.showRef)
	assert.True(t, cfg.dryRun)
	_, ok := cfg.codec.(jsonCodec)
	assert.True(t, ok)
	assert.InDelta(t, 95.5, cfg.confidence, 0.001)
}

func TestShouldRun(t *testing.T) {
	b := &B{config: config{filter: "foo"}}
	assert.True(t, b.shouldRun("foobar"))
	assert.False(t, b.shouldRun("bar"))
	b.filter = ""
	assert.True(t, b.shouldRun("anything"))
}

func TestRunAndFiltering(t *testing.T) {
	file := "test_bench2.json"
	defer os.Remove(file)
	var ran, ranRef bool
	Run(func(b *B) {
		b.Run("foo", func(i int) { ran = true })
		b.Run("bar", func(i int) {}, func(i int) { ranRef = true })
	}, WithFile(file), WithFilter("foo"))
	assert.True(t, ran, "filtered benchmark did not run")
	assert.False(t, ranRef, "filtered out benchmark ran")
}

func TestRunWithReferenceAndNoPrev(t *testing.T) {
	file := "test_bench3.json"
	defer os.Remove(file)
	Run(func(b *B) {
		b.Run("bench", func(i int) {}, func(i int) {})
	}, WithFile(file), WithReference())
	_, err := os.Stat(file)
	assert.NoError(t, err, "results file not created")
}

func TestRunDryRun(t *testing.T) {
	file := "test_bench_dry.json"
	defer os.Remove(file)
	Run(func(b *B) {
		b.Run("bench", func(i int) {})
	}, WithFile(file), WithDryRun())
	_, err := os.Stat(file)
	assert.Error(t, err, "results file should not be created")
}

func TestBCaBootstrap(t *testing.T) {
	// Create test data with known difference
	control := []float64{10.0, 12.0, 11.0, 13.0, 9.0, 11.5, 10.5, 12.5}
	experiment := []float64{8.0, 9.0, 7.5, 8.5, 7.0, 8.0, 9.5, 8.2}

	// Run BCa bootstrap with 95% confidence
	result := bca(control, experiment, 0.95, 1000)

	// Check that we get reasonable results
	assert.True(t, result.Delta < 0, "Expected negative delta (experiment faster)")
	assert.True(t, result.CI[0] < result.CI[1], "Lower CI should be less than upper CI")
	assert.Equal(t, 0.95, result.Confidence, "Confidence level should match")
	assert.Equal(t, 1000, result.Samples, "Bootstrap samples should match")

	// The difference should be significant given the clear separation
	assert.True(t, result.Significant, "Difference should be significant")

	// Test with identical data (should not be significant)
	identical := []float64{10.0, 10.0, 10.0, 10.0}
	result2 := bca(identical, identical, 0.95, 1000)
	assert.False(t, result2.Significant, "Identical data should not be significant")
	assert.InDelta(t, 0.0, result2.Delta, 0.001, "Delta should be near zero for identical data")
}

func TestRunWithBCaBootstrap(t *testing.T) {
	file := "test_bca_bootstrap.json"
	defer os.Remove(file)

	// Test that benchmark execution works with BCa bootstrap (always enabled)
	Run(func(b *B) {
		b.Run("test_bca", func(i int) {
			time.Sleep(time.Microsecond) // Simulate some work
		})
	}, WithFile(file), WithSamples(10))

	// Verify results file was created
	_, err := os.Stat(file)
	assert.NoError(t, err, "results file should be created")
}
