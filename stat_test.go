// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBCaBootstrapBasic(t *testing.T) {
	t.Parallel()

	// Test with clearly different data sets
	control := []float64{10.0, 12.0, 11.0, 13.0, 9.0, 11.5, 10.5, 12.5}
	experiment := []float64{8.0, 9.0, 7.5, 8.5, 7.0, 8.0, 9.5, 8.2}

	result := bca(control, experiment, 0.95, 1000)

	// Basic validation
	assert.True(t, result.Delta < 0, "Expected negative delta (experiment faster)")
	assert.True(t, result.CI[0] < result.CI[1], "Lower CI should be less than upper CI")
	assert.Equal(t, 0.95, result.Confidence, "Confidence level should match")
	assert.Equal(t, 1000, result.Samples, "Bootstrap samples should match")
	assert.True(t, result.Significant, "Should be significant with clear difference")
}

func TestBCaBootstrapIdentical(t *testing.T) {
	t.Parallel()

	// Test with identical data (should not be significant)
	identical := []float64{10.0, 10.0, 10.0, 10.0, 10.0}
	result := bca(identical, identical, 0.95, 1000)

	assert.False(t, result.Significant, "Identical data should not be significant")
	assert.InDelta(t, 0.0, result.Delta, 0.001, "Delta should be near zero for identical data")
	assert.True(t, result.CI[0] <= 0.0, "Lower CI should be <= 0")
	assert.True(t, result.CI[1] >= 0.0, "Upper CI should be >= 0")
}

func TestBCaBootstrapSmallDifference(t *testing.T) {
	t.Parallel()

	// Test with small difference that might not be significant
	control := []float64{10.0, 10.1, 9.9, 10.0, 10.1}
	experiment := []float64{10.05, 10.15, 9.95, 10.05, 10.15}

	result := bca(control, experiment, 0.95, 1000)

	// Should have reasonable CI bounds
	assert.True(t, result.CI[0] < result.CI[1], "Lower CI should be less than upper CI")
	assert.InDelta(t, 0.05, result.Delta, 0.1, "Delta should be around 0.05")
}

func TestBCaBootstrapEdgeCases(t *testing.T) {
	t.Parallel()

	// Test with empty slices
	result := bca([]float64{}, []float64{1.0}, 0.95, 100)
	assert.Equal(t, Report{}, result, "Should return empty result for empty control")

	result = bca([]float64{1.0}, []float64{}, 0.95, 100)
	assert.Equal(t, Report{}, result, "Should return empty result for empty experiment")

	// Test with single values
	result = bca([]float64{5.0}, []float64{10.0}, 0.95, 100)
	assert.Equal(t, 5.0, result.Delta, "Delta should be 5.0")
	assert.Equal(t, 0.95, result.Confidence, "Confidence should match")
}

func TestBCaBootstrapConsistency(t *testing.T) {
	t.Parallel()

	// Test that identical data gives consistent results
	data := []float64{10.0, 10.1, 9.9, 10.0, 10.05}

	// Run multiple times - should be consistent due to deterministic seeding
	result1 := bca(data, data, 0.95, 1000)
	result2 := bca(data, data, 0.95, 1000)

	assert.Equal(t, result1.Delta, result2.Delta, "Should get identical deltas")
	assert.Equal(t, result1.Significant, result2.Significant, "Should get identical significance")
	assert.False(t, result1.Significant, "Identical data should not be significant")
}

func TestBCaBootstrapPracticalSignificance(t *testing.T) {
	t.Parallel()

	// Test that small differences are not considered practically significant
	control := []float64{100.0, 100.1, 99.9, 100.0}
	experiment := []float64{103.0, 103.1, 102.9, 103.0} // 3% difference (below 5% threshold)

	result := bca(control, experiment, 0.95, 1000)

	// Should not be significant due to conservative practical significance threshold (5%)
	assert.False(t, result.Significant, "Small differences (< 5%) should not be practically significant")

	// Test with larger difference that should be significant
	control2 := []float64{100.0, 100.1, 99.9, 100.0}
	experiment2 := []float64{90.0, 90.1, 89.9, 90.0} // 10% difference (above 5% threshold)

	result2 := bca(control2, experiment2, 0.95, 1000)

	// Should be significant due to large practical difference
	assert.True(t, result2.Significant, "Large differences (> 5%) should be practically significant")
}
