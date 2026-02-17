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
	"sync"
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

	for _, tc := range []struct {
		name       string
		statusCode int
	}{
		{name: "404", statusCode: 404},
		{name: "500", statusCode: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			targetURL := "https://example.com/status"

			client := &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.statusCode,
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
			assert.Equal(t, tc.statusCode, report.Pages[0].HTTPStatus)
			assert.Equal(t, "error", report.Pages[0].Status)
		})
	}
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
	assert.Equal(t, "error", report.Pages[0].Status)
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
	assert.Equal(t, "error", report.Pages[0].Status)
	assert.NotEmpty(t, report.Pages[0].Error)
}

func TestAnalyzeBrokenLinksFromHTML(t *testing.T) {
	t.Parallel()

	pageURL := "http://simple.test/blog/index.html"
	okCSS := "http://simple.test/assets/ok.css"
	ghostCSS := "http://simple.test/assets/ghost.css"
	okURL := "http://simple.test/ok"
	missingURL := "http://simple.test/missing"
	cdnJS := "https://cdn.simple.test/app.js"

	htmlBody := `<!doctype html><html><head>
<link rel="stylesheet" href="/assets/ok.css">
<link rel="stylesheet" href="/assets/ghost.css">
<script src="https://cdn.simple.test/app.js"></script>
</head><body>
<a href="/ok">ok</a>
<a href="/missing">missing</a>
<a href="#anchor">anchor</a>
<a href="mailto:test@example.com">mail</a>
</body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case pageURL:
				if r.Method != http.MethodGet {
					return nil, errors.New("unexpected method for page")
				}

				h := make(http.Header)
				h.Set("Content-Type", "text/html; charset=utf-8")

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(htmlBody)),
					Header:     h,
					Request:    r,
				}, nil
			case okCSS, okURL:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			case ghostCSS, missingURL:
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(strings.NewReader("nope")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			case cdnJS:
				return nil, errors.New("no such host")
			default:
				return nil, errors.New("unexpected url: " + r.URL.String())
			}
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        pageURL,
		Depth:      1,
		Retries:    0,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)

	page := report.Pages[0]
	assert.Equal(t, pageURL, page.URL)
	assert.Equal(t, 200, page.HTTPStatus)
	assert.Equal(t, "ok", page.Status)

	got := make(map[string]domain.BrokenLink, len(page.BrokenLinks))
	for _, bl := range page.BrokenLinks {
		got[bl.URL] = bl
	}

	_, ok := got[okCSS]
	assert.False(t, ok)
	_, ok = got[ghostCSS]
	assert.False(t, ok)

	_, ok = got[okURL]
	assert.False(t, ok)

	require.Contains(t, got, missingURL)
	assert.Equal(t, 404, got[missingURL].StatusCode)
	assert.Equal(t, "Not Found", got[missingURL].Error)

	_, ok = got[cdnJS]
	assert.False(t, ok)
}

func TestAnalyzeSEOAllTagsAndEntities(t *testing.T) {
	t.Parallel()

	pageURL := "http://example.test"
	htmlBody := `<!doctype html><html><head>
<title>  Example &amp; Test  </title>
<meta name="description" content="  Best &amp; brightest  ">
</head><body>
<h1>  Hello &amp; World </h1>
</body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != pageURL {
				return nil, errors.New("unexpected url: " + r.URL.String())
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(htmlBody)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        pageURL,
		Depth:      1,
		Retries:    0,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)

	seo := report.Pages[0].SEO
	assert.True(t, seo.HasTitle)
	assert.Equal(t, "Example & Test", seo.Title)
	assert.True(t, seo.HasDescription)
	assert.Equal(t, "Best & brightest", seo.Description)
	assert.True(t, seo.HasH1)
}

