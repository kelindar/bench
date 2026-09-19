// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"math/bits"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStat(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		// Test with clearly different data sets
		control := []float64{10.0, 12.0, 11.0, 13.0, 9.0, 11.5, 10.5, 12.5}
		experiment := []float64{6.0, 7.0, 5.5, 6.5, 5.0, 6.0, 7.5, 6.2}

		result := bca(control, experiment, 0.95, 1000, defaultThreshold)

		// Basic validation
		assert.True(t, result.Delta < 0, "Expected negative delta (experiment faster)")
		assert.True(t, result.CI[0] < result.CI[1], "Lower CI should be less than upper CI")
		assert.True(t, result.Ratio < 1, "Expected experiment/control ratio below 1")
		assert.True(t, result.RatioCI[0] > 0, "Lower ratio CI should be positive")
		assert.True(t, result.RatioCI[0] < result.RatioCI[1], "Ratio CI should be ordered")
		assert.Equal(t, 0.95, result.Confidence, "Confidence level should match")
		assert.Equal(t, 1000, result.Samples, "Bootstrap samples should match")
		assert.True(t, result.Significant, "Should be significant with clear difference")
	})

	t.Run("identical", func(t *testing.T) {
		// Test with identical data (should not be significant)
		identical := []float64{10.0, 10.0, 10.0, 10.0, 10.0}
		result := bca(identical, identical, 0.95, 1000, defaultThreshold)

		assert.True(t, result.Degenerate, "Identical data should produce a degenerate bootstrap distribution")
		assert.False(t, result.Significant, "Identical data should not be significant")
		assert.InDelta(t, 0.0, result.Delta, 0.001, "Delta should be near zero for identical data")
		assert.True(t, result.CI[0] <= 0.0, "Lower CI should be <= 0")
		assert.True(t, result.CI[1] >= 0.0, "Upper CI should be >= 0")
	})

	t.Run("degenerate", func(t *testing.T) {
		control := []float64{100, 100, 100, 100}
		experiment := []float64{80, 80, 80, 80}

		result := bca(control, experiment, 0.95, 1000, defaultThreshold)

		assert.True(t, result.Degenerate, "Constant bootstrap distribution should be reported")
		assert.False(t, result.Significant, "Degenerate bootstrap distribution should be inconclusive")
		assert.InDelta(t, math.Log(0.8), result.Delta, 1e-12)
	})

	t.Run("small difference", func(t *testing.T) {
		// Test with small difference that might not be significant
		control := []float64{10.0, 10.1, 9.9, 10.0, 10.1}
		experiment := []float64{10.05, 10.15, 9.95, 10.05, 10.15}

		result := bca(control, experiment, 0.95, 1000, defaultThreshold)

		// Should have reasonable CI bounds
		assert.True(t, result.CI[0] < result.CI[1], "Lower CI should be less than upper CI")
		assert.InDelta(t, math.Log(10.05/10.0), result.Delta, 0.01, "Delta should be the log median ratio")
	})

	t.Run("edge cases", func(t *testing.T) {
		// Test with empty slices
		result := bca([]float64{}, []float64{1.0}, 0.95, 100, defaultThreshold)
		assert.Equal(t, "no samples", result.Inconclusive)

		result = bca([]float64{1.0}, []float64{}, 0.95, 100, defaultThreshold)
		assert.Equal(t, "no samples", result.Inconclusive)

		// Test with single values
		result = bca([]float64{5.0}, []float64{10.0}, 0.95, 100, defaultThreshold)
		assert.InDelta(t, math.Log(2), result.Delta, 1e-15, "Delta should be log(2)")
		assert.InDelta(t, 2.0, result.Ratio, 1e-15, "Ratio should be 2.0")
		assert.Equal(t, 0.95, result.Confidence, "Confidence should match")
	})

	t.Run("consistency", func(t *testing.T) {
		// Test that identical data gives consistent results
		data := []float64{10.0, 10.1, 9.9, 10.0, 10.05}

		// Run multiple times - should be consistent due to deterministic seeding
		result1 := bca(data, data, 0.95, 1000, defaultThreshold)
		result2 := bca(data, data, 0.95, 1000, defaultThreshold)

		assert.Equal(t, result1.Delta, result2.Delta, "Should get identical deltas")
		assert.Equal(t, result1.Significant, result2.Significant, "Should get identical significance")
		assert.False(t, result1.Significant, "Identical data should not be significant")
	})

	t.Run("practical significance", func(t *testing.T) {
		// Test that small differences are not considered practically significant
		control := []float64{100.0, 100.1, 99.9, 100.0}
		experiment := []float64{103.0, 103.1, 102.9, 103.0} // 3% difference (below 5% threshold)

		result := bca(control, experiment, 0.95, 1000, defaultThreshold)

		// Should not be significant due to conservative practical significance threshold (5%)
		assert.False(t, result.Significant, "Small differences (< 5%) should not be practically significant")

		// Test with larger difference that should be significant
		control2 := []float64{100.0, 100.1, 99.9, 100.0, 99.8, 100.2, 100.0, 100.1}
		experiment2 := []float64{90.0, 90.1, 89.9, 90.0, 89.8, 90.2, 90.0, 90.1} // 10% difference

		result2 := bca(control2, experiment2, 0.95, 1000, defaultThreshold)

		// Should be significant due to large practical difference
		assert.True(t, result2.Significant, "Large differences (> 5%) should be practically significant")
	})

	t.Run("threshold", func(t *testing.T) {
		control := []float64{99.9, 100.0, 100.0, 100.1, 100.1, 99.8, 100.2, 100.0}
		experiment := []float64{93.9, 94.0, 94.0, 94.1, 94.1, 93.8, 94.2, 94.0}

		assert.True(t, bca(control, experiment, 0.95, 1000, 5).Significant)
		assert.False(t, bca(control, experiment, 0.95, 1000, 10).Significant)
	})

	t.Run("bias correction", func(t *testing.T) {
		assert.InDelta(t, 0, computeBiasCorrection(1, []float64{1, 1, 1, 1}), 1e-12)
	})

	t.Run("acceleration", func(t *testing.T) {
		control := []float64{10, 11, 12, 13, 14}
		experiment := []float64{7, 8, 9, 10, 15, 16, 20}

		assert.InDelta(t, 0.015157626068123558, computeAcceleration(control, experiment), 1e-15)
	})

	t.Run("confidence", func(t *testing.T) {
		result := bca([]float64{1, 2, 3}, []float64{2, 3, 4}, -1, 100, defaultThreshold)

		assert.Equal(t, defaultConfidence/100.0, result.Confidence)
	})

	t.Run("median", func(t *testing.T) {
		data := []float64{3, 1, 2}

		assert.Equal(t, 2.0, median(data))
		assert.Equal(t, []float64{3, 1, 2}, data)
	})
}

