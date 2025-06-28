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

	assert.Equal(t, "foo.json", cfg.filename)
	assert.Equal(t, "bar", cfg.filter)
	assert.Equal(t, 42, cfg.samples)
	assert.Equal(t, 123*time.Millisecond, cfg.duration)
	assert.True(t, cfg.showRef)
}

func TestFormatHelpers(t *testing.T) {
	b := &B{config: config{}}
	assert.Equal(t, "1.5K", b.formatAllocs(1500))
	assert.Equal(t, "10", b.formatAllocs(10))
	assert.Equal(t, "0", b.formatAllocs(0.5))

	assert.Contains(t, b.formatTime(2e6), "ms")
	assert.Contains(t, b.formatTime(2e3), "µs")
	assert.Contains(t, b.formatTime(2), "ns")

	assert.Contains(t, b.formatOps(2e6), "M")
	assert.Contains(t, b.formatOps(2e3), "K")
	assert.Equal(t, "2", b.formatOps(2))
}

func TestShouldRun(t *testing.T) {
	b := &B{config: config{filter: "foo"}}
	assert.True(t, b.shouldRun("foobar"))
	assert.False(t, b.shouldRun("bar"))
	b.filter = ""
	assert.True(t, b.shouldRun("anything"))
}

func TestSaveLoadResult(t *testing.T) {
	file := "test_bench.json"
	defer os.Remove(file)
	b := &B{config: config{filename: file}}
	res := Result{Name: "bench", Samples: []float64{1, 2, 3}, Timestamp: 123}
	b.saveResult(res)
	loaded := b.loadResults()
	assert.Equal(t, int64(123), loaded["bench"].Timestamp)
}

func TestRunAndFiltering(t *testing.T) {
	file := "test_bench2.json"
	defer os.Remove(file)
	var ran, ranRef bool
	Run(func(b *B) {
		b.Run("foo", func(b *B, op int) { ran = true })
		b.Run("bar", func(b *B, op int) {}, func(b *B, op int) { ranRef = true })
	}, WithFile(file), WithFilter("foo"))
	assert.True(t, ran, "filtered benchmark did not run")
	assert.False(t, ranRef, "filtered out benchmark ran")
}

func TestRunWithReferenceAndNoPrev(t *testing.T) {
	file := "test_bench3.json"
	defer os.Remove(file)
	Run(func(b *B) {
		b.Run("bench", func(b *B, op int) {}, func(b *B, op int) {})
	}, WithFile(file), WithReference())
	_, err := os.Stat(file)
	assert.NoError(t, err, "results file not created")
}

func TestFormatComparisonEdgeCases(t *testing.T) {
	b := &B{config: config{}}
	// No previous samples
	assert.Equal(t, "new", b.formatComparison([]float64{1, 2, 3}, nil))
	// Zero mean in reference
	assert.Contains(t, b.formatComparison([]float64{1, 2, 3}, []float64{0, 0, 0}), "inf")
}
