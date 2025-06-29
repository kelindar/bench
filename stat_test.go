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

	result := BCaBootstrap(control, experiment, 0.95, 1000)

	// Basic validation
	assert.True(t, result.Delta < 0, "Expected negative delta (experiment faster)")
	assert.True(t, result.LowerCI < result.UpperCI, "Lower CI should be less than upper CI")
	assert.Equal(t, 0.95, result.Confidence, "Confidence level should match")
	assert.Equal(t, 1000, result.Samples, "Bootstrap samples should match")
	assert.True(t, result.Significant, "Should be significant with clear difference")
}

func TestBCaBootstrapIdentical(t *testing.T) {
	t.Parallel()

	// Test with identical data (should not be significant)
	identical := []float64{10.0, 10.0, 10.0, 10.0, 10.0}
	result := BCaBootstrap(identical, identical, 0.95, 1000)

	assert.False(t, result.Significant, "Identical data should not be significant")
	assert.InDelta(t, 0.0, result.Delta, 0.001, "Delta should be near zero for identical data")
	assert.True(t, result.LowerCI <= 0.0, "Lower CI should be <= 0")
	assert.True(t, result.UpperCI >= 0.0, "Upper CI should be >= 0")
}

func TestBCaBootstrapSmallDifference(t *testing.T) {
	t.Parallel()

	// Test with small difference that might not be significant
	control := []float64{10.0, 10.1, 9.9, 10.0, 10.1}
	experiment := []float64{10.05, 10.15, 9.95, 10.05, 10.15}

	result := BCaBootstrap(control, experiment, 0.95, 1000)

	// Should have reasonable CI bounds
	assert.True(t, result.LowerCI < result.UpperCI, "Lower CI should be less than upper CI")
	assert.InDelta(t, 0.05, result.Delta, 0.1, "Delta should be around 0.05")
}

func TestBCaBootstrapEdgeCases(t *testing.T) {
	t.Parallel()

	// Test with empty slices
	result := BCaBootstrap([]float64{}, []float64{1.0}, 0.95, 100)
	assert.Equal(t, boostrap{}, result, "Should return empty result for empty control")

	result = BCaBootstrap([]float64{1.0}, []float64{}, 0.95, 100)
	assert.Equal(t, boostrap{}, result, "Should return empty result for empty experiment")

	// Test with single values
	result = BCaBootstrap([]float64{5.0}, []float64{10.0}, 0.95, 100)
	assert.Equal(t, 5.0, result.Delta, "Delta should be 5.0")
	assert.Equal(t, 0.95, result.Confidence, "Confidence should match")
}
