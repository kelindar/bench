package bench

import (
	"fmt"
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

func TestWithFileGob(t *testing.T) {
	cfg := config{}
	WithFile("foo.gob")(&cfg)
	_, ok := cfg.codec.(gobCodec)
	assert.True(t, ok)
}

func TestInitFlags(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"cmd", "-bench=foo", "-n"}
	cfg := config{}
	initFlags(&cfg)
	assert.Equal(t, "foo", cfg.filter)
	assert.True(t, cfg.dryRun)
}

func TestRunN(t *testing.T) {
	file := "test_runn.json"
	defer os.Remove(file)
	var count int
	Run(func(b *B) {
		b.RunN("bench", func(i int) int { count++; return 1 })
	}, WithFile(file), WithSamples(1), WithDuration(time.Millisecond))
	assert.Greater(t, count, 0)
}

type failCodec struct{}

func (failCodec) load(string) map[string]Result        { return map[string]Result{} }
func (failCodec) save(string, map[string]Result) error { return fmt.Errorf("fail") }

func TestSaveResultError(t *testing.T) {
	file := "fail.json"
	defer os.Remove(file)
	b := &B{config: config{filename: file, codec: failCodec{}}}
	b.saveResult(Result{Name: "bench"})
	_, err := os.Stat(file)
	assert.Error(t, err)
}

func TestLoadResultsMissing(t *testing.T) {
	b := &B{config: config{filename: "does_not_exist.json", codec: jsonCodec{}}}
	res := b.loadResults()
	assert.Equal(t, 0, len(res))
}

func TestLoadResultsDefaultCodec(t *testing.T) {
	b := &B{config: config{filename: "does_not_exist.json"}}
	res := b.loadResults()
	assert.Equal(t, 0, len(res))
	_, ok := b.codec.(jsonCodec)
	assert.True(t, ok)
}

func TestSaveResultDefaultCodec(t *testing.T) {
	file := "default.json"
	defer os.Remove(file)
	b := &B{config: config{filename: file}}
	b.saveResult(Result{Name: "bench"})
	_, err := os.Stat(file)
	assert.NoError(t, err)
}
