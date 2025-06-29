// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"fmt"
	"math"
)

// formatComparison formats statistical comparison between two sample sets using BCa bootstrap
func (r *B) formatComparison(ourSamples, otherSamples []float64) string {
	if len(otherSamples) == 0 {
		return "new"
	}

	ourMean := 0.0
	for _, v := range ourSamples {
		ourMean += v
	}
	ourMean /= float64(len(ourSamples))

	otherMean := 0.0
	for _, v := range otherSamples {
		otherMean += v
	}
	otherMean /= float64(len(otherSamples))

	// Handle edge cases more robustly
	if otherMean == 0 || ourMean == 0 {
		return "🟰 similar" // Conservative: avoid infinite or invalid ratios
	}

	// Check for unreasonable performance differences (likely measurement error)
	ratio := ourMean / otherMean
	if ratio > 1000 || ratio < 0.001 {
		return "🟰 similar" // Conservative: avoid reporting massive differences
	}

	// Perform BCa bootstrap with 10,000 samples
	bootstrapResult := bca(otherSamples, ourSamples, r.confidence/100.0, 10000)

	speedup := otherMean / ourMean
	change := (speedup - 1) * 100

	// Convert delta confidence interval to percentage bounds correctly
	// If delta CI is [lowerCI, upperCI] in absolute units,
	// convert to percentage changes relative to baseline
	var interval [2]float64
	if otherMean != 0 {
		// Calculate percentage change for each CI bound
		// Lower bound: what % change if the true difference is lowerCI
		// Upper bound: what % change if the true difference is upperCI
		if (otherMean - bootstrapResult.LowerCI) != 0 {
			interval[0] = (otherMean/(otherMean-bootstrapResult.LowerCI) - 1) * 100
		}
		if (otherMean - bootstrapResult.UpperCI) != 0 {
			interval[1] = (otherMean/(otherMean-bootstrapResult.UpperCI) - 1) * 100
		}

		// Ensure interval is ordered correctly (lower <= upper)
		if interval[0] > interval[1] {
			interval[0], interval[1] = interval[1], interval[0]
		}
	}

	switch {
	case !bootstrapResult.Significant:
		return "🟰 similar"
	case speedup > 1:
		return fmt.Sprintf("✅ %s", formatChange(change, interval))
	default:
		return fmt.Sprintf("❌ %s", formatChange(change, interval))
	}
}

// formatChange formats the change in performance
func formatChange(changePercent float64, interval [2]float64) string {
	var sign string
	if changePercent > 0 {
		sign = "+"
	}

	switch {
	case changePercent >= 1000:
		return fmt.Sprintf("+%.0fx", changePercent)
	case changePercent > 100:
		return fmt.Sprintf("+%.1fx", changePercent)
	default:
		return fmt.Sprintf("%s%.0f%% %s", sign, changePercent, formatCI(interval))
	}
}

func formatCI(interval [2]float64) string {
	switch {
	case math.IsNaN(interval[0]) || math.IsNaN(interval[1]) ||
		math.IsInf(interval[0], 0) || math.IsInf(interval[1], 0):
		return ""
	case math.Abs(interval[1]-interval[0]) > 100:
		return ""
	case math.Abs(interval[0]) > 1000 || math.Abs(interval[1]) > 1000:
		return ""
	default:
		return fmt.Sprintf("[%.0f%%,%.0f%%]", interval[0], interval[1])
	}
}

// formatTime formats nanoseconds per operation
func formatTime(nsPerOp float64) string {
	switch {
	case nsPerOp >= 1000000:
		return fmt.Sprintf("%.1f ms", nsPerOp/1000000)
	case nsPerOp >= 1000:
		return fmt.Sprintf("%.1f µs", nsPerOp/1000)
	default:
		return fmt.Sprintf("%.1f ns", nsPerOp)
	}
}

// formatOps formats operations per second
func formatOps(opsPerSec float64) string {
	if opsPerSec >= 1000000 {
		return fmt.Sprintf("%.1fM", opsPerSec/1000000)
	}
	if opsPerSec >= 1000 {
		return fmt.Sprintf("%.1fK", opsPerSec/1000)
	}
	return fmt.Sprintf("%.0f", opsPerSec)
}

// formatAllocs formats number of allocations per operation
func formatAllocs(allocsPerOp float64) string {
	switch {
	case allocsPerOp >= 1000:
		return fmt.Sprintf("%.1fK", allocsPerOp/1000)
	case allocsPerOp >= 1:
		return fmt.Sprintf("%.0f", allocsPerOp)
	default:
		return "0"
	}
}
