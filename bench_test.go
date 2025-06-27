package bench

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestWithOptions(t *testing.T) {
	cfg := config{}
	WithFile("foo.json")(&cfg)
	WithFilter("bar")(&cfg)
	WithSamples(42)(&cfg)
	WithDuration(123 * time.Millisecond)(&cfg)
	WithReference()(&cfg)

	if cfg.filename != "foo.json" || cfg.filter != "bar" || cfg.samples != 42 || cfg.duration != 123*time.Millisecond || !cfg.showRef {
		t.Fatalf("options not set correctly: %+v", cfg)
	}
}

func TestFormatHelpers(t *testing.T) {
	b := &B{config: config{}}
	if b.formatAllocs(1500) != "1.5K" {
		t.Error("formatAllocs K")
	}
	if b.formatAllocs(10) != "10" {
		t.Error("formatAllocs int")
	}
	if b.formatAllocs(0.5) != "0" {
		t.Error("formatAllocs zero")
	}

	if !strings.Contains(b.formatTime(2e6), "ms") {
		t.Error("formatTime ms")
	}
	if !strings.Contains(b.formatTime(2e3), "µs") {
		t.Error("formatTime us")
	}
	if !strings.Contains(b.formatTime(2), "ns") {
		t.Error("formatTime ns")
	}

	if !strings.Contains(b.formatOps(2e6), "M") {
		t.Error("formatOps M")
	}
	if !strings.Contains(b.formatOps(2e3), "K") {
		t.Error("formatOps K")
	}
	if b.formatOps(2) != "2" {
		t.Error("formatOps int")
	}
}

func TestShouldRun(t *testing.T) {
	b := &B{config: config{filter: "foo"}}
	if !b.shouldRun("foobar") {
		t.Error("shouldRun true")
	}
	if b.shouldRun("bar") {
		t.Error("shouldRun false")
	}
	b.filter = ""
	if !b.shouldRun("anything") {
		t.Error("shouldRun empty filter")
	}
}

func TestSaveLoadResult(t *testing.T) {
	file := "test_bench.json"
	defer os.Remove(file)
	b := &B{config: config{filename: file}}
	res := Result{Name: "bench", Samples: []float64{1, 2, 3}, Timestamp: 123}
	b.saveResult(res)
	loaded := b.loadResults()
	if loaded["bench"].Timestamp != 123 {
		t.Fatalf("loadResults failed: %+v", loaded)
	}
}

func TestRunAndFiltering(t *testing.T) {
	file := "test_bench2.json"
	defer os.Remove(file)
	var ran, ranRef bool
	Run(func(b *B) {
		b.Run("foo", func(*B) { ran = true })
		b.Run("bar", func(*B) {}, func(*B) { ranRef = true })
	}, WithFile(file), WithFilter("foo"))
	if !ran {
		t.Error("filtered benchmark did not run")
	}
	if ranRef {
		t.Error("filtered out benchmark ran")
	}
}

func TestRunWithReferenceAndNoPrev(t *testing.T) {
	file := "test_bench3.json"
	defer os.Remove(file)
	Run(func(b *B) {
		b.Run("bench", func(*B) {}, func(*B) {})
	}, WithFile(file), WithReference())
	// Just check that it doesn't panic and file is created
	if _, err := os.Stat(file); err != nil {
		t.Error("results file not created")
	}
}

func TestFormatComparisonEdgeCases(t *testing.T) {
	b := &B{config: config{}}
	// No previous samples
	if b.formatComparison([]float64{1, 2, 3}, nil) != "new" {
		t.Error("formatComparison new")
	}
	// Zero mean in reference
	if !strings.Contains(b.formatComparison([]float64{1, 2, 3}, []float64{0, 0, 0}), "inf") {
		t.Error("formatComparison inf")
	}
}
