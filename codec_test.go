// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadResult(t *testing.T) {
	for _, extension := range []string{"json", "gob"} {
		t.Run(extension, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "bench."+extension)
			b := &B{config: config{filename: file, codec: codecFor(file)}}
			res := Result{Name: "bench", Samples: []float64{1, 2, 3}, Allocs: []float64{0, 1, 1}, Timestamp: 123,
				Calibration: []float64{10, 11, 12}, CPUUsage: 18.5,
				Environment: Environment{CPUModel: "test", CalibrationVersion: calibrationVersion, Build: "test"}}
			b.saveResult(res)
			assert.Equal(t, res, b.loadResults()["bench"])
		})
	}
}

func TestCodec(t *testing.T) {
	t.Run("json load error", func(t *testing.T) {
		file := "bad.json"
		require.NoError(t, os.WriteFile(file, []byte("bad"), 0644))
		defer os.Remove(file)
		res := jsonCodec{}.load(file)
		assert.Empty(t, res)
	})

	t.Run("gob load error", func(t *testing.T) {
		file := "bad.gob"
		require.NoError(t, os.WriteFile(file, []byte("bad"), 0644))
		defer os.Remove(file)
		res := gobCodec{}.load(file)
		assert.Empty(t, res)
	})

	t.Run("unwritable destination", func(t *testing.T) {
		for _, codec := range []codec{jsonCodec{}, gobCodec{}} {
			assert.Error(t, codec.save(t.TempDir(), map[string]Result{}))
		}
	})

	t.Run("invalid JSON leaves baseline intact", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "baseline.json")
		original := map[string]Result{"bench": {Name: "bench", Samples: []float64{1, 2}}}
		require.NoError(t, (jsonCodec{}).save(file, original))
		b := &B{config: config{filename: file}}
		assert.Equal(t, original, b.loadResults())
		b.saveResult(Result{Name: "bench", Samples: []float64{math.NaN()}})
		assert.Equal(t, original, b.loadResults())
	})

	t.Run("default codec and legacy metadata", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "baseline.json")
		b := &B{config: config{filename: file}}
		legacy := Result{Name: "bench", Samples: []float64{1, 2}}
		b.saveResult(legacy)
		assert.Equal(t, legacy, b.loadResults()["bench"])
		assert.Empty(t, b.loadResults()["bench"].Calibration)
	})
}
