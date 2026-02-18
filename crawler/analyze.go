package crawler

import (
	"code/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"sync"
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
	assetCache := newAssetCache()
	linkCache := newLinkCheckCache()

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
	workers := opts.Concurrency
	if workers <= 0 {
		workers = 1
	}

	maxDepthLevels := opts.Depth
	if maxDepthLevels <= 0 {
		maxDepthLevels = 1
	}

	seen := map[string]struct{}{opts.URL: {}}
	currLevel := []string{opts.URL}

	var crawlErr error

	type pageResult struct {
		page      domain.Page
		pageLinks []string
		err       error
	}

	processPage := func(pageURL string, depth int) pageResult {
		status, body, ferr := fetchHTML(ctx, opts, pageURL, timeout, limiter)
		page := domain.Page{
			URL:          pageURL,
			Depth:        depth,
			HTTPStatus:   status,
			Status:       "",
			Error:        "",
			SEO:          domain.SEO{},
			DiscoveredAt: nowUTC(),
		}

		if ferr != nil {
			page.Status = "error"
			page.Error = ferr.Error()

			return pageResult{page: page, err: ferr}
		}

		if status >= 200 && status < 400 {
			page.Status = "ok"
		} else {
			page.Status = "error"

			return pageResult{page: page}
		}

		page.Assets = []domain.Asset{}
		page.BrokenLinks = []domain.BrokenLink{}

		if len(body) == 0 {
			return pageResult{page: page}
		}

		page.SEO = extractSEO(body)
		pageLinks, _ := extractPageLinks(pageURL, body)
		assets, _ := extractAssets(pageURL, body)

		seenAssets := make(map[string]struct{})
		for _, a := range assets {
			if _, ok := seenAssets[a.URL]; ok {
				continue
			}

			seenAssets[a.URL] = struct{}{}

			info, cerr := assetCache.GetOrCompute(ctx, a.URL, func() domain.Asset {
				x := fetchAssetInfo(ctx, opts, a.URL, timeout, limiter)
				x.URL = a.URL
				x.Type = a.Type
				return x
			})
			if cerr != nil {
				info = domain.Asset{
					URL:        a.URL,
					Type:       a.Type,
					StatusCode: 0,
					SizeBytes:  0,
					Error:      cerr.Error(),
				}
			}

			page.Assets = append(page.Assets, info)
		}

		broken := make([]domain.BrokenLink, 0)
		for _, linkURL := range pageLinks {
			st, lerr := linkCache.GetOrCompute(ctx, linkURL, func() (int, error) {
				return checkURL(ctx, opts, linkURL, timeout, limiter)
			})

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

		return pageResult{page: page, pageLinks: pageLinks}
	}

	for depth := 0; depth < maxDepthLevels && len(currLevel) > 0; depth++ {
		if ctx.Err() != nil {
			crawlErr = ctx.Err()
			break
		}

		jobs := make(chan string)
		results := make(chan pageResult, len(currLevel))
		var wg sync.WaitGroup

		for i := 0; i < workers; i++ {
			wg.Add(1)

			go func(d int) {
				defer wg.Done()

				for u := range jobs {
					results <- processPage(u, d)
				}
			}(depth)
		}

		for _, u := range currLevel {
			jobs <- u
		}

		close(jobs)
		wg.Wait()
		close(results)

		levelResults := make([]pageResult, 0, len(currLevel))
		for r := range results {
			levelResults = append(levelResults, r)
		}
		sort.Slice(levelResults, func(i, j int) bool { return levelResults[i].page.URL < levelResults[j].page.URL })

		nextLevel := make([]string, 0)
		for _, r := range levelResults {
			res.Pages = append(res.Pages, r.page)
			if r.err != nil && crawlErr == nil {
				crawlErr = r.err
			}

			if depth+1 >= maxDepthLevels {
				continue
			}

			for _, linkURL := range r.pageLinks {
				u, perr := url.Parse(linkURL)
				if perr != nil || u.Host != startHost {
					continue
				}

				if _, ok := seen[linkURL]; ok {
					continue
				}

				seen[linkURL] = struct{}{}
				nextLevel = append(nextLevel, linkURL)
			}
		}
		currLevel = nextLevel
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
