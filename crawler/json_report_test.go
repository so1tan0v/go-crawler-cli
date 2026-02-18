package crawler

import (
	"code/internal/domain"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAnalyzeJSONReportMatchesReference(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2024, 6, 1, 12, 34, 56, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixed }

	t.Cleanup(func() { timeNow = oldNow })

	root := "https://example.com"
	missing := "https://example.com/missing"
	asset := "https://example.com/static/logo.png"

	htmlBody := `<!doctype html><html><head>
<title>Example title</title>
<meta name="description" content="Example description">
<link rel="stylesheet" href="` + asset + `">
</head><body>
<h1>Hello</h1>
<a href="` + missing + `">missing</a>
</body></html>`

	client := &http.Client{
		Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case root:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(htmlBody)),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			case missing:
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(strings.NewReader("Not Found")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			case asset:
				h := make(http.Header)
				h.Set("Content-Length", "12345")

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("x")),
					Header:     h,
					Request:    r,
				}, nil
			default:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			}
		}),
	}

	out, err := Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Retries:    0,
		Delay:      0,
		RPS:        0,
		Timeout:    2 * time.Second,
		IndentJSON: false,
		HTTPClient: client,
	})
	require.NoError(t, err)

	ref := `{"root_url":"https://example.com","depth":1,"generated_at":"2024-06-01T12:34:56Z","pages":[{"url":"https://example.com","depth":0,"http_status":200,"status":"ok","seo":{"has_title":true,"title":"Example title","has_description":true,"description":"Example description","has_h1":true},"broken_links":[{"url":"https://example.com/missing","status_code":404,"error":"Not Found"}],"assets":[{"url":"https://example.com/static/logo.png","type":"style","status_code":200,"size_bytes":12345}],"discovered_at":"2024-06-01T12:34:56Z"}]}`

	var gotObj any
	var refObj any
	require.NoError(t, json.Unmarshal(out, &gotObj))
	require.NoError(t, json.Unmarshal([]byte(ref), &refObj))
	require.Equal(t, refObj, gotObj)

	require.Equal(t, ref, string(out))
}

func TestAnalyzeIndentJSONChangesOnlyFormatting(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2024, 6, 1, 12, 34, 56, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixed }

	t.Cleanup(func() { timeNow = oldNow })

	root := "https://example.com"
	htmlBody := `<!doctype html><html><head><title>Example</title></head><body><h1>x</h1></body></html>`

	client := &http.Client{
		Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(htmlBody)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	outCompact, err := Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Timeout:    2 * time.Second,
		IndentJSON: false,
		HTTPClient: client,
	})
	require.NoError(t, err)

	outPretty, err := Analyze(context.Background(), domain.Options{
		URL:        root,
		Depth:      1,
		Timeout:    2 * time.Second,
		IndentJSON: true,
		HTTPClient: client,
	})
	require.NoError(t, err)

	require.NotEqual(t, string(outCompact), string(outPretty))
	require.Contains(t, string(outPretty), "\n")

	var a any
	var b any

	require.NoError(t, json.Unmarshal(outCompact, &a))
	require.NoError(t, json.Unmarshal(outPretty, &b))
	require.Equal(t, a, b)
}
