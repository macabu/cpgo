package gh_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/macabu/cpgo/internal/gitops/gh"
)

func TestParseRepoURL(t *testing.T) {
	testcases := []struct {
		inputURL     string
		expectedRepo gh.Repository
	}{
		{"https://github.com/my-org/my-repo", gh.Repository{Org: "my-org", Name: "my-repo"}},
		{"https://github.com/my-org/my-repo/", gh.Repository{Org: "my-org", Name: "my-repo"}},
	}

	for _, tt := range testcases {
		actualRepo := gh.ParseRepoURL(tt.inputURL)
		require.EqualValues(t, tt.expectedRepo, actualRepo)
	}
}