func TestStability(t *testing.T) {
	t.Run("insufficient observations", func(t *testing.T) {
		result := bca([]float64{99, 101}, []float64{119, 121}, 0.999, 10000, 5)
		assert.False(t, result.Significant, "two observations cannot establish a population median change at 99.9 percent confidence")
		assert.Equal(t, "too few samples", result.Inconclusive)
	})

	t.Run("invalid observation", func(t *testing.T) {
		for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1)} {
			result := bca([]float64{99, 100, 101, invalid}, []float64{119, 120, 121, 122}, 0.95, 1000, 5)
			assert.False(t, result.Significant, "invalid timings must not be silently discarded: %v", invalid)
		}
	})

	t.Run("sustained load is confounded", func(t *testing.T) {
		control, busy := make([]float64, 100), make([]float64, 100)
		for i := range control {
			control[i] = 100 + float64((i*5)%11)
			busy[i] = 1.25 * control[i]
		}
		result := bca(control, busy, 0.999, 10000, 5)
		assert.True(t, result.Significant, "timings alone cannot distinguish CPU slowdown from code slowdown")
		assert.InDelta(t, 1.25, result.Ratio, 1e-12)
	})

	t.Run("clustered timings", func(t *testing.T) {
		control, variant := make([]float64, 100), make([]float64, 100)
		for i := range control {
			control[i] = 100 + float64(i)/100
			variant[i] = 1.25 * control[i]
		}
		result := bca(control, variant, 0.999, 1000, 5)
		assert.False(t, result.Significant, "ordered drift does not provide 100 independent observations")
		assert.Equal(t, "correlated timings", result.Inconclusive)
	})
}