func TestAnalyzeSEOMissingTags(t *testing.T) {
	t.Parallel()

	pageURL := "http://example.test/no-seo"
	htmlBody := `<!doctype html><html><head></head><body><p>no tags</p></body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(htmlBody)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        pageURL,
		Depth:      1,
		Retries:    0,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)

	seo := report.Pages[0].SEO
	assert.False(t, seo.HasTitle)
	assert.Empty(t, seo.Title)
	assert.False(t, seo.HasDescription)
	assert.Empty(t, seo.Description)
	assert.False(t, seo.HasH1)
}

func TestAnalyzeCrawlDepth1OnlyRoot(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	a := "http://simple.test/a"
	b := "http://simple.test/b"
	ext := "https://external.test/x"

	rootHTML := `<!doctype html><html><head><title>root</title></head><body>
<a href="/a">a</a>
<a href="/b">b</a>
<a href="` + ext + `">ext</a>
</body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case root:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(rootHTML)),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			case a, b, ext:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			default:
				return nil, errors.New("unexpected url: " + r.URL.String())
			}
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1, // only root
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)
	assert.Equal(t, root, report.Pages[0].URL)
	assert.Equal(t, 0, report.Pages[0].Depth)
}

func TestAnalyzeCrawlDepth2InternalOnlyDedupExternalExcluded(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	a := "http://simple.test/a"
	b := "http://simple.test/b"
	ext := "https://external.test/x"

	rootHTML := `<!doctype html><html><head><title>Root</title></head><body>
<a href="/a">a</a>
<a href="/a">a-dup</a>
<a href="/b">b</a>
<a href="` + ext + `">ext</a>
</body></html>`

	aHTML := `<!doctype html><html><head>
<title>Page A</title>
<meta name="description" content="Desc A">
</head><body><h1>Header A</h1><a href="/b">to b</a></body></html>`

	bHTML := `<!doctype html><html><head></head><body><p>B</p></body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodGet {
				switch r.URL.String() {
				case root:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
				case a:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(aHTML)), Header: make(http.Header), Request: r}, nil
				case b:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(bHTML)), Header: make(http.Header), Request: r}, nil
				}
			}

			switch r.URL.String() {
			case a, b, ext:
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
			default:
				return nil, errors.New("unexpected url: " + r.Method + " " + r.URL.String())
			}
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      2,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))

	got := make(map[string]domain.Page, len(report.Pages))
	for _, p := range report.Pages {
		got[p.URL] = p
	}

	require.Len(t, report.Pages, 3)
	require.Contains(t, got, root)
	require.Contains(t, got, a)
	require.Contains(t, got, b)
	assert.NotContains(t, got, ext)

	assert.Equal(t, 0, got[root].Depth)
	assert.Equal(t, 1, got[a].Depth)
	assert.Equal(t, 1, got[b].Depth)

	assert.True(t, got[root].SEO.HasTitle)
	assert.Equal(t, "Root", got[root].SEO.Title)

	assert.True(t, got[a].SEO.HasTitle)
	assert.Equal(t, "Page A", got[a].SEO.Title)
	assert.True(t, got[a].SEO.HasDescription)
	assert.Equal(t, "Desc A", got[a].SEO.Description)
	assert.True(t, got[a].SEO.HasH1)

	assert.False(t, got[b].SEO.HasTitle)
	assert.False(t, got[b].SEO.HasDescription)
	assert.False(t, got[b].SEO.HasH1)
}

func TestAnalyzeRateLimitDelayGlobal(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	l1 := "http://simple.test/a"
	l2 := "http://simple.test/b"
	l3 := "http://simple.test/c"
	l4 := "https://external.test/x"

	rootHTML := `<!doctype html><html><head><title>root</title></head><body>
<a href="` + l1 + `">1</a>
<a href="` + l2 + `">2</a>
<a href="` + l3 + `">3</a>
<a href="` + l4 + `">4</a>
</body></html>`

	var times []time.Time
	var mu sync.Mutex

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			times = append(times, time.Now())
			mu.Unlock()

			if r.Method == http.MethodGet && r.URL.String() == root {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
		}),
	}

	delay := 20 * time.Millisecond
	start := time.Now()
	_, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Delay:      delay,
		RPS:        0,
		Timeout:    2 * time.Second,
		HTTPClient: client,
	})
	elapsed := time.Since(start)
	require.NoError(t, err)

	mu.Lock()
	n := len(times)
	mu.Unlock()

	require.GreaterOrEqual(t, n, 5)

	minExpected := time.Duration(n-1) * delay
	require.GreaterOrEqual(t, elapsed+5*time.Millisecond, minExpected)
}

func TestAnalyzeRateLimitRPSOverridesDelay(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	l1 := "http://simple.test/a"
	l2 := "http://simple.test/b"
	l3 := "http://simple.test/c"
	l4 := "http://simple.test/d"

	rootHTML := `<!doctype html><html><head><title>root</title></head><body>
<a href="` + l1 + `">1</a>
<a href="` + l2 + `">2</a>
<a href="` + l3 + `">3</a>
<a href="` + l4 + `">4</a>
</body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodGet && r.URL.String() == root {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
		}),
	}

	delay := 200 * time.Millisecond
	rps := 50

	start := time.Now()
	_, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Delay:      delay,
		RPS:        rps,
		Timeout:    2 * time.Second,
		HTTPClient: client,
	})
	elapsed := time.Since(start)
	require.NoError(t, err)

	require.Less(t, elapsed, 500*time.Millisecond)
	require.GreaterOrEqual(t, elapsed, 60*time.Millisecond)
}

