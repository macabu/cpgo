package flags_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/macabu/cpgo/internal/flags"
)

func TestParse(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name          string
		args          []string
		expectedErr   error
		expectedFlags flags.Flags
	}{
		{
			name:        "when no GitHub token is passed, an error is returned",
			args:        nil,
			expectedErr: flags.ErrGitHubTokenNotFound,
		},
		{
			name: "when valid options are passed, no error is returned",
			args: []string{"-verbose", "-githubToken", "my-token", "-configPath", "/path/to/config.sample.yaml"},
			expectedFlags: flags.Flags{
				GithubToken: "my-token",
				ConfigPath:  "/path/to/config.sample.yaml",
				LogVerbose:  true,
			},
		},
	}

	for _, tt := range testcases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actualFlags, _, err := flags.Parse(tt.name, tt.args)
			require.ErrorIs(t, err, tt.expectedErr)
			require.EqualValues(t, tt.expectedFlags, actualFlags)
		})
	}
}
