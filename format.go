// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"fmt"
)

// formatComparison formats statistical comparison between two sample sets using BCa bootstrap
func (r *B) formatComparison(report Report) string {
	switch {
	case report.MeanControl == 0 || report.MeanVariant == 0:
		return "🟰 similar" // A infinite or invalid ratios
	case report.Significant && report.MeanVariant > 1000*report.MeanControl:
		return "❌ uncomparable"
	case report.Significant && report.MeanVariant < 0.001*report.MeanControl:
		return "✅ uncomparable"
	}

	speedup := report.MeanControl / report.MeanVariant
	change := (speedup - 1) * 100

	// Convert delta confidence interval to percentage bounds correctly
	var interval [2]float64
	if report.MeanControl != 0 {
		if (report.MeanControl - report.CI[0]) != 0 {
			interval[0] = (report.MeanControl/(report.MeanControl-report.CI[0]) - 1) * 100
		}
		if (report.MeanControl - report.CI[1]) != 0 {
			interval[1] = (report.MeanControl/(report.MeanControl-report.CI[1]) - 1) * 100
		}

		// Ensure interval is ordered correctly (lower <= upper)
		if interval[0] > interval[1] {
			interval[0], interval[1] = interval[1], interval[0]
		}
	}

	switch {
	case !report.Significant:
		return "🟰 similar"
	case speedup > 1:
		return fmt.Sprintf("✅ %s", formatChange(change))
	default:
		return fmt.Sprintf("❌ %s", formatChange(change))
	}
}

// formatChange formats the change in performance
func formatChange(changePercent float64) string {
	var sign string
	if changePercent > 0 {
		sign = "+"
	}

	switch {
	case changePercent >= 1000:
		return fmt.Sprintf("%+.0fx", 1+changePercent/100)
	case changePercent > 100:
		return fmt.Sprintf("%+.1fx", 1+changePercent/100)
	default:
		return fmt.Sprintf("%s%.0f%%", sign, changePercent)
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