func TestMedianInterval(t *testing.T) {
	// For n=10, P(Binomial(10, 0.5) <= 1) = 11/1024 < 0.025,
	// while P(Binomial(10, 0.5) <= 2) = 56/1024 > 0.025.
	data := []float64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	low, high := medianInterval(data, 0.025)
	assert.Equal(t, 2.0, low)
	assert.Equal(t, 9.0, high)
	assert.Equal(t, 10.0, data[0], "input order must be preserved")
	low, high = medianInterval(data, 0.00025)
	assert.Zero(t, low)
	assert.True(t, math.IsInf(high, 1))
}

func TestCoverage(t *testing.T) {
	// Fixed-seed simulation exercises coverage with skew and occasional stalls.
	// This checks the implementation; it does not validate serial independence.
	for _, shape := range []string{"lognormal", "stalls"} {
		t.Run(shape, func(t *testing.T) {
			rng := rand.New(rand.NewPCG(31, 47))
			misses := 0
			const trials = 500
			for trial := 0; trial < trials; trial++ {
				control, variant := make([]float64, 40), make([]float64, 60)
				for group, data := range [][]float64{control, variant} {
					for i := range data {
						data[i] = math.Exp(0.2 * rng.NormFloat64())
						if shape == "stalls" && rng.Float64() < 0.1 {
							data[i] *= 10
						}
						if group == 1 {
							data[i] *= 1.2
						}
					}
				}
				report := bca(control, variant, 0.95, 200, 5)
				if report.RatioCI[0] > 1.2 || report.RatioCI[1] < 1.2 {
					misses++
				}
			}
			t.Logf("%s: %d/%d intervals missed the true ratio", shape, misses, trials)
			assert.LessOrEqual(t, misses, 25, "conservative intervals should attain at least nominal coverage in this fixed simulation")
		})
	}
}

func TestClustered(t *testing.T) {
	// Of the 252 balanced sequences of length 10, only the two sequences
	// with two runs have lower-tail probability below 1% (2/252).
	flagged := 0
	for mask := uint(0); mask < 1<<10; mask++ {
		if bits.OnesCount(mask) != 5 {
			continue
		}
		data := make([]float64, 10)
		for i := range data {
			data[i] = float64((mask>>i)&1) * 2
		}
		if clustered(data, 1) {
			flagged++
		}
	}
	assert.Equal(t, 2, flagged)
	assert.False(t, clustered([]float64{1, 1, 1}, 1))
}

func TestDependence(t *testing.T) {
	rng := rand.New(rand.NewPCG(21, 79))
	flagged, inconclusive := 0, 0
	for trial := 0; trial < 100; trial++ {
		control, variant := make([]float64, 100), make([]float64, 100)
		for _, data := range [][]float64{control, variant} {
			x := rng.NormFloat64()
			for i := range data {
				x = 0.9*x + math.Sqrt(1-0.9*0.9)*rng.NormFloat64()
				data[i] = math.Exp(0.3 * x)
			}
		}
		report := bca(control, variant, 0.999, 200, 0)
		if report.Significant {
			flagged++
		}
		if report.Inconclusive != "" {
			inconclusive++
		}
	}
	assert.Zero(t, flagged, "equal population medians with correlated timings must not produce false flags in this simulation")
	assert.GreaterOrEqual(t, inconclusive, 95)
}
