// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"fmt"
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

	// Perform BCa bootstrap with 10,000 samples
	bootstrapResult := BCaBootstrap(otherSamples, ourSamples, r.confidence/100.0, 10000)

	speedup := otherMean / ourMean
	change := (speedup - 1) * 100

	// Convert delta confidence interval to percentage bounds
	// If delta CI is [lowerCI, upperCI] in absolute units (ns),
	// then percentage CI is approximately:
	// [lowerCI/otherMean * 100, upperCI/otherMean * 100]
	var interval [2]float64
	if otherMean != 0 {
		interval[0] = bootstrapResult.LowerCI / otherMean * 100
		interval[1] = bootstrapResult.UpperCI / otherMean * 100
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
		return fmt.Sprintf("%.0fx", changePercent)
	case changePercent > 100:
		return fmt.Sprintf("%.1fx", changePercent)
	default:
		return fmt.Sprintf("%s%.0f%% %s", sign, changePercent, formatCI(interval))
	}
}

func formatCI(interval [2]float64) string {
	return fmt.Sprintf("[%.0f%%,%.0f%%]", interval[0], interval[1])
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
