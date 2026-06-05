// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"math/rand/v2"
	"sort"

	"gonum.org/v1/gonum/stat/distuv"
)

// Report represents the result of BCa Report inference.
type Report struct {
	Delta         float64    // Delta is log(MedianVariant / MedianControl); positive is slower
	CI            [2]float64 // CI is the confidence interval for Delta
	Ratio         float64    // Ratio is MedianVariant / MedianControl
	RatioCI       [2]float64 // RatioCI is exp(CI)
	MedianControl float64    // MedianControl is the median of the control group
	MedianVariant float64    // MedianVariant is the median of the variant group
	Confidence    float64    // Confidence is the confidence level (e.g., 0.95 for 95%)
	Significant   bool       // Significant indicates statistical and practical significance
	Degenerate    bool       // Degenerate indicates a bootstrap distribution without variation
	Samples       int        // Samples is the number of bootstrap samples used
}

// bca performs BCa (Bias-Corrected accelerated) bootstrap inference comparing
// two samples. The test statistic is the log median time ratio.
func bca(control, experiment []float64, confidence float64, bootstrapSamples int, minChangePercent float64) Report {
	return bcaWithSeed(control, experiment, confidence, bootstrapSamples, minChangePercent, 0)
}

func bcaWithSeed(control, experiment []float64, confidence float64, bootstrapSamples int, minChangePercent float64, seed uint64) Report {
	if len(control) == 0 || len(experiment) == 0 {
		return Report{}
	}
	if bootstrapSamples <= 0 {
		return Report{}
	}
	confidence = normalizeConfidence(confidence)

	medianControl := median(control)
	medianVariant := median(experiment)
	originalLogRatio, ok := logRatio(medianControl, medianVariant)
	if !ok {
		return Report{
			MedianControl: medianControl,
			MedianVariant: medianVariant,
			Confidence:    confidence,
			Samples:       bootstrapSamples,
		}
	}
	rng := bootstrapRNG(len(control), len(experiment), bootstrapSamples, seed)

	bootstrapStats := make([]float64, 0, bootstrapSamples)
	for i := 0; i < bootstrapSamples; i++ {

		// Resample with replacement using our seeded RNG
		controlBootstrap := resampleWithReplacement(control, rng)
		variantBootstrap := resampleWithReplacement(experiment, rng)

		// Compute statistic for this bootstrap sample
		controlBootMedian := medianInPlace(controlBootstrap)
		variantBootMedian := medianInPlace(variantBootstrap)
		if stat, ok := logRatio(controlBootMedian, variantBootMedian); ok {
			bootstrapStats = append(bootstrapStats, stat)
		}
	}
	if len(bootstrapStats) == 0 {
		return Report{
			Delta:         originalLogRatio,
			Ratio:         math.Exp(originalLogRatio),
			MedianControl: medianControl,
			MedianVariant: medianVariant,
			Confidence:    confidence,
			Degenerate:    true,
			Samples:       bootstrapSamples,
		}
	}

	biasCorrection := computeBiasCorrection(originalLogRatio, bootstrapStats)

	acceleration := computeAcceleration(control, experiment)
	degenerate := degenerateBootstrap(bootstrapStats)

	// Step 4: Compute BCa confidence interval
	alpha := 1.0 - confidence
	lowerCI, upperCI := computeBCaCI(bootstrapStats, biasCorrection, acceleration, alpha)

	// Step 5: More conservative significance detection
	significant := !degenerate && isSignificant(lowerCI, upperCI, originalLogRatio, minChangePercent)

	return Report{
		Delta:         originalLogRatio,
		CI:            [2]float64{lowerCI, upperCI},
		Ratio:         math.Exp(originalLogRatio),
		RatioCI:       [2]float64{math.Exp(lowerCI), math.Exp(upperCI)},
		MedianControl: medianControl,
		MedianVariant: medianVariant,
		Confidence:    confidence,
		Significant:   significant,
		Degenerate:    degenerate,
		Samples:       len(bootstrapStats),
	}
}

func bootstrapRNG(controlSamples, experimentSamples, bootstrapSamples int, seed uint64) *rand.Rand {
	seed1 := uint64(controlSamples)<<32 ^ uint64(experimentSamples)<<16 ^ uint64(bootstrapSamples) ^ 0x9e3779b97f4a7c15
	seed2 := uint64(experimentSamples)<<32 ^ uint64(controlSamples)<<16 ^ uint64(bootstrapSamples) ^ 0xbf58476d1ce4e5b9
	seed1 ^= seed
	seed2 ^= seed<<1 | seed>>63
	return rand.New(rand.NewPCG(seed1, seed2))
}

// isSignificant requires the log-ratio interval to clear the practical threshold.
func isSignificant(lowerCI, upperCI, logRatio, minChangePercent float64) bool {
	if !isFinite(lowerCI) || !isFinite(upperCI) || !isFinite(logRatio) {
		return false
	}

	threshold := math.Log1p(math.Max(0, minChangePercent) / 100.0)
	if math.Abs(logRatio) < threshold {
		return false
	}

	return lowerCI > threshold || upperCI < -threshold
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

	clone := append([]float64(nil), data...)
	return medianInPlace(clone)
}

func medianInPlace(data []float64) float64 {
	sort.Float64s(data)
	return medianSorted(data)
}

func medianSorted(data []float64) float64 {
	n := len(data)
	if n%2 == 1 {
		return data[n/2]
	}

	mid1 := data[n/2-1]
	mid2 := data[n/2]
	return (mid1 + mid2) / 2.0
}

