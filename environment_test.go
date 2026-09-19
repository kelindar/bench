// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"runtime"
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
}
