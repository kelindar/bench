// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"fmt"
	"math"
)

type allocChange int

const (
	allocUnknown allocChange = iota
	allocSame
	allocBetter
	allocWorse
)

// formatComparison formats statistical comparison between two sample sets using BCa bootstrap
func (r *B) formatComparison(report Report) string {
	ratio := report.Ratio
	if ratio == 0 && report.MedianControl > 0 && report.MedianVariant > 0 {
		ratio = report.MedianVariant / report.MedianControl
	}

	switch {
	case report.MedianControl <= 0 || report.MedianVariant <= 0 || ratio <= 0:
		return "🟰 similar" // A infinite or invalid ratios
	case report.Significant && ratio > 1000:
		return "❌ uncomparable"
	case report.Significant && ratio < 0.001:
		return "✅ uncomparable"
	}

	change := ratioToChange(ratio)

	switch {
	case !report.Significant:
		return "🟰 similar"
	case ratio < 1:
		return fmt.Sprintf("✅ %s", formatChange(change))
	default:
		return fmt.Sprintf("❌ %s", formatChange(change))
	}
}

func ratioToChange(ratio float64) float64 {
	return (1/ratio - 1) * 100
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

func formatAllocsWithChange(allocsPerOp float64, change allocChange) string {
	value := formatAllocs(allocsPerOp)
	switch change {
	case allocBetter:
		return "✅ " + value
	case allocWorse:
		return "❌ " + value
	case allocSame:
		return "🟰 " + value
	default:
		return value
	}
}

func allocIntValue(allocsPerOp float64) int64 {
	if allocsPerOp < 1 {
		return 0
	}
	return int64(math.Round(allocsPerOp))
}

func compareAllocs(previous, current []float64) allocChange {
	if len(previous) == 0 || len(current) == 0 {
		return allocUnknown
	}

	prev := allocIntValue(median(previous))
	curr := allocIntValue(median(current))
	switch {
	case curr == prev:
		return allocSame
	case curr < prev:
		return allocBetter
	default:
		return allocWorse
	}
}
