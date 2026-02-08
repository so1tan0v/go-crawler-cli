package crawler_test

import (
	"code/crawler"
	"code/src/domain"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAnalyzeSuccess200(t *testing.T) {
	t.Parallel()

	targetURL := "https://example.com"

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != targetURL {
				return nil, errors.New("unexpected url")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        targetURL,
		Depth:      1,
		Retries:    0,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))

	assert.Equal(t, targetURL, report.RootURL)
	require.Len(t, report.Pages, 1)
	assert.Equal(t, 200, report.Pages[0].HTTPStatus)
	assert.Equal(t, "ok", report.Pages[0].Status)
	assert.False(t, report.GeneratedAt.IsZero())
}

func TestAnalyzeNon2xxStatusIsFailed(t *testing.T) {
	t.Parallel()

	targetURL := "https://example.com/not-found"

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("nope")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        targetURL,
		Depth:      1,
		Retries:    0,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))

	require.Len(t, report.Pages, 1)
	assert.Equal(t, 404, report.Pages[0].HTTPStatus)
	assert.Equal(t, "failed", report.Pages[0].Status)
}

func TestAnalyzeNetworkErrorReturnsErrorAndReport(t *testing.T) {
	t.Parallel()

	targetURL := "https://example.com"

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("network failure")
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        targetURL,
		Depth:      1,
		Retries:    0,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.Error(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))

	require.Len(t, report.Pages, 1)
	assert.Equal(t, "failed", report.Pages[0].Status)
	assert.Contains(t, report.Pages[0].Error, "network failure")
}

func TestAnalyzeTimeout(t *testing.T) {
	t.Parallel()

	targetURL := "https://example.com"

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        targetURL,
		Depth:      1,
		Retries:    0,
		Timeout:    5 * time.Millisecond,
		HTTPClient: client,
	})
	require.Error(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))

	require.Len(t, report.Pages, 1)
	assert.Equal(t, "failed", report.Pages[0].Status)
	assert.NotEmpty(t, report.Pages[0].Error)
}
