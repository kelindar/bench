// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatHelpers(t *testing.T) {
	assert.Equal(t, "1.5K", formatAllocs(1500))
	assert.Equal(t, "10", formatAllocs(10))
	assert.Equal(t, "0", formatAllocs(0.5))

	assert.Contains(t, formatTime(2e6), "ms")
	assert.Contains(t, formatTime(2e3), "µs")
	assert.Contains(t, formatTime(2), "ns")

	assert.Contains(t, formatOps(2e6), "M")
	assert.Contains(t, formatOps(2e3), "K")
	assert.Equal(t, "2", formatOps(2))
}

func TestFormatChange(t *testing.T) {
	// Large speedups should be formatted as multipliers
	assert.Equal(t, "+3.5x", formatChange(250))
	assert.Equal(t, "+13x", formatChange(1200))

	// Percent formatting with interval
	out := formatChange(10)
	assert.Equal(t, "+10%", out)
}

func TestFormatComparisonCases(t *testing.T) {
	b := &B{}

	// Zero means
	r := Report{}
	assert.Equal(t, "🟰 similar", b.formatComparison(r))

	// Variant extremely slower
	r = Report{MedianControl: 1, MedianVariant: 2000, Significant: true}
	assert.Equal(t, "❌ uncomparable", b.formatComparison(r))

	// Variant extremely faster
	r = Report{MedianControl: 1000, MedianVariant: 0.5, Significant: true}
	assert.Equal(t, "✅ uncomparable", b.formatComparison(r))

	// Typical improvement with confidence interval
	r = Report{MedianControl: 100, MedianVariant: 50, Significant: true, CI: [2]float64{-60, -40}}
	assert.Contains(t, b.formatComparison(r), "+100%")
}
