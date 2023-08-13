package pprof_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/macabu/cpgo/internal/pprof"
)

type mockRoundTripper func(r *http.Request) (*http.Response, error)

func (m mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m(r)
}

func TestFetcher(t *testing.T) {
	t.Parallel()

	t.Run("given a valid profile, then it parses it and returns no error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		profileValid := &profile.Profile{
			TimeNanos:     10000,
			PeriodType:    &profile.ValueType{Type: "cpu", Unit: "milliseconds"},
			Period:        1,
			DurationNanos: 10e9,
			SampleType: []*profile.ValueType{
				{Type: "samples", Unit: "count"},
				{Type: "cpu", Unit: "milliseconds"},
			},
			Sample: []*profile.Sample{},
		}

		var w bytes.Buffer

		err := profileValid.WriteUncompressed(&w)
		require.NoError(t, err)

		client := &http.Client{
			Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(w.Bytes())),
				}, nil
			}),
		}

		actualProfile, err := pprof.NewFetcher(client).FromURL(ctx, "does-not-matter")
		require.NoError(t, err)
		require.NotNil(t, actualProfile)
		require.NoError(t, actualProfile.CheckValid())
	})

	t.Run("when creating the request fails due to an invalid url, then an error is returned", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		actualProfile, err := pprof.NewFetcher(nil).FromURL(ctx, string([]byte{0x7f}))
		require.Error(t, err)
		require.Nil(t, actualProfile)
	})

	t.Run("when the server responds with an error, then an error is returned", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockErr := fmt.Errorf("mock error")

		client := &http.Client{
			Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`An internal error.`))),
				}, mockErr
			}),
		}

		actualProfile, err := pprof.NewFetcher(client).FromURL(ctx, "does-not-matter")
		require.ErrorIs(t, err, mockErr)
		require.Nil(t, actualProfile)
	})

	t.Run("given a invalid profile, then it returns an error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		client := &http.Client{
			Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`Something that is not a profile.`))),
				}, nil
			}),
		}

		actualProfile, err := pprof.NewFetcher(client).FromURL(ctx, "does-not-matter")
		require.Error(t, err)
		require.Nil(t, actualProfile)
	})
}
