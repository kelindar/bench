package bench

import (
	"os"
	"testing"
)

func TestSaveLoadResult(t *testing.T) {
	file := "test_codec.json"
	defer os.Remove(file)
	b := &B{config: config{filename: file, codec: jsonCodec{}}}
	res := Result{Name: "bench", Samples: []float64{1, 2, 3}, Timestamp: 123}
	b.saveResult(res)
	loaded := b.loadResults()
	if loaded["bench"].Timestamp != 123 {
		t.Fatalf("expected timestamp 123")
	}
}

func TestGobCodec(t *testing.T) {
	file := "test_codec.gob"
	defer os.Remove(file)
	b := &B{config: config{filename: file, codec: gobCodec{}}}
	res := Result{Name: "bench", Samples: []float64{1, 2, 3}, Timestamp: 321}
	b.saveResult(res)
	loaded := b.loadResults()
	if loaded["bench"].Timestamp != 321 {
		t.Fatalf("expected timestamp 321")
	}
}

func TestJSONCodecLoadError(t *testing.T) {
	file := "bad.json"
	os.WriteFile(file, []byte("bad"), 0644)
	defer os.Remove(file)
	res := jsonCodec{}.load(file)
	if len(res) != 0 {
		t.Fatalf("expected empty result")
	}
}

func TestGobCodecLoadError(t *testing.T) {
	file := "bad.gob"
	os.WriteFile(file, []byte("bad"), 0644)
	defer os.Remove(file)
	res := gobCodec{}.load(file)
	if len(res) != 0 {
		t.Fatalf("expected empty result")
	}
}
