package crawler

import (
	"code/internal/domain"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func fetchAssetInfo(ctx context.Context, opts domain.Options, targetURL string, timeout time.Duration, limiter *rateLimiter) domain.Asset {
	a := domain.Asset{
		URL:        targetURL,
		Type:       "other",
		StatusCode: 0,
		SizeBytes:  0,
		Error:      "",
	}

	attempts := 1
	if opts.Retries > 0 {
		attempts = 1 + opts.Retries
	}

	var lastErr error
	var lastStatus int
	var lastSize int64
	var lastSizeErr error

attemptLoop:
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if err := retryPause(ctx); err != nil {
				lastErr = err
				break attemptLoop
			}
		}

		if err := limiter.Wait(ctx); err != nil {
			lastErr = err
			break attemptLoop
		}

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
		if err != nil {
			cancel()

			lastErr = err

			continue
		}

		if opts.UserAgent != "" {
			req.Header.Set("User-Agent", opts.UserAgent)
		}

		resp, err := opts.HTTPClient.Do(req)
		if err != nil {
			cancel()

			if ctx.Err() != nil {
				lastErr = ctx.Err()

				break attemptLoop
			}

			lastErr = err

			continue
		}

		lastStatus = resp.StatusCode

		if lastStatus >= 400 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
			_ = resp.Body.Close()

			cancel()

			lastErr = nil
			lastSize = 0
			lastSizeErr = nil

			if shouldRetryStatus(lastStatus) && i < attempts-1 {
				continue
			}

			break
		}

		size, serr := assetSizeFromResponse(resp)
		lastSize = size
		lastSizeErr = serr

		_ = resp.Body.Close()

		cancel()

		lastErr = nil

		if shouldRetryStatus(lastStatus) && i < attempts-1 {
			continue
		}

		break
	}

	a.StatusCode = lastStatus
	a.SizeBytes = lastSize

	if lastErr != nil {
		a.Error = lastErr.Error()

		return a
	}

	if lastStatus >= 400 {
		if text := http.StatusText(lastStatus); text != "" {
			a.Error = text
		} else {
			a.Error = "http status " + strconv.Itoa(lastStatus)
		}

		return a
	}

	if lastSizeErr != nil {
		a.SizeBytes = 0
		a.Error = "size: " + lastSizeErr.Error()
	}

	return a
}

func assetSizeFromResponse(resp *http.Response) (int64, error) {
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		n, err := strconv.ParseInt(strings.TrimSpace(cl), 10, 64)
		if err == nil && n >= 0 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))

			return n, nil
		}
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, defaultAssetLimit+1))
	if err != nil {
		return 0, err
	}

	if int64(len(b)) > defaultAssetLimit {
		return 0, errors.New("asset body too large to measure")
	}

	return int64(len(b)), nil
}
