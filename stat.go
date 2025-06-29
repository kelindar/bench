// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"math/rand/v2"
	"sort"

	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// bootstrap represents the result of BCa bootstrap inference
type bootstrap struct {
	Delta       float64 // Delta is the difference between the means (new - old)
	LowerCI     float64 // LowerCI is the lower bound of the confidence interval
	UpperCI     float64 // UpperCI is the upper bound of the confidence interval
	Confidence  float64 // Confidence is the confidence level (e.g., 0.95 for 95%)
	Significant bool    // Significant indicates if the confidence interval excludes zero
	Samples     int     // Samples is the number of bootstrap samples used
}

// BCaBootstrap performs BCa (Bias-Corrected accelerated) bootstrap inference
// comparing two samples. Returns confidence interval for the difference in means.
func BCaBootstrap(control, experiment []float64, confidence float64, bootstrapSamples int) bootstrap {
	if len(control) == 0 || len(experiment) == 0 {
		return bootstrap{}
	}

	// Use deterministic seeding based on data for reproducible results within session
	seed1, seed2 := hashSamples(control, experiment)
	rng := rand.New(rand.NewPCG(seed1, seed2))

	// Original statistic (difference in means)
	controlMean := stat.Mean(control, nil)
	experimentMean := stat.Mean(experiment, nil)
	originalDelta := experimentMean - controlMean

	// Step 1: Bootstrap resampling
	bootstrapDeltas := make([]float64, bootstrapSamples)
	for i := 0; i < bootstrapSamples; i++ {
		// Resample with replacement using our seeded RNG
		controlBootstrap := resampleWithReplacement(control, rng)
		experimentBootstrap := resampleWithReplacement(experiment, rng)

		// Compute statistic for this bootstrap sample
		controlBootMean := stat.Mean(controlBootstrap, nil)
		experimentBootMean := stat.Mean(experimentBootstrap, nil)
		bootstrapDeltas[i] = experimentBootMean - controlBootMean
	}

	// Step 2: Compute bias-correction parameter (z₀)
	biasCorrection := computeBiasCorrection(originalDelta, bootstrapDeltas)

	// Step 3: Compute acceleration parameter (â) using jackknife
	acceleration := computeAcceleration(control, experiment)

	// Step 4: Compute BCa confidence interval
	alpha := 1.0 - confidence
	lowerCI, upperCI := computeBCaCI(bootstrapDeltas, biasCorrection, acceleration, alpha)

	// Step 5: Determine significance with tolerance for jittering
	significant := isSignificant(lowerCI, upperCI, originalDelta, controlMean, experimentMean)

	return bootstrap{
		Delta:       originalDelta,
		LowerCI:     lowerCI,
		UpperCI:     upperCI,
		Confidence:  confidence,
		Significant: significant,
		Samples:     bootstrapSamples,
	}
}

// hashSamples creates a deterministic seed from sample data
func hashSamples(control, experiment []float64) (uint64, uint64) {
	var hash1, hash2 uint64 = 0x9e3779b9, 0x85ebca6b

	for _, v := range control {
		bits := math.Float64bits(v)
		hash1 ^= bits + 0x9e3779b9 + (hash1 << 6) + (hash1 >> 2)
	}

	for _, v := range experiment {
		bits := math.Float64bits(v)
		hash2 ^= bits + 0x85ebca6b + (hash2 << 6) + (hash2 >> 2)
	}

	return hash1, hash2
}

// isSignificant determines significance with tolerance to prevent jittering
func isSignificant(lowerCI, upperCI, delta, controlMean, experimentMean float64) bool {
	// If either mean is zero, use absolute difference threshold
	if controlMean == 0 || experimentMean == 0 {
		return math.Abs(delta) > 0.1 && (lowerCI > 0 || upperCI < 0)
	}

	// Calculate percentage effect size
	percentEffect := math.Abs(delta / controlMean * 100)

	// Practical significance threshold: effect must be > 1% and CI must exclude 0
	practicallySignificant := percentEffect > 1.0

	// Statistical significance: CI must clearly exclude 0 (with tolerance)
	tolerance := math.Max(0.01, math.Abs(delta)*0.01)
	statistically := lowerCI > tolerance || upperCI < -tolerance

	// Both conditions must be met
	return practicallySignificant && statistically
}

// resampleWithReplacement performs bootstrap resampling with replacement using provided RNG
func resampleWithReplacement(data []float64, rng *rand.Rand) []float64 {
	n := len(data)
	resampled := make([]float64, n)
	for i := 0; i < n; i++ {
		idx := rng.IntN(n)
		resampled[i] = data[idx]
	}
	return resampled
}

