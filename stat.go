// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"math/rand/v2"
	"sort"

	"gonum.org/v1/gonum/stat/distuv"
)

// Report represents the result of BCa Report inference
type Report struct {
	Delta         float64    // Delta is the difference between the medians (new - old)
	CI            [2]float64 // Interval is the confidence interval
	MedianControl float64    // MedianControl is the median of the control group
	MedianVariant float64    // MedianVariant is the median of the variant group
	Confidence    float64    // Confidence is the confidence level (e.g., 0.95 for 95%)
	Significant   bool       // Significant indicates if the confidence interval excludes zero
	Samples       int        // Samples is the number of bootstrap samples used
}

// bca performs BCa (Bias-Corrected accelerated) bootstrap inference
// comparing two samples. Returns confidence interval for the difference in medians.
func bca(control, experiment []float64, confidence float64, bootstrapSamples int) Report {
	if len(control) == 0 || len(experiment) == 0 {
		return Report{}
	}
	// Use more stable seeding based on sample statistics rather than raw values
	// This reduces sensitivity to measurement noise
	medianControl := median(control)
	medianVariant := median(experiment)
	seed := uint64(math.Float64bits(medianControl) ^ math.Float64bits(medianVariant))
	rng := rand.New(rand.NewPCG(seed, seed+1))

	// Original statistic (difference in medians)
	originalDelta := medianVariant - medianControl

	// Step 1: Bootstrap resampling
	bootstrapDeltas := make([]float64, bootstrapSamples)
	for i := 0; i < bootstrapSamples; i++ {

		// Resample with replacement using our seeded RNG
		controlBootstrap := resampleWithReplacement(control, rng)
		variantBootstrap := resampleWithReplacement(experiment, rng)

		// Compute statistic for this bootstrap sample
		controlBootMedian := median(controlBootstrap)
		variantBootMedian := median(variantBootstrap)
		bootstrapDeltas[i] = variantBootMedian - controlBootMedian
	}

	// Step 2: Compute bias-correction parameter (z₀)
	biasCorrection := computeBiasCorrection(originalDelta, bootstrapDeltas)

	// Step 3: Compute acceleration parameter (â) using jackknife
	acceleration := computeAcceleration(control, experiment)

	// Step 4: Compute BCa confidence interval
	alpha := 1.0 - confidence
	lowerCI, upperCI := computeBCaCI(bootstrapDeltas, biasCorrection, acceleration, alpha)

	// Step 5: More conservative significance detection
	significant := isSignificant(lowerCI, upperCI, originalDelta, medianControl, medianVariant)

	return Report{
		Delta:         originalDelta,
		CI:            [2]float64{lowerCI, upperCI},
		MedianControl: medianControl,
		MedianVariant: medianVariant,
		Confidence:    confidence,
		Significant:   significant,
		Samples:       bootstrapSamples,
	}
}

// isSignificant uses more conservative thresholds to reduce false positives
func isSignificant(lowerCI, upperCI, delta, controlMedian, experimentMedian float64) bool {
	if controlMedian == 0 || experimentMedian == 0 {
		return false // Don't claim significance for edge cases
	}

	// CI must clearly exclude 0 with larger tolerance. Use 5% of the control
	// median as minimum detectable difference
	mde := math.Abs(controlMedian * 0.05)
	tolerance := math.Max(mde, math.Abs(delta)*0.1)
	statsig := lowerCI > tolerance || upperCI < -tolerance
	return statsig
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

// median calculates the median of a slice of float64.
func median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	n := len(data)
	sort.Float64s(data)
	if n%2 == 1 {
		return data[n/2]
	}

	mid1 := data[n/2-1]
	mid2 := data[n/2]
	return (mid1 + mid2) / 2.0
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
		controlJack[i] = median(jackSample)
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
		experimentJack[i] = median(jackSample)
	}

	// Compute jackknife differences
	jackDiffs := make([]float64, n1+n2)
	for i := 0; i < n1; i++ {
		jackDiffs[i] = median(experiment) - controlJack[i]
	}
	for i := 0; i < n2; i++ {
		jackDiffs[n1+i] = experimentJack[i] - median(control)
	}

	// Compute acceleration parameter
	jackMedian := median(jackDiffs)

	sumCubed := 0.0
	sumSquared := 0.0
	for _, diff := range jackDiffs {
		dev := jackMedian - diff
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
	alpha1 = max(0, min(alpha1, 1))
	alpha2 = max(0, min(alpha2, 1))

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
