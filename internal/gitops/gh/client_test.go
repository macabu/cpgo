package gh_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/require"

	"github.com/macabu/cpgo/internal/gitops"
	"github.com/macabu/cpgo/internal/gitops/gh"
)

func TestExistingPGOFileURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	opts := gh.Options{
		Repo: gh.Repository{
			Org:  "my-org",
			Name: "my-repo",
		},
		Filename:   "default.pgo",
		MainBranch: "main",
	}

	t.Run("given an existing PGO file, it returns its download URL", func(t *testing.T) {
		t.Parallel()

		mockDownloadURL := "https://github.com/my-org/my-repo/blob/main/default.pgo"

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write(mock.MustMarshal(github.RepositoryContent{
						DownloadURL: &mockDownloadURL,
					}))
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		downloadURL, err := client.ExistingPGOFileURL(ctx, opts)
		require.NoError(t, err)

		validURL, err := url.Parse(downloadURL)
		require.NoError(t, err)
		require.NotNil(t, validURL)
	})

	t.Run("when there is no access or not found for the repo or the file, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusNotFound, "repo or file not found or no access")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		downloadURL, err := client.ExistingPGOFileURL(ctx, opts)
		require.ErrorIs(t, err, gitops.ErrPGOFileNotFound)
		require.Empty(t, downloadURL)
	})

	t.Run("when the provider responds with an error, it is propagated", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposContentsByOwnerByRepoByPath,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "this is fine")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		downloadURL, err := client.ExistingPGOFileURL(ctx, opts)
		require.Error(t, err)
		require.Empty(t, downloadURL)
	})
}

func TestUpdatePGOFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	opts := gh.Options{
		Repo: gh.Repository{
			Org:  "my-org",
			Name: "my-repo",
		},
		Filename:   "default.pgo",
		MainBranch: "main",
	}

	mockValidCreateBlob := mock.WithRequestMatchHandler(
		mock.PostReposGitBlobsByOwnerByRepo,
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			blobSHA := "definitely-a-valid-blob-sha"

			_, _ = w.Write(mock.MustMarshal(github.Blob{
				SHA: &blobSHA,
			}))
		}),
	)

	mockValidListMatchingRefs := mock.WithRequestMatchHandler(
		mock.EndpointPattern{
			Pattern: "/repos/{owner}/{repo}/git/matching-refs/",
			Method:  "GET",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mainRef := "refs/heads/" + opts.MainBranch
			mainRefCurrentSHA := "another-very-real-sha"

			_, _ = w.Write(mock.MustMarshal([]*github.Reference{
				{
					Ref: &mainRef,
					Object: &github.GitObject{
						SHA: &mainRefCurrentSHA,
					},
				},
			}))
		}),
	)

	mockValidCreateTree := mock.WithRequestMatchHandler(
		mock.PostReposGitTreesByOwnerByRepo,
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(mock.MustMarshal(github.Tree{}))
		}),
	)

	mockValidCreateCommit := mock.WithRequestMatchHandler(
		mock.PostReposGitCommitsByOwnerByRepo,
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			newCommitSHA := "new-commit-sha"

			_, _ = w.Write(mock.MustMarshal(github.Tree{
				SHA: &newCommitSHA,
			}))
		}),
	)

	mockValidCreateRef := mock.WithRequestMatchHandler(
		mock.PostReposGitRefsByOwnerByRepo,
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			newRefSHA := "new-ref-sha"

			_, _ = w.Write(mock.MustMarshal(github.Reference{
				Ref: &newRefSHA,
			}))
		}),
	)

	mockValidCreatePullRequest := mock.WithRequestMatchHandler(
		mock.PostReposPullsByOwnerByRepo,
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			pullRequestHTMLURL := "https://github.com/my-org/my-repo/pulls/1"

			_, _ = w.Write(mock.MustMarshal(github.PullRequest{
				HTMLURL: &pullRequestHTMLURL,
			}))
		}),
	)

	t.Run("given PGO file, it creates the blob and pushes it into a branch, opening a PR", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,        // Create blob
			mockValidListMatchingRefs,  // Find main branch ref
			mockValidCreateTree,        // Create tree from the main ref (base tree)
			mockValidCreateCommit,      // Create a commit on the new tree
			mockValidCreateRef,         // Create a new ref for the commit
			mockValidCreatePullRequest, // Create the Pull Request
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.NoError(t, err)
		require.NotEmpty(t, pullRequestURL)

		validURL, err := url.Parse(pullRequestURL)
		require.NoError(t, err)
		require.NotNil(t, validURL)
	})

	t.Run("when there is a problem uploading the blob, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.PostReposGitBlobsByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to upload blob")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there is a problem listing the refs for a repo, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mock.WithRequestMatchHandler(
				mock.EndpointPattern{
					Pattern: "/repos/{owner}/{repo}/git/matching-refs/",
					Method:  "GET",
				},
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to list matching refs")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there are no matching refs, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mock.WithRequestMatchHandler(
				mock.EndpointPattern{
					Pattern: "/repos/{owner}/{repo}/git/matching-refs/",
					Method:  "GET",
				},
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write(mock.MustMarshal([]*github.Reference{}))
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there is a problem creating the new tree, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mockValidListMatchingRefs,
			mock.WithRequestMatchHandler(
				mock.PostReposGitTreesByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to create tree")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there is a problem creating a new commit, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mockValidListMatchingRefs,
			mockValidCreateTree,
			mock.WithRequestMatchHandler(
				mock.PostReposGitCommitsByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to create commit")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there is a problem creating a new commit, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mockValidListMatchingRefs,
			mockValidCreateTree,
			mock.WithRequestMatchHandler(
				mock.PostReposGitCommitsByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to create commit")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there is a problem creating a new ref, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mockValidListMatchingRefs,
			mockValidCreateTree,
			mockValidCreateCommit,
			mock.WithRequestMatchHandler(
				mock.PostReposGitRefsByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to create new ref")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})

	t.Run("when there is a problem creating the pull request, an error is returned", func(t *testing.T) {
		t.Parallel()

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mockValidCreateBlob,
			mockValidListMatchingRefs,
			mockValidCreateTree,
			mockValidCreateCommit,
			mockValidCreateRef,
			mock.WithRequestMatchHandler(
				mock.PostReposPullsByOwnerByRepo,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mock.WriteError(w, http.StatusInternalServerError, "failed to create pull request")
				}),
			),
		)

		ghClient := github.NewClient(mockedHTTPClient)
		client := gh.NewClient(ghClient)

		pullRequestURL, err := client.UpdatePGOFile(ctx, opts, []byte("some content"))
		require.Error(t, err)
		require.Empty(t, pullRequestURL)
	})
}

func TestNewClientWithAccessToken(t *testing.T) {
	ctx := context.Background()

	client := gh.NewClientWithAccessToken(ctx, "access-token")
	require.NotNil(t, client)
}
