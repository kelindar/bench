package bench

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestFormatComparisonEdgeCases(t *testing.T) {
	b := &B{config: config{}}
	// No previous samples
	assert.Equal(t, "new", b.formatComparison([]float64{1, 2, 3}, nil))
	// Zero mean in reference
	assert.Contains(t, b.formatComparison([]float64{1, 2, 3}, []float64{0, 0, 0}), "inf")
}

func TestFormatComparisonBranches(t *testing.T) {
	b := &B{config: config{confidence: 99.9}}
	// Identical samples -> similar
	res := b.formatComparison([]float64{1, 2, 3, 4}, []float64{1, 2, 3, 4})
	assert.Equal(t, "🟰 similar", res)

	// Large but not significant difference
	res = b.formatComparison([]float64{1, 2, 3, 4}, []float64{2, 4, 6, 8})
	assert.True(t, strings.HasPrefix(res, "🟰 "))

	// Significant improvement
	b.confidence = 80
	res = b.formatComparison([]float64{1, 2, 3, 4}, []float64{2, 4, 6, 8})
	assert.True(t, strings.HasPrefix(res, "✅"))

	// Significant regression
	res = b.formatComparison([]float64{2, 4, 6, 8}, []float64{1, 2, 3, 4})
	assert.True(t, strings.HasPrefix(res, "❌"))
}
