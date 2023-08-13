package gh

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"

	"github.com/macabu/cpgo/internal/gitops"
)

type Options struct {
	Repo       Repository
	Filename   string
	MainBranch string
}

type Client struct {
	github *github.Client
}

func NewClient(client *github.Client) *Client {
	return &Client{
		github: client,
	}
}

func NewClientWithAccessToken(ctx context.Context, accessToken string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)

	return NewClient(github.NewClient(tc))
}

// ExistingPGOFileURL searches for the Options.Filename in the repository. Returns a signed URL to download it.
func (c Client) ExistingPGOFileURL(ctx context.Context, opts Options) (string, error) {
	fileContent, _, resp, err := c.github.Repositories.GetContents(ctx, opts.Repo.Org, opts.Repo.Name, opts.Filename, nil)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("github.Repositories.GetContents: %w", gitops.ErrPGOFileNotFound)
		}

		return "", fmt.Errorf("github.Repositories.GetContents: %w", err)
	}

	return *fileContent.DownloadURL, nil
}

// UpdatePGOFile creates the blob, branch and a pull request with the new PGO file. Returns the pull request URL.
func (c Client) UpdatePGOFile(ctx context.Context, opts Options, fileContent []byte) (string, error) {
	blobSHA, err := c.createBlob(ctx, opts, fileContent)
	if err != nil {
		return "", fmt.Errorf("createBlob: %w", err)
	}

	mainBranchRef, err := c.findMainBranchRef(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("findMainBranchRef: %w", err)
	}

	tree, err := c.createTree(ctx, opts, *blobSHA, *mainBranchRef.Object.SHA)
	if err != nil {
		return "", fmt.Errorf("createTree: %w", err)
	}

	commitSHA, err := c.commitFile(ctx, opts, tree, *mainBranchRef.Object.SHA)
	if err != nil {
		return "", fmt.Errorf("commitFile: %w", err)
	}

	prRef, err := c.createNewRef(ctx, opts, commitSHA)
	if err != nil {
		return "", fmt.Errorf("createNewRef: %w", err)
	}

	prURL, err := c.openPullRequest(ctx, opts, *prRef, *mainBranchRef.Ref)
	if err != nil {
		return "", fmt.Errorf("openPullRequest: %w", err)
	}

	return prURL, nil
}

// createBlob object as a base64 encoded file. Returns the blob SHA.
func (c Client) createBlob(ctx context.Context, opts Options, fileContent []byte) (*string, error) {
	content := base64.StdEncoding.EncodeToString(fileContent)

	blob, _, err := c.github.Git.CreateBlob(ctx, opts.Repo.Org, opts.Repo.Name, &github.Blob{
		Content:  github.String(content),
		Encoding: github.String("base64"),
	})
	if err != nil {
		return nil, fmt.Errorf("github.Git.CreateBlob: %w", err)
	}

	return blob.SHA, nil
}

// findMainBranchRef lists all refs and find the one used as the main branch based on the options. Returns the ref obj.
func (c Client) findMainBranchRef(ctx context.Context, opts Options) (*github.Reference, error) {
	refs, _, err := c.github.Git.ListMatchingRefs(ctx, opts.Repo.Org, opts.Repo.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("github.Git.ListMatchingRefs: %w", err)
	}

	for _, ref := range refs {
		if ref != nil && ref.Ref != nil && strings.HasSuffix(*ref.Ref, opts.MainBranch) {
			return ref, nil
		}
	}

	return nil, fmt.Errorf("could not find ref for branch %v", opts.MainBranch)
}

// createTree using the main branch as the base of the tree. Returns the tree object.
func (c Client) createTree(ctx context.Context, opts Options, blobSHA, mainRef string) (*github.Tree, error) {
	entry := &github.TreeEntry{
		SHA:  github.String(blobSHA),
		Type: github.String("blob"),
		Mode: github.String("100644"),
		Path: github.String(opts.Filename),
	}

	tree, _, err := c.github.Git.CreateTree(ctx, opts.Repo.Org, opts.Repo.Name, mainRef, []*github.TreeEntry{entry})
	if err != nil {
		return nil, fmt.Errorf("github.Git.CreateTree: %w", err)
	}

	return tree, nil
}

// commitFile on the newly created tree using the latest main branch commit sha as its parent. Returns the commit SHA.
func (c Client) commitFile(ctx context.Context, opts Options, tree *github.Tree, mainRef string) (*string, error) {
	commitReq := &github.Commit{
		Tree:    tree,
		Message: github.String("chore: update PGO file with new traces"),
		Author: &github.CommitAuthor{
			Name:  github.String("CPGO Automatic Updates"),
			Email: github.String("example@example.com"),
			Date: &github.Timestamp{
				Time: time.Now(),
			},
		},
		Parents: []*github.Commit{
			{SHA: github.String(mainRef)},
		},
	}

	commitRes, _, err := c.github.Git.CreateCommit(ctx, opts.Repo.Org, opts.Repo.Name, commitReq)
	if err != nil {
		return nil, fmt.Errorf("github.Git.CreateCommit: %w", err)
	}

	return commitRes.SHA, nil
}

// createNewRef with the new commit sha, this is the branch. Returns the reference SHA.
func (c Client) createNewRef(ctx context.Context, opts Options, commitSHA *string) (*string, error) {
	refName := "refs/heads/cpgo-update-" + strconv.Itoa(int(time.Now().Unix()))

	ref, _, err := c.github.Git.CreateRef(ctx, opts.Repo.Org, opts.Repo.Name, &github.Reference{
		Ref: github.String(refName),
		Object: &github.GitObject{
			SHA: commitSHA,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("github.Git.CreateRef: %w", err)
	}

	return ref.Ref, nil
}

// openPullRequest from the newly created ref using the main branch ref as a base. Returns the pull request URL.
func (c Client) openPullRequest(ctx context.Context, opts Options, prRef, mainRef string) (string, error) {
	const refsPrefix = "refs/heads/"

	head, _ := strings.CutPrefix(prRef, refsPrefix)
	base, _ := strings.CutPrefix(mainRef, refsPrefix)

	title := fmt.Sprintf("Update PGO file [%v]", time.Now().Format(time.RFC3339))
	body := "This pull request updates the PGO file with newer traces.\nFeel free to merge or close it."

	pr, _, err := c.github.PullRequests.Create(ctx, opts.Repo.Org, opts.Repo.Name, &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(head),
		Base:  github.String(base),
		Body:  github.String(body),
	})
	if err != nil {
		return "", fmt.Errorf("github.PullRequests.Create: %w", err)
	}

	return *pr.HTMLURL, nil
}
