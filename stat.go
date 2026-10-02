// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"math/rand/v2"
	"sort"

	"gonum.org/v1/gonum/stat/combin"
	"gonum.org/v1/gonum/stat/distuv"
)

// Report compares median timings. Intervals assume independent observations
// or, for clustered samples, approximately independent consecutive blocks.
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
	Inconclusive  string     // Inconclusive explains why no improvement/regression can be established
}

// bca performs BCa (Bias-Corrected accelerated) bootstrap inference comparing
// two samples. The test statistic is the log median time ratio.
func bca(control, experiment []float64, confidence float64, bootstrapSamples int, minChangePercent float64) Report {
	return bcaWithSeed(control, experiment, confidence, bootstrapSamples, minChangePercent, 0)
}

func bcaWithSeed(control, experiment []float64, confidence float64, bootstrapSamples int, minChangePercent float64, seed uint64) Report {
	switch {
	case len(control) == 0 || len(experiment) == 0:
		return Report{Inconclusive: "invalid"}
	case bootstrapSamples <= 0:
		return Report{Inconclusive: "invalid"}
	}
	confidence = normalizeConfidence(confidence)
	if !validSamples(control) || !validSamples(experiment) {
		return Report{Confidence: confidence, Inconclusive: "invalid"}
	}

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
	controlBlock, variantBlock := blockSize(control), blockSize(experiment)
	rng := bootstrapRNG(len(control), len(experiment), bootstrapSamples, seed)

	bootstrapStats := make([]float64, 0, bootstrapSamples)
	controlBootstrap := make([]float64, len(control))
	variantBootstrap := make([]float64, len(experiment))
	for i := 0; i < bootstrapSamples; i++ {

		// Resample with replacement using our seeded RNG
		resampleBlocks(controlBootstrap, control, controlBlock, rng)
		resampleBlocks(variantBootstrap, experiment, variantBlock, rng)

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

	// The single-observation jackknife assumes independence. Clustered samples
	// use bias correction without acceleration and wider block bounds below.
	acceleration := 0.0
	if controlBlock == 1 && variantBlock == 1 {
		acceleration = computeAcceleration(control, experiment)
	}
	degenerate := degenerateBootstrap(bootstrapStats)

	// Step 4: Compute BCa confidence interval
	alpha := 1.0 - confidence
	lowerCI, upperCI := computeBCaCI(bootstrapStats, biasCorrection, acceleration, alpha)

	// BCa is approximate, especially for small or tied samples. Never report a
	// narrower interval than the order-statistic bounds for two medians.
	// Bonferroni allocates alpha/4 to each of the four tails.
	controlLow, controlHigh := blockInterval(control, alpha/4, controlBlock)
	variantLow, variantHigh := blockInterval(experiment, alpha/4, variantBlock)
	lowerCI = math.Min(lowerCI, math.Log(variantLow)-math.Log(controlHigh))
	upperCI = math.Max(upperCI, math.Log(variantHigh)-math.Log(controlLow))
	significant := !degenerate && isSignificant(lowerCI, upperCI, originalLogRatio, minChangePercent)

	inconclusive := ""
	threshold := math.Log1p(math.Max(0, minChangePercent) / 100)
	switch {
	case !isFinite(lowerCI) || !isFinite(upperCI):
		inconclusive = "uncertain"
	case !significant && (lowerCI < -threshold || upperCI > threshold):
		inconclusive = "uncertain"
	}

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
		Inconclusive:  inconclusive,
	}
}

// clustered screens for positive serial dependence with the exact lower-tail
// runs test at 1%. Median ties are omitted. Passing is not proof of independence.
func clustered(data []float64, center float64) bool {
	below, above, runs, last := 0, 0, 0, 0
	for _, v := range data {
		sign := 0
		switch {
		case v < center:
			below++
			sign = -1
		case v > center:
			above++
			sign = 1
		default:
			continue
		}
		if sign != last {
			runs++
		}
		last = sign
	}
	if below == 0 || above == 0 {
		return false
	}

	denominator := logChoose(below+above, below)
	p := 0.0
	for r := 2; r <= runs; r++ {
		k := r / 2
		if r%2 == 0 {
			p += 2 * math.Exp(logChoose(below-1, k-1)+logChoose(above-1, k-1)-denominator)
		} else {
			p += math.Exp(logChoose(below-1, k) + logChoose(above-1, k-1) - denominator)
			p += math.Exp(logChoose(below-1, k-1) + logChoose(above-1, k) - denominator)
		}
		if p >= 0.01 {
			return false
		}
	}
	return p < 0.01
}

func logChoose(n, k int) float64 {
	if k < 0 || k > n {
		return math.Inf(-1)
	}
	return combin.LogGeneralizedBinomial(float64(n), float64(k))
}

func blockSize(data []float64) int {
	if !clustered(data, median(data)) {
		return 1
	}

	// ponytail: sqrt(n) blocks cover short dependence; use a measured correlation
	// horizon if workloads exhibit longer dependence. Arbitrary drift is not IID.
	return int(math.Ceil(math.Sqrt(float64(len(data)))))
}

// blockInterval bounds the raw population median using block minima and maxima.
// Each minimum is below the median with probability at least 1/2; each maximum
// is above it with probability at least 1/2. Binomial ranks are conservative when
// blocks are independent, without assuming independence inside each block.
func blockInterval(data []float64, tail float64, block int) (float64, float64) {
	if block == 1 {
		return medianInterval(data, tail)
	}

	n := len(data) / block
	if n == 0 {
		return 0, math.Inf(1)
	}
	lows, highs := make([]float64, n), make([]float64, n)
	for i := range n {
		end := (i + 1) * block
		if i == n-1 {
			end = len(data)
		}
		lows[i], highs[i] = data[i*block], data[i*block]
		for _, value := range data[i*block : end] {
			lows[i] = min(lows[i], value)
			highs[i] = max(highs[i], value)
		}
	}
	lower, _ := medianInterval(lows, tail)
	_, upper := medianInterval(highs, tail)
	return lower, upper
}

func validSamples(data []float64) bool {
	if len(data) == 0 {
		return false
	}
	for _, v := range data {
		if v <= 0 || !isFinite(v) {
			return false
		}
	}
	return true
}

// medianInterval uses binomial ranks, without interpolating order statistics.
// If even the sample extremes cannot attain the confidence, return open bounds.
func medianInterval(data []float64, tail float64) (float64, float64) {
	n := len(data)
	dist := distuv.Binomial{N: float64(n), P: 0.5}
	k := sort.Search(n/2, func(k int) bool { return dist.CDF(float64(k)) > tail })
	if k == 0 {
		return 0, math.Inf(1)
	}
	sorted := append([]float64(nil), data...)
	sort.Float64s(sorted)
	return sorted[k-1], sorted[n-k]
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

// resampleBlocks draws consecutive circular blocks with replacement. A block
// size of one is the ordinary independent bootstrap.
func resampleBlocks(resampled, data []float64, block int, rng *rand.Rand) {
	n := len(data)
	for i := 0; i < len(resampled); i += block {
		idx := rng.IntN(n)
		for j := 0; j < min(block, len(resampled)-i); j++ {
			resampled[i+j] = data[idx]
			idx++
			if idx == n {
				idx = 0
			}
		}
	}
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
	return mid1 + (mid2-mid1)/2
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
	switch {
	case n == 1 || p <= 0:
		return sorted[0]
	case p >= 1:
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

	v := math.Log(experiment) - math.Log(control)
	return v, isFinite(v)
}
