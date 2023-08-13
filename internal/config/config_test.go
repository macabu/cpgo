package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/macabu/cpgo/internal/config"
)

const validConfig = `---
backends:
- url: http://localhost:6060/debug/pprof/profile?seconds=30
  schedule: '* * * * *'
  open_pull_request:
    repository: http://github.com/example/example
    target_file: default.pgo
    target_branch: main
`

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("happy path with a valid file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		path := "valid-file"

		file, err := os.CreateTemp(tempDir, path)
		require.NoError(t, err)

		n, err := file.Write([]byte(validConfig))
		require.NoError(t, err)
		require.NotZero(t, n)

		cfg, err := config.Parse(file.Name())
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("when the file does not exist, return an error", func(t *testing.T) {
		t.Parallel()

		cfg, err := config.Parse(t.Name())
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("when the config file is not a valid yaml, return an error", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		path := "invalid-file"

		file, err := os.CreateTemp(tempDir, path)
		require.NoError(t, err)

		n, err := file.Write([]byte("0"))
		require.NoError(t, err)
		require.NotZero(t, n)

		cfg, err := config.Parse(file.Name())
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