// computeBiasCorrection computes the bias-correction parameter.
func computeBiasCorrection(originalStat float64, bootstrapStats []float64) float64 {
	if len(bootstrapStats) == 0 {
		return 0
	}

	less := 0
	equal := 0
	for _, bootStat := range bootstrapStats {
		switch {
		case bootStat < originalStat:
			less++
		case bootStat == originalStat:
			equal++
		}
	}

	// Use mid-ranks so tied medians do not push z0 toward the tails.
	proportion := (float64(less) + 0.5*float64(equal)) / float64(len(bootstrapStats))

	// Avoid edge cases
	if proportion <= 0 {
		proportion = 1.0 / (2.0 * float64(len(bootstrapStats)))
	} else if proportion >= 1 {
		proportion = 1.0 - 1.0/(2.0*float64(len(bootstrapStats)))
	}

	// The bias correction is the inverse normal of the proportion.
	dist := distuv.UnitNormal
	return dist.Quantile(proportion)
}

// computeAcceleration computes the multi-sample BCa acceleration parameter using jackknife.
func computeAcceleration(control, experiment []float64) float64 {
	n1, n2 := len(control), len(experiment)
	if n1 < 2 || n2 < 2 {
		return 0
	}

	controlMedian := median(control)
	experimentMedian := median(experiment)
	if _, ok := logRatio(controlMedian, experimentMedian); !ok {
		return 0
	}

	controlJack := make([]float64, n1)
	for i := 0; i < n1; i++ {
		jackSample := make([]float64, 0, n1-1)
		for j := 0; j < n1; j++ {
			if j != i {
				jackSample = append(jackSample, control[j])
			}
		}
		stat, ok := logRatio(medianInPlace(jackSample), experimentMedian)
		if !ok {
			return 0
		}
		controlJack[i] = stat
	}

	experimentJack := make([]float64, n2)
	for i := 0; i < n2; i++ {
		jackSample := make([]float64, 0, n2-1)
		for j := 0; j < n2; j++ {
			if j != i {
				jackSample = append(jackSample, experiment[j])
			}
		}
		stat, ok := logRatio(controlMedian, medianInPlace(jackSample))
		if !ok {
			return 0
		}
		experimentJack[i] = stat
	}

	sumCubedControl, sumSquaredControl := accelerationTerms(controlJack)
	sumCubedExperiment, sumSquaredExperiment := accelerationTerms(experimentJack)
	sumCubed := sumCubedControl + sumCubedExperiment
	sumSquared := sumSquaredControl + sumSquaredExperiment
	if sumSquared == 0 {
		return 0
	}

	acceleration := sumCubed / (6.0 * math.Pow(sumSquared, 1.5))
	if !isFinite(acceleration) {
		return 0
	}

	return acceleration
}

func accelerationTerms(jackStats []float64) (sumCubed, sumSquared float64) {
	n := float64(len(jackStats))
	jackMean := mean(jackStats)
	for _, stat := range jackStats {
		u := (n - 1) * (jackMean - stat)
		sumCubed += u * u * u
		sumSquared += u * u
	}

	return sumCubed / (n * n * n), sumSquared / (n * n)
}

// computeBCaCI computes the BCa confidence interval
func computeBCaCI(bootstrapStats []float64, biasCorrection, acceleration, alpha float64) (float64, float64) {
	if len(bootstrapStats) == 0 {
		return 0, 0
	}

	// Sort bootstrap statistics
	sortedStats := make([]float64, len(bootstrapStats))
	copy(sortedStats, bootstrapStats)
	sort.Float64s(sortedStats)

	dist := distuv.UnitNormal

	// Compute adjusted percentiles
	lowerAlpha := alpha / 2.0
	upperAlpha := 1.0 - alpha/2.0
	z_alpha2 := dist.Quantile(lowerAlpha)
	z_1minus_alpha2 := dist.Quantile(upperAlpha)

	// BCa adjustments
	alpha1 := adjustedPercentile(biasCorrection, acceleration, z_alpha2, lowerAlpha)
	alpha2 := adjustedPercentile(biasCorrection, acceleration, z_1minus_alpha2, upperAlpha)

	// Ensure valid percentiles
	alpha1 = max(0, min(alpha1, 1))
	alpha2 = max(0, min(alpha2, 1))
	if alpha1 > alpha2 {
		alpha1, alpha2 = alpha2, alpha1
	}

	// Get percentiles from sorted bootstrap statistics
	return percentile(sortedStats, alpha1), percentile(sortedStats, alpha2)
}

func adjustedPercentile(biasCorrection, acceleration, z, fallback float64) float64 {
	denominator := 1.0 - acceleration*(biasCorrection+z)
	if denominator == 0 || !isFinite(denominator) {
		return fallback
	}

	adjusted := biasCorrection + (biasCorrection+z)/denominator
	if !isFinite(adjusted) {
		return fallback
	}

	return distuv.UnitNormal.CDF(adjusted)
}

func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 1 || p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[n-1]
	}

	pos := p * float64(n-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}

	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func degenerateBootstrap(stats []float64) bool {
	if len(stats) == 0 {
		return true
	}

	first := stats[0]
	for _, stat := range stats[1:] {
		if stat != first {
			return false
		}
	}

	return true
}

func normalizeConfidence(confidence float64) float64 {
	if !isFinite(confidence) || confidence <= 0 || confidence >= 1 {
		return defaultConfidence / 100.0
	}

	return confidence
}

func logRatio(control, experiment float64) (float64, bool) {
	if control <= 0 || experiment <= 0 {
		return 0, false
	}

	v := math.Log(experiment / control)
	return v, isFinite(v)
}
