package crawler

import (
	"code/src/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout = 15 * time.Second

/*Analyze makes an HTTP request to the root URL and returns a draft JSON report.*/
func Analyze(ctx context.Context, opts domain.Options) ([]byte, error) {
	res := domain.AnalyzeResult{
		RootURL:     opts.URL,
		Depth:       opts.Depth,
		GeneratedAt: time.Now().UTC(),
		Pages:       []domain.Page{},
	}

	page := domain.Page{
		URL:   opts.URL,
		Depth: 0,
	}

	if opts.URL == "" {
		page.Status = "failed"
		page.Error = "url is required"

		res.Pages = append(res.Pages, page)
		out, _ := json.Marshal(res)

		return out, errors.New(page.Error)
	}

	if _, err := url.ParseRequestURI(opts.URL); err != nil {
		page.Status = "failed"
		page.Error = "invalid url"

		res.Pages = append(res.Pages, page)
		out, _ := json.Marshal(res)

		return out, err
	}

	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{}
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	attempts := 1
	if opts.Retries > 0 {
		attempts = 1 + opts.Retries
	}

	var lastErr error
	var lastStatus int

	for i := 0; i < attempts; i++ {
		if i > 0 && opts.Delay > 0 {
			select {
			case <-time.After(opts.Delay):
			case <-ctx.Done():
				lastErr = ctx.Err()
				break
			}
		}

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, opts.URL, nil)
		if err != nil {
			cancel()

			lastErr = err

			continue
		}

		if opts.UserAgent != "" {
			req.Header.Set("User-Agent", opts.UserAgent)
		}

		resp, err := opts.HTTPClient.Do(req)
		cancel()
		if err != nil {
			lastErr = err

			continue
		}

		lastStatus = resp.StatusCode

		_ = resp.Body.Close()
		lastErr = nil

		break
	}

	page.HTTPStatus = lastStatus

	if lastErr != nil {
		page.Status = "failed"
		page.Error = lastErr.Error()
	} else if lastStatus >= 200 && lastStatus < 400 {
		page.Status = "ok"
	} else {
		page.Status = "failed"
	}

	res.Pages = append(res.Pages, page)

	var (
		out []byte
		err error
	)

	if opts.IndentJSON {
		out, err = json.MarshalIndent(res, "", "  ")
	} else {
		out, err = json.Marshal(res)
	}
	if err != nil {
		return nil, err
	}

	return out, lastErr
}
