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
	ref := asSet(testdata)

	// Run the benchmarks
	bench.Run(func(b *bench.B) {

		// Run a benchmark that checks if the string "orange" is in the testdata
		// and compares it to a reference implementation that uses a pre-allocated map.
		b.Run("find", func(i int) {
			for _, v := range testdata {
				_ = strings.Contains(v, "orange")
			}
		}, func(i int) {
			_ = ref["orange"]
		})

		// Run a benchmark that sorts the testdata
		b.Run("sort", func(i int) {
			clone := make([]string, 0, len(testdata))
			copy(clone, testdata)
			sort.Strings(clone)
		})

	}, bench.WithSamples(50), bench.WithReference())

}

func asSet(data []string) map[string]struct{} {
	set := make(map[string]struct{}, len(data))
	for _, v := range data {
		set[v] = struct{}{}
	}
	return set
}
