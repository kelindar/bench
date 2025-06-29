package bench

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
)

// codec defines methods for encoding and decoding benchmark results.
type codec interface {
	load(filename string) map[string]Result
	save(filename string, results map[string]Result) error
}

type jsonCodec struct{}

type gobCodec struct{}

func (jsonCodec) load(filename string) map[string]Result {
	data, err := os.ReadFile(filename)
	if err != nil {
		return make(map[string]Result)
	}
	var results map[string]Result
	if err := json.Unmarshal(data, &results); err != nil {
		return make(map[string]Result)
	}
	return results
}

func (jsonCodec) save(filename string, results map[string]Result) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (gobCodec) load(filename string) map[string]Result {
	f, err := os.Open(filename)
	if err != nil {
		return make(map[string]Result)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var results map[string]Result
	if err := dec.Decode(&results); err != nil {
		return make(map[string]Result)
	}
	return results
}

func (gobCodec) save(filename string, results map[string]Result) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return enc.Encode(results)
}

// loadResults loads previous results using the configured codec.
func (r *B) loadResults() map[string]Result {
	if r.codec == nil {
		r.codec = jsonCodec{}
	}
	return r.codec.load(r.filename)
}

// saveResult saves a single result incrementally using the configured codec.
func (r *B) saveResult(result Result) {
	if r.dryRun {
		return
	}
	if r.codec == nil {
		r.codec = jsonCodec{}
	}
	current := r.loadResults()
	current[result.Name] = result
	if err := r.codec.save(r.filename, current); err != nil {
		fmt.Printf("Error writing results file: %v\n", err)
	}
}
