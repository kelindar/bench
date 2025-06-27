// Copyright 2021 The tinystat Authors
// Copyright 2025 Roman Atachiants
// This is a fork of https://github.com/codahale/tinystat
package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeOdd(t *testing.T) {
	t.Parallel()

	s := Summarize([]float64{1, 2, 3})

	assert.Equal(t, float64(3), s.N, "N")
	assert.InDelta(t, 2, s.Mean, 0.001, "Mean")
	assert.InDelta(t, 1, s.Variance, 0.001, "Variance")
	assert.InDelta(t, 1.0, s.StdDev(), 0.001, "StdDev")
	assert.InDelta(t, 0.5773502691896258, s.StdErr(), 0.001, "StdErr")
}

func TestSummarizeEven(t *testing.T) {
	t.Parallel()

	s := Summarize([]float64{1, 2, 3, 4})

	assert.Equal(t, float64(4), s.N, "N")
	assert.InDelta(t, 2.5, s.Mean, 0.001, "Mean")
	assert.InDelta(t, 1.6666666666666667, s.Variance, 0.001, "Variance")
	assert.InDelta(t, 1.2909944487358056, s.StdDev(), 0.001, "StdDev")
	assert.InDelta(t, 0.6454972243679028, s.StdErr(), 0.001, "StdErr")
}

func TestCompareSimilarData(t *testing.T) {
	t.Parallel()

	a := Summarize([]float64{1, 2, 3, 4})
	b := Summarize([]float64{1, 2, 3, 4})
	d := Compare(a, b, 80)

	assert.InDelta(t, 0, d.Effect, 0.001, "Effect")
	assert.InDelta(t, 0, d.EffectSize, 0.001, "EffectSize")
	assert.InDelta(t, 1.314, d.CriticalValue, 0.001, "CriticalValue")
	assert.InDelta(t, 1, d.PValue, 0.001, "PValue")
	assert.InDelta(t, 0.2, d.Alpha, 0.001, "Alpha")
	assert.InDelta(t, 0, d.Beta, 0.001, "Beta")
	assert.False(t, d.Significant(), "Significant")
}

func TestCompareDifferentData(t *testing.T) {
	t.Parallel()

	a := Summarize([]float64{1, 2, 3, 4})
	b := Summarize([]float64{10, 20, 30, 40})
	d := Compare(a, b, 80)

	assert.InDelta(t, 22.5, d.Effect, 0.001, "Effect")
	assert.InDelta(t, 2.452519415855564, d.EffectSize, 0.001, "EffectSize")
	assert.InDelta(t, 10.568, d.CriticalValue, 0.001, "CriticalValue")
	assert.InDelta(t, 0.03916791618893338, d.PValue, 0.001, "PValue")
	assert.InDelta(t, 0.2, d.Alpha, 0.001, "Alpha")
	assert.InDelta(t, 0.9856216842773273, d.Beta, 0.001, "Beta")
	assert.True(t, d.Significant(), "Significant")
}
