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
