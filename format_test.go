// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatHelpers(t *testing.T) {
	assert.Equal(t, "1.5K", formatAllocs(1500))
	assert.Equal(t, "10", formatAllocs(10))
	assert.Equal(t, "0", formatAllocs(0.5))

	assert.Contains(t, formatTime(2e6), "ms")
	assert.Contains(t, formatTime(2e3), "µs")
	assert.Contains(t, formatTime(2), "ns")

	assert.Contains(t, formatOps(2e6), "M")
	assert.Contains(t, formatOps(2e3), "K")
	assert.Equal(t, "2", formatOps(2))
}
