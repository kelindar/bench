// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnvironment(t *testing.T) {
	duration := 7 * time.Millisecond
	environment := captureEnvironment(duration)

	assert.NotEmpty(t, environment.Hostname)
	assert.Equal(t, runtime.GOOS, environment.GOOS)
	assert.Equal(t, runtime.GOARCH, environment.GOARCH)
	assert.NotEmpty(t, environment.GoVersion)
	assert.NotEmpty(t, environment.CPUModel)
	assert.GreaterOrEqual(t, environment.NumCPU, 1)
	assert.GreaterOrEqual(t, environment.GOMAXPROCS, 1)
	assert.Equal(t, duration.Nanoseconds(), environment.DurationNS)
	assert.Equal(t, calibrationVersion, environment.CalibrationVersion)
	assert.NotEmpty(t, environment.Build)
	gogc, gomemlimit := gcSettings()
	assert.Equal(t, gogc, environment.GOGC)
	assert.Equal(t, gomemlimit, environment.GOMEMLIMIT)
	assert.True(t, environment.valid())

	// Keep the type comparable so baseline compatibility can use ==.
	assert.True(t, environment == environment)
}

func TestCPUSnapshot(t *testing.T) {
	snapshot := cpuSnapshot()
	if !assert.True(t, snapshot.valid) {
		return
	}
	assert.GreaterOrEqual(t, snapshot.total, 0.0)
	assert.GreaterOrEqual(t, snapshot.busy, 0.0)
	assert.LessOrEqual(t, snapshot.busy, snapshot.total)
}

func TestCPUUsage(t *testing.T) {
	tests := []struct {
		name   string
		before snapshot
		after  snapshot
		want   float64
	}{
		{
			name:   "valid delta",
			before: snapshot{busy: 20, total: 100, valid: true},
			after:  snapshot{busy: 50, total: 200, valid: true},
			want:   30,
		},
		{
			name:   "invalid before",
			before: snapshot{busy: 20, total: 100},
			after:  snapshot{busy: 50, total: 200, valid: true},
			want:   -1,
		},
		{
			name:   "no elapsed counters",
			before: snapshot{busy: 20, total: 100, valid: true},
			after:  snapshot{busy: 20, total: 100, valid: true},
			want:   -1,
		},
		{
			name:   "counter moved backwards",
			before: snapshot{busy: 50, total: 200, valid: true},
			after:  snapshot{busy: 40, total: 300, valid: true},
			want:   -1,
		},
		{
			name:   "busy exceeds total",
			before: snapshot{busy: 20, total: 100, valid: true},
			after:  snapshot{busy: 130, total: 200, valid: true},
			want:   -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, cpuUsage(test.before, test.after))
		})
	}
}

func TestCalibration(t *testing.T) {
	before := calibrationSink.Load()
	assert.Equal(t, 1, calibration(17))
	after := calibrationSink.Load()
	assert.NotEqual(t, before, after)
	assert.Zero(t, testing.AllocsPerRun(100, func() { calibration(17) }), "calibration must not add allocation noise")
}

func TestMetadata(t *testing.T) {
	valid := captureEnvironment(time.Millisecond)
	for _, test := range []struct {
		name   string
		change func(*Environment)
	}{
		{"hostname", func(e *Environment) { e.Hostname = "" }},
		{"unknown CPU", func(e *Environment) { e.CPUModel = unknownEnvironment }},
		{"CPU count", func(e *Environment) { e.NumCPU = 0 }},
		{"parallelism", func(e *Environment) { e.GOMAXPROCS = 0 }},
		{"duration", func(e *Environment) { e.DurationNS = 0 }},
		{"version", func(e *Environment) { e.CalibrationVersion++ }},
		{"build", func(e *Environment) { e.Build = unknownEnvironment }},
		{"GC", func(e *Environment) { e.GOGC = unknownEnvironment }},
		{"memory limit", func(e *Environment) { e.GOMEMLIMIT = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := valid
			test.change(&changed)
			assert.False(t, changed.valid())
		})
	}
	assert.Equal(t, unknownEnvironment, metricValue(metrics.Value{}))

	t.Run("runtime GC settings", func(t *testing.T) {
		previousGC := debug.SetGCPercent(73)
		previousLimit := debug.SetMemoryLimit(1 << 30)
		t.Cleanup(func() {
			debug.SetGCPercent(previousGC)
			debug.SetMemoryLimit(previousLimit)
		})
		gogc, limit := gcSettings()
		assert.Equal(t, "73", gogc)
		assert.Equal(t, "1073741824", limit)
	})

	t.Run("invalid CPU counters", func(t *testing.T) {
		before := snapshot{busy: 10, total: 100, valid: true}
		assert.Equal(t, -1.0, cpuUsage(before, snapshot{}))
		assert.Equal(t, -1.0, cpuUsage(before, snapshot{busy: math.NaN(), total: 200, valid: true}))
		assert.Equal(t, -1.0, cpuUsage(before, snapshot{busy: 20, total: math.Inf(1), valid: true}))
		assert.Zero(t, cpuUsage(before, snapshot{busy: 10, total: 200, valid: true}))
		assert.Equal(t, 100.0, cpuUsage(before, snapshot{busy: 110, total: 200, valid: true}))
	})
}
