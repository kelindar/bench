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
	assert.Equal(t, "✅ 0", formatAllocsWithChange(0, allocBetter))
	assert.Equal(t, "❌ 1", formatAllocsWithChange(1, allocWorse))
	assert.Equal(t, "🟰 0", formatAllocsWithChange(0, allocSame))
	assert.Equal(t, "0", formatAllocsWithChange(0, allocUnknown))

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

func TestFormatComparison(t *testing.T) {
	b := &B{}
	for _, reason := range []string{"new", "changed", "invalid", "uncertain", "filtered"} {
		t.Run("inconclusive "+reason, func(t *testing.T) {
			assert.Equal(t, "❔ "+reason, b.formatComparison(Report{Inconclusive: reason}))
		})
	}

	tests := map[string]struct {
		report Report
		want   string
	}{
		"zero medians": {
			report: Report{},
			want:   "🟰 similar",
		},
		"variant extremely slower": {
			report: Report{MedianControl: 1, MedianVariant: 2000, Significant: true},
			want:   "❌ uncomparable",
		},
		"variant extremely faster": {
			report: Report{MedianControl: 1000, MedianVariant: 0.5, Significant: true},
			want:   "✅ uncomparable",
		},
		"improvement without interval suffix": {
			report: Report{MedianControl: 100, MedianVariant: 50, Ratio: 0.5, RatioCI: [2]float64{0.4, 0.6}, Significant: true},
			want:   "✅ +100%",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, b.formatComparison(tc.report))
		})
	}
}

func TestCompareAllocs(t *testing.T) {
	tests := map[string]struct {
		previous, current []float64
		want              allocChange
	}{
		"missing previous":        {nil, []float64{1}, allocUnknown},
		"same":                    {[]float64{1, 1, 1}, []float64{1, 1, 1}, allocSame},
		"better":                  {[]float64{2, 2, 2}, []float64{1, 1, 1}, allocBetter},
		"worse":                   {[]float64{1, 1, 1}, []float64{2, 2, 2}, allocWorse},
		"same displayed median":   {[]float64{35.8, 36.2}, []float64{36.1, 35.9}, allocSame},
		"same displayed rounding": {[]float64{35.5}, []float64{36.5}, allocSame},
		"same displayed thousand": {[]float64{1501}, []float64{1548}, allocSame},
		"same displayed fraction": {[]float64{0.1}, []float64{0.9}, allocSame},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, compareAllocs(tc.previous, tc.current))
		})
	}
}