func TestAnalyzeRateLimitCancelStopsWaiting(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	l1 := "http://simple.test/a"
	rootHTML := `<!doctype html><html><head><title>root</title></head><body><a href="` + l1 + `">1</a></body></html>`

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodGet && r.URL.String() == root {
				cancel()
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
		}),
	}

	start := time.Now()
	_, err := crawler.Analyze(ctx, domain.Options{
		URL:        root,
		Depth:      1,
		Delay:      500 * time.Millisecond,
		Timeout:    2 * time.Second,
		HTTPClient: client,
	})
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 200*time.Millisecond)
	_ = err
}

func TestAnalyzeRetriesStopsAfterRetriesPlusOneOnNetworkErrors(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"

	var mu sync.Mutex
	calls := 0

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			calls++
			mu.Unlock()

			return nil, errors.New("temporary network failure")
		}),
	}

	_, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Retries:    2,
		Timeout:    50 * time.Millisecond,
		HTTPClient: client,
	})
	require.Error(t, err)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 3, calls)
}

func TestAnalyzeRetriesSucceedsOnSecondAttempt(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"

	var mu sync.Mutex
	calls := 0

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()

			if n == 1 {
				return nil, errors.New("temporary network failure")
			}

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<!doctype html><html><head><title>OK</title></head><body></body></html>`)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Retries:    2,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)
	assert.Equal(t, "ok", report.Pages[0].Status)
	assert.Equal(t, 200, report.Pages[0].HTTPStatus)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 2, calls)
}

func TestAnalyzeNoRetriesOnNonTemporaryStatus(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"

	var mu sync.Mutex
	calls := 0

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			calls++
			mu.Unlock()

			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("nope")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Retries:    2,
		Timeout:    100 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)
	assert.Equal(t, "error", report.Pages[0].Status)
	assert.Equal(t, 404, report.Pages[0].HTTPStatus)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 1, calls)
}

func TestAnalyzeBrokenLinksRetryReflectsLastAttempt(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	link := "http://simple.test/broken"

	var mu sync.Mutex
	linkCalls := 0

	rootHTML := `<!doctype html><html><head><title>root</title></head><body><a href="` + link + `">x</a></body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case root:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(rootHTML)),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			case link:
				mu.Lock()
				linkCalls++
				n := linkCalls
				mu.Unlock()

				if n == 1 {
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(strings.NewReader("fail")),
						Header:     make(http.Header),
						Request:    r,
					}, nil
				}

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			default:
				return nil, errors.New("unexpected url")
			}
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Retries:    1,
		Timeout:    200 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)

	assert.Empty(t, report.Pages[0].BrokenLinks)

	mu.Lock()
	defer mu.Unlock()

	assert.LessOrEqual(t, linkCalls, 2)
}

