package main

import (
	"sort"
	"strings"

	"github.com/kelindar/bench"
)

func main() {
	testdata := []string{
		"fig", "grape", "honeydew", "kiwi", "lemon",
		"mango", "nectarine", "orange", "pear", "pineapple",
		"apple", "banana", "cherry", "date", "elderberry",
	}

	bench.Run(func(b *bench.B) {

		b.Run("contains", func(i int) {
			for _, v := range testdata {
				_ = strings.Contains(v, "orange")
			}
		})

		b.Run("sort", func(i int) {
			clone := make([]string, 0, len(testdata))
			copy(clone, testdata)
			sort.Strings(clone)
		})

	}, bench.WithSamples(50), bench.WithReference())

}
