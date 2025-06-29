package main

import (
	"strings"

	"github.com/kelindar/bench"
)

func main() {
	bench.Run(func(b *bench.B) {
		testString := strings.Repeat("hello world testing ", 1000)

		b.Run("contains", func(i int) {
			_ = strings.Contains(testString, "testing")
		})

		b.Run("contains_ref", func(i int) {
			_ = strings.Contains(testString, "testing")
		}, func(i int) {
			_ = strings.Count(testString, "testing") == 1

		})

	}, bench.WithSamples(50), bench.WithReference())

}