func TestAnalyzeAssetsCacheAcrossPages(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	a := "http://simple.test/a"
	asset := "http://simple.test/static/app.js"

	rootHTML := `<!doctype html><html><head><title>root</title></head><body>
<a href="/a">a</a>
<script src="/static/app.js"></script>
</body></html>`
	aHTML := `<!doctype html><html><head><title>a</title></head><body>
<script src="/static/app.js"></script>
</body></html>`

	var mu sync.Mutex
	calls := make(map[string]int)

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			calls[r.Method+" "+r.URL.String()]++
			mu.Unlock()

			if r.Method == http.MethodGet {
				switch r.URL.String() {
				case root:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
				case a:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(aHTML)), Header: make(http.Header), Request: r}, nil
				case asset:
					h := make(http.Header)
					h.Set("Content-Length", "3")

					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("hey")), Header: h, Request: r}, nil
				}
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      2,
		Timeout:    200 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 2)

	for _, p := range report.Pages {
		found := false
		for _, as := range p.Assets {
			if as.URL == asset {
				found = true
				assert.Equal(t, "script", as.Type)
				assert.Equal(t, 200, as.StatusCode)
				assert.Equal(t, int64(3), as.SizeBytes)
				assert.Empty(t, as.Error)
			}
		}
		assert.True(t, found)
	}

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 1, calls["GET "+asset], "asset must be fetched once due to cache")
}

func TestAnalyzeAssetSizeWithoutContentLength(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	img := "http://simple.test/static/logo.png"

	rootHTML := `<!doctype html><html><head><title>root</title></head><body>
<img src="/static/logo.png">
</body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodGet && r.URL.String() == root {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
			}

			if r.Method == http.MethodGet && r.URL.String() == img {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("12345")), Header: make(http.Header), Request: r}, nil
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Timeout:    200 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)

	require.Len(t, report.Pages[0].Assets, 1)
	as := report.Pages[0].Assets[0]

	assert.Equal(t, img, as.URL)
	assert.Equal(t, "image", as.Type)
	assert.Equal(t, 200, as.StatusCode)
	assert.Equal(t, int64(5), as.SizeBytes)
	assert.Empty(t, as.Error)
}

func TestAnalyzeAssetErrorStatusInReport(t *testing.T) {
	t.Parallel()

	root := "http://simple.test/index.html"
	css := "http://simple.test/static/app.css"

	rootHTML := `<!doctype html><html><head>
<link rel="stylesheet" href="/static/app.css">
</head><body></body></html>`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodGet && r.URL.String() == root {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(rootHTML)), Header: make(http.Header), Request: r}, nil
			}

			if r.Method == http.MethodGet && r.URL.String() == css {
				return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("fail")), Header: make(http.Header), Request: r}, nil
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
		}),
	}

	out, err := crawler.Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Timeout:    200 * time.Millisecond,
		HTTPClient: client,
	})
	require.NoError(t, err)

	var report domain.AnalyzeResult
	require.NoError(t, json.Unmarshal(out, &report))
	require.Len(t, report.Pages, 1)

	require.Len(t, report.Pages[0].Assets, 1)
	as := report.Pages[0].Assets[0]
	assert.Equal(t, css, as.URL)
	assert.Equal(t, "style", as.Type)
	assert.Equal(t, 500, as.StatusCode)
	assert.Equal(t, int64(0), as.SizeBytes)
	assert.NotEmpty(t, as.Error)
}
