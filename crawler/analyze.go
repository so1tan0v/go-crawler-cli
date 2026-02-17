package crawler

import (
	"code/src/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"time"
)

func Analyze(ctx context.Context, opts domain.Options) ([]byte, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	interval := time.Duration(0)
	if opts.RPS > 0 {
		interval = time.Second / time.Duration(opts.RPS)
	} else if opts.Delay > 0 {
		interval = opts.Delay
	}

	limiter := newRateLimiter(interval)
	assetCache := make(map[string]domain.Asset)

	res := domain.AnalyzeResult{
		RootURL:     opts.URL,
		Depth:       opts.Depth,
		GeneratedAt: nowUTC(),
		Pages:       []domain.Page{},
	}

	if opts.URL == "" {
		page := domain.Page{
			URL:          opts.URL,
			Depth:        0,
			Status:       "error",
			Error:        "url is required",
			SEO:          domain.SEO{},
			DiscoveredAt: nowUTC(),
		}

		res.Pages = append(res.Pages, page)
		out, _ := json.Marshal(res)

		return out, errors.New(page.Error)
	}

	startURL, err := url.ParseRequestURI(opts.URL)
	if err != nil {
		page := domain.Page{
			URL:          opts.URL,
			Depth:        0,
			Status:       "error",
			Error:        "invalid url",
			SEO:          domain.SEO{},
			DiscoveredAt: nowUTC(),
		}

		res.Pages = append(res.Pages, page)
		out, _ := json.Marshal(res)

		return out, err
	}

	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{}
	}

	startHost := startURL.Host

	type queueItem struct {
		url   string
		depth int
	}

	maxDepthLevels := opts.Depth
	if maxDepthLevels <= 0 {
		maxDepthLevels = 1
	}

	seen := map[string]int{opts.URL: 0}
	q := []queueItem{{url: opts.URL, depth: 0}}

	var crawlErr error

	for len(q) > 0 {
		if ctx.Err() != nil {
			crawlErr = ctx.Err()
			break
		}

		item := q[0]
		q = q[1:]

		status, body, ferr := fetchHTML(ctx, opts, item.url, timeout, limiter)
		page := domain.Page{
			URL:          item.url,
			Depth:        item.depth,
			HTTPStatus:   status,
			Status:       "",
			Error:        "",
			SEO:          domain.SEO{},
			DiscoveredAt: nowUTC(),
		}

		if ferr != nil {
			page.Status = "error"
			page.Error = ferr.Error()
			res.Pages = append(res.Pages, page)

			if crawlErr == nil {
				crawlErr = ferr
			}

			continue
		}

		if status >= 200 && status < 400 {
			page.Status = "ok"
		} else {
			page.Status = "error"
		}

		if len(body) > 0 && status >= 200 && status < 400 {
			page.Assets = []domain.Asset{}
			page.BrokenLinks = []domain.BrokenLink{}

			page.SEO = extractSEO(body)

			pageLinks, _ := extractPageLinks(item.url, body)
			assets, _ := extractAssets(item.url, body)

			seenAssets := make(map[string]struct{})
			assetURLs := make(map[string]struct{})
			for _, a := range assets {
				if _, ok := seenAssets[a.URL]; ok {
					continue
				}

				seenAssets[a.URL] = struct{}{}
				assetURLs[a.URL] = struct{}{}

				if cached, ok := assetCache[a.URL]; ok {
					page.Assets = append(page.Assets, cached)

					continue
				}

				info := fetchAssetInfo(ctx, opts, a.URL, timeout, limiter)
				info.URL = a.URL
				info.Type = a.Type
				assetCache[a.URL] = info
				page.Assets = append(page.Assets, info)
			}

			broken := make([]domain.BrokenLink, 0)
			for _, linkURL := range pageLinks {
				st, lerr := checkURL(ctx, opts, linkURL, timeout, limiter)
				if lerr != nil {
					broken = append(broken, domain.BrokenLink{URL: linkURL, StatusCode: 0, Error: lerr.Error()})
					continue
				}

				if st >= 400 {
					broken = append(broken, domain.BrokenLink{URL: linkURL, StatusCode: st, Error: http.StatusText(st)})
				}
			}

			sort.Slice(broken, func(i, j int) bool { return broken[i].URL < broken[j].URL })
			page.BrokenLinks = broken

			nextDepth := item.depth + 1
			if nextDepth < maxDepthLevels {
				for _, linkURL := range pageLinks {
					u, perr := url.Parse(linkURL)
					if perr != nil {
						continue
					}

					if u.Host != startHost {
						continue
					}

					if _, ok := seen[linkURL]; ok {
						continue

					}
					seen[linkURL] = nextDepth
					q = append(q, queueItem{url: linkURL, depth: nextDepth})
				}
			}
		}

		res.Pages = append(res.Pages, page)
	}

	sort.Slice(res.Pages, func(i, j int) bool {
		if res.Pages[i].Depth != res.Pages[j].Depth {
			return res.Pages[i].Depth < res.Pages[j].Depth
		}

		return res.Pages[i].URL < res.Pages[j].URL
	})

	var (
		out  []byte
		merr error
	)

	if opts.IndentJSON {
		out, merr = json.MarshalIndent(res, "", "  ")
	} else {
		out, merr = json.Marshal(res)
	}
	if merr != nil {
		return nil, merr
	}

	return out, crawlErr
}