// computeBiasCorrection computes the bias-correction parameter z₀
func computeBiasCorrection(originalStat float64, bootstrapStats []float64) float64 {
	// Count how many bootstrap statistics are less than the original
	count := 0
	for _, bootStat := range bootstrapStats {
		if bootStat < originalStat {
			count++
		}
	}

	// Proportion of bootstrap statistics less than original
	proportion := float64(count) / float64(len(bootstrapStats))

	// Avoid edge cases
	if proportion <= 0 {
		proportion = 1.0 / (2.0 * float64(len(bootstrapStats)))
	} else if proportion >= 1 {
		proportion = 1.0 - 1.0/(2.0*float64(len(bootstrapStats)))
	}

	// z₀ is the inverse normal of the proportion
	dist := distuv.UnitNormal
	return dist.Quantile(proportion)
}

// computeAcceleration computes the acceleration parameter â using jackknife
func computeAcceleration(control, experiment []float64) float64 {
	n1, n2 := len(control), len(experiment)

	// Jackknife estimates for control group
	controlJack := make([]float64, n1)
	for i := 0; i < n1; i++ {
		// Create jackknife sample (all except i-th element)
		jackSample := make([]float64, 0, n1-1)
		for j := 0; j < n1; j++ {
			if j != i {
				jackSample = append(jackSample, control[j])
			}
		}
		controlJack[i] = stat.Mean(jackSample, nil)
	}

	// Jackknife estimates for experiment group
	experimentJack := make([]float64, n2)
	for i := 0; i < n2; i++ {
		// Create jackknife sample (all except i-th element)
		jackSample := make([]float64, 0, n2-1)
		for j := 0; j < n2; j++ {
			if j != i {
				jackSample = append(jackSample, experiment[j])
			}
		}
		experimentJack[i] = stat.Mean(jackSample, nil)
	}

	// Compute jackknife differences
	jackDiffs := make([]float64, n1+n2)
	for i := 0; i < n1; i++ {
		jackDiffs[i] = stat.Mean(experiment, nil) - controlJack[i]
	}
	for i := 0; i < n2; i++ {
		jackDiffs[n1+i] = experimentJack[i] - stat.Mean(control, nil)
	}

	// Compute acceleration parameter
	jackMean := stat.Mean(jackDiffs, nil)

	sumCubed := 0.0
	sumSquared := 0.0
	for _, diff := range jackDiffs {
		dev := jackMean - diff
		sumCubed += dev * dev * dev
		sumSquared += dev * dev
	}

	if sumSquared == 0 {
		return 0
	}

	acceleration := sumCubed / (6.0 * math.Pow(sumSquared, 1.5))
	return acceleration
}

// computeBCaCI computes the BCa confidence interval
func computeBCaCI(bootstrapStats []float64, biasCorrection, acceleration, alpha float64) (float64, float64) {
	// Sort bootstrap statistics
	sortedStats := make([]float64, len(bootstrapStats))
	copy(sortedStats, bootstrapStats)
	sort.Float64s(sortedStats)

	n := float64(len(sortedStats))
	dist := distuv.UnitNormal

	// Compute adjusted percentiles
	z_alpha2 := dist.Quantile(alpha / 2.0)
	z_1minus_alpha2 := dist.Quantile(1.0 - alpha/2.0)

	// BCa adjustments
	alpha1 := dist.CDF(biasCorrection + (biasCorrection+z_alpha2)/(1.0-acceleration*(biasCorrection+z_alpha2)))
	alpha2 := dist.CDF(biasCorrection + (biasCorrection+z_1minus_alpha2)/(1.0-acceleration*(biasCorrection+z_1minus_alpha2)))

	// Ensure valid percentiles
	if alpha1 < 0 {
		alpha1 = 0
	}
	if alpha1 > 1 {
		alpha1 = 1
	}
	if alpha2 < 0 {
		alpha2 = 0
	}
	if alpha2 > 1 {
		alpha2 = 1
	}

	// Get percentiles from sorted bootstrap statistics
	idx1 := int(math.Floor(alpha1 * n))
	idx2 := int(math.Floor(alpha2 * n))

	// Handle edge cases
	if idx1 >= len(sortedStats) {
		idx1 = len(sortedStats) - 1
	}
	if idx2 >= len(sortedStats) {
		idx2 = len(sortedStats) - 1
	}
	if idx1 < 0 {
		idx1 = 0
	}
	if idx2 < 0 {
		idx2 = 0
	}

	return sortedStats[idx1], sortedStats[idx2]
}
