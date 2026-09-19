// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math/bits"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

const (
	calibrationVersion = 1
	calibrationWork    = 1024
	unknownEnvironment = "<unknown>"
)

// Environment identifies the process and machine settings that affect a
// benchmark result.
type Environment struct {
	Hostname           string `json:"hostname"`
	GOOS               string `json:"goos"`
	GOARCH             string `json:"goarch"`
	GoVersion          string `json:"go_version"`
	CPUModel           string `json:"cpu_model"`
	NumCPU             int    `json:"num_cpu"`
	GOMAXPROCS         int    `json:"gomaxprocs"`
	DurationNS         int64  `json:"duration_ns"`
	CalibrationVersion int    `json:"calibration_version"`
	Build              string `json:"build"`
	GOGC               string `json:"gogc"`
	GOMEMLIMIT         string `json:"gomemlimit"`
}

func (e Environment) valid() bool {
	return e.Hostname != "" && e.Hostname != unknownEnvironment &&
		e.GOOS != "" && e.GOARCH != "" && e.GoVersion != "" &&
		e.CPUModel != "" && e.CPUModel != unknownEnvironment &&
		e.NumCPU > 0 && e.GOMAXPROCS > 0 && e.DurationNS > 0 &&
		e.CalibrationVersion == calibrationVersion && e.Build != "" &&
		e.Build != unknownEnvironment && e.GOGC != "" && e.GOGC != unknownEnvironment &&
		e.GOMEMLIMIT != "" && e.GOMEMLIMIT != unknownEnvironment
}

// captureEnvironment records the machine and process settings before collection.
func captureEnvironment(duration time.Duration) Environment {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = unknownEnvironment
	}

	environment := Environment{
		Hostname:           hostname,
		GOOS:               runtime.GOOS,
		GOARCH:             runtime.GOARCH,
		GoVersion:          runtime.Version(),
		CPUModel:           cpuModel(),
		NumCPU:             runtime.NumCPU(),
		GOMAXPROCS:         runtime.GOMAXPROCS(0),
		DurationNS:         duration.Nanoseconds(),
		CalibrationVersion: calibrationVersion,
	}
	environment.Build = buildSettings()
	environment.GOGC, environment.GOMEMLIMIT = gcSettings()
	return environment
}

func cpuModel() string {
	info, err := cpu.Info()
	if err != nil {
		return unknownEnvironment
	}
	for _, item := range info {
		if model := strings.TrimSpace(item.ModelName); model != "" {
			return model
		}
		if model := strings.TrimSpace(item.Model); model != "" {
			return model
		}
	}
	return unknownEnvironment
}

type snapshot struct {
	busy  float64
	total float64
	valid bool
}

// cpuSnapshot reads aggregate CPU counters without using gopsutil's cached
// Percent implementation.
func cpuSnapshot() snapshot {
	times, err := cpu.Times(false)
	if err != nil || len(times) == 0 {
		return snapshot{}
	}

	t := times[0]
	total := t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal + t.Guest + t.GuestNice
	if runtime.GOOS == "linux" {
		// Linux includes guest time in User and GuestNice time in Nice.
		total -= t.Guest + t.GuestNice
	}
	busy := total - t.Idle - t.Iowait
	if !isFinite(total) || !isFinite(busy) || total < 0 || busy < 0 || busy > total {
		return snapshot{}
	}

	return snapshot{busy: busy, total: total, valid: true}
}

// cpuUsage returns aggregate CPU utilization as a percentage, or -1 when the
// two counter readings cannot produce a valid delta.
func cpuUsage(before, after snapshot) float64 {
	if !before.valid || !after.valid {
		return -1
	}
	total := after.total - before.total
	busy := after.busy - before.busy
	if !isFinite(total) || !isFinite(busy) || total <= 0 || busy < 0 || busy > total {
		return -1
	}
	return busy / total * 100
}

func gcSettings() (gogc, gomemlimit string) {
	samples := []metrics.Sample{
		{Name: "/gc/gogc:percent"},
		{Name: "/gc/gomemlimit:bytes"},
	}
	metrics.Read(samples)
	return metricValue(samples[0].Value), metricValue(samples[1].Value)
}

func metricValue(value metrics.Value) string {
	if value.Kind() != metrics.KindUint64 {
		return unknownEnvironment
	}
	return strconv.FormatUint(value.Uint64(), 10)
}

var buildSettingKeys = [...]string{
	"-race",
	"-msan",
	"-asan",
	"-gcflags",
	"-asmflags",
	"-tags",
	"CGO_ENABLED",
	"GOAMD64",
	"GOARM",
	"GOARM64",
}

func buildSettings() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return unknownEnvironment
	}

	var builder strings.Builder
	for _, key := range buildSettingKeys {
		for _, setting := range info.Settings {
			if setting.Key != key {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteByte(';')
			}
			builder.WriteString(key)
			builder.WriteByte('=')
			builder.WriteString(setting.Value)
			break
		}
	}
	if builder.Len() == 0 {
		return "default"
	}
	return builder.String()
}

var calibrationSink atomic.Uint64

//go:noinline
func mixCalibration(seed uint64) uint64 {
	value := seed + 0x9e3779b97f4a7c15
	for i := 0; i < calibrationWork; i++ {
		value ^= value >> 30
		value *= 0xbf58476d1ce4e5b9
		value ^= value << 27
		value *= 0x94d049bb133111eb
		value = bits.RotateLeft64(value, i&63)
	}
	return value ^ (value >> 31)
}

// calibration performs one fixed, allocation-free CPU workload.
func calibration(op int) int {
	calibrationSink.Add(mixCalibration(uint64(op)))
	return 1
}
