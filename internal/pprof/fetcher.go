package pprof

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/pprof/profile"
)

type Fetcher struct {
	client *http.Client
}

func NewFetcher(client *http.Client) *Fetcher {
	return &Fetcher{
		client: client,
	}
}

// FromURL fetches a profile from the designated `url` and parses it.
func (f Fetcher) FromURL(ctx context.Context, url string) (*profile.Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do: %w", err)
	}

	defer resp.Body.Close()

	prof, err := profile.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("profile.Parse: %w", err)
	}

	return prof, nil
}
