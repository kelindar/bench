package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainExample(t *testing.T) {
	t.Chdir(t.TempDir())
	oldArgs, oldStdout := os.Args, os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Args, os.Stdout = oldArgs, oldStdout
		reader.Close()
		writer.Close()
	})
	os.Args, os.Stdout = []string{"example", "-n"}, writer
	main()
	require.NoError(t, writer.Close())
	os.Stdout = oldStdout
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Contains(t, string(output), "find")
	assert.Contains(t, string(output), "sort")
	assert.Contains(t, string(output), "vs ref")
	_, err = os.Stat("bench.gob")
	assert.True(t, os.IsNotExist(err), "dry-run example must not create a baseline")
}
