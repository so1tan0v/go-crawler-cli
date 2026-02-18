package crawler

import (
	"code/internal/domain"
	"context"
	"io"
	"net/http"
	"time"
)

func fetchHTML(ctx context.Context, opts domain.Options, targetURL string, timeout time.Duration, limiter *rateLimiter) (int, []byte, error) {
	attempts := 1
	if opts.Retries > 0 {
		attempts = 1 + opts.Retries
	}

	var lastErr error
	var lastStatus int
	var lastBody []byte

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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, defaultHTMLLimit))
		_ = resp.Body.Close()

		cancel()

		lastBody = body
		lastErr = nil

		if shouldRetryStatus(lastStatus) && i < attempts-1 {
			continue
		}

		break
	}

	return lastStatus, lastBody, lastErr
}

func checkURL(ctx context.Context, opts domain.Options, linkURL string, timeout time.Duration, limiter *rateLimiter) (int, error) {
	status, err := doRequest(ctx, opts, http.MethodHead, linkURL, timeout, limiter)
	if err != nil {
		return 0, err
	}

	if status == http.StatusMethodNotAllowed || status == http.StatusNotImplemented {
		return doRequest(ctx, opts, http.MethodGet, linkURL, timeout, limiter)
	}

	return status, nil
}

func doRequest(ctx context.Context, opts domain.Options, method, targetURL string, timeout time.Duration, limiter *rateLimiter) (int, error) {
	attempts := 1
	if opts.Retries > 0 {
		attempts = 1 + opts.Retries
	}

	var lastErr error
	var lastStatus int

attemptLoop:
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if err := retryPause(ctx); err != nil {
				return 0, err
			}
		}

		if err := limiter.Wait(ctx); err != nil {
			return 0, err
		}

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		req, err := http.NewRequestWithContext(reqCtx, method, targetURL, nil)
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
				return 0, ctx.Err()
			}
			lastErr = err

			continue
		}

		lastStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
		_ = resp.Body.Close()
		cancel()

		lastErr = nil

		if shouldRetryStatus(lastStatus) && i < attempts-1 {
			continue
		}

		break attemptLoop
	}

	if lastErr != nil {
		return 0, lastErr
	}

	return lastStatus, nil
}
