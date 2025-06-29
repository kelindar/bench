package bench

import (
	"fmt"

	"github.com/codahale/tinystat"
)

// formatAllocs formats number of allocations per operation
func (r *B) formatAllocs(allocsPerOp float64) string {
	switch {
	case allocsPerOp >= 1000:
		return fmt.Sprintf("%.1fK", allocsPerOp/1000)
	case allocsPerOp >= 1:
		return fmt.Sprintf("%.0f", allocsPerOp)
	default:
		return "0"
	}
}

// formatComparison formats statistical comparison between two sample sets
func (r *B) formatComparison(ourSamples, otherSamples []float64) string {
	if len(otherSamples) == 0 {
		return "new"
	}

	our := tinystat.Summarize(ourSamples)
	other := tinystat.Summarize(otherSamples)
	if other.Mean == 0 {
		if our.Mean > 0 {
			return "✅ +inf%"
		}
		return "~ similar"
	}

	speedup := our.Mean / other.Mean
	changePercent := (speedup - 1) * 100
	diff := tinystat.Compare(our, other, 99.9)

	// For non-significant changes close to zero, show "similar"
	if !diff.Significant() && changePercent >= -2 && changePercent <= 2 {
		return "~ similar"
	}

	var sign string
	if changePercent > 0 {
		sign = "+"
	} else {
		sign = ""
	}

	if !diff.Significant() {
		return fmt.Sprintf("~ %s%.0f%% (p=%.3f)", sign, changePercent, diff.PValue)
	}

	if speedup > 1 {
		return fmt.Sprintf("✅ %s%.0f%% (p=%.3f)", sign, changePercent, diff.PValue)
	}

	return fmt.Sprintf("❌ %s%.0f%% (p=%.3f)", sign, changePercent, diff.PValue)
}

// formatTime formats nanoseconds per operation
func (r *B) formatTime(nsPerOp float64) string {
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
func (r *B) formatOps(opsPerSec float64) string {
	if opsPerSec >= 1000000 {
		return fmt.Sprintf("%.1fM", opsPerSec/1000000)
	}
	if opsPerSec >= 1000 {
		return fmt.Sprintf("%.1fK", opsPerSec/1000)
	}
	return fmt.Sprintf("%.0f", opsPerSec)
}
