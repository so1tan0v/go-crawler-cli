# Crawler

A command-line tool for analyzing website structure. Crawls a site up to a configurable depth, collects SEO metadata (title, meta description, h1), detects broken links, and lists assets (images, scripts, styles) with their sizes and status codes. Outputs a JSON report suitable for audits and monitoring.

## Installation

``` bash
curl -sfL https://raw.githubusercontent.com/so1tan0v/go-crawler-cli/main/install.sh | sh
```


## Help
``` bash
NAME:
   so1-crawler - analyze a website structure

USAGE:
   so1-crawler [global options] command [command options] <url>

VERSION:
   some-version

GLOBAL OPTIONS:
   --depth int          crawl depth (default: 10)
   --retries int        number of retries for failed requests (default: 1)
   --delay duration     delay between requests (example: 200ms, 1s) (default: 0s)
   --timeout duration   per-request timeout (default: 15s)
   --rps int            limit requests per second (overrides delay) (default: 0)
   --user-agent string  custom user agent
   --workers int        number of concurrent workers (default: 4)
   --indent-json        pretty-print JSON output
   --help, -h           show help
```

## Usage
``` bash
so1-crawler --depth=2 --delay=200ms --workers=5 --indent-json https://ya.ru

>> Output
{
  "root_url": "https://ya.ru",
  "depth": 2,
  "generated_at": "2026-03-02T19:26:19Z",
  "pages": [
    {
      "url": "https://ya.ru",
      "depth": 0,
      "http_status": 200,
      "status": "ok",
      "seo": {
        "has_title": true,
        "title": "Вы не робот?",
        "has_description": false,
        "description": "",
        "has_h1": true
      },
      "broken_links": [],
      "assets": [
        {
          "url": "https://adfstat.yandex.ru/captcha?req_id=1772479582340718-3447255322749179888-balancer-l7leveler-kubr-yp-vla-71-BAL\u0026unique_key=8268626693327871969",
          "type": "image",
          "status_code": 200,
          "size_bytes": 43
        },
        {
          "url": "https://ya.ru/captcha_smart_error.c1f6a7cf8d410e04e643.min.js?k=1770297661809",
          "type": "script",
          "status_code": 200,
          "size_bytes": 39003
        },
        {
          "url": "https://ya.ru/captcha_smart_react.min.js?k=1770297661809",
          "type": "script",
          "status_code": 200,
          "size_bytes": 166783
        },
        {
          "url": "https://ya.ru/captcha_smart.c1f6a7cf8d410e04e643.js?k=1770297661809",
          "type": "script",
          "status_code": 200,
          "size_bytes": 628803
        },
        {
          "url": "https://ya.ru/captcha_smart.c1f6a7cf8d410e04e643.min.css?k=1770297661809",
          "type": "style",
          "status_code": 200,
          "size_bytes": 90985
        }
      ],
      "discovered_at": "2026-03-02T19:26:22Z"
    }
  ]
}
```

## Example JSON report

```json
{
  "root_url": "https://example.com",
  "depth": 1,
  "generated_at": "2024-06-01T12:34:56Z",
  "pages": [
    {
      "url": "https://example.com",
      "depth": 0,
      "http_status": 200,
      "status": "ok",
      "error": "",
      "seo": {
        "has_title": true,
        "title": "Example title",
        "has_description": true,
        "description": "Example description",
        "has_h1": true
      },
      "broken_links": [
        {
          "url": "https://example.com/missing",
          "status_code": 404,
          "error": "Not Found"
        }
      ],
      "assets": [
        {
          "url": "https://example.com/static/logo.png",
          "type": "image",
          "status_code": 200,
          "size_bytes": 12345,
          "error": ""
        }
      ],
      "discovered_at": "2024-06-01T12:34:56Z"
    }
  ]
}
```

## Field meaning (short)

- `root_url`: start URL
- `depth`: maximum crawl depth (levels; `1` means only root page)
- `generated_at`: report generation time (ISO8601)
- `pages[]`: list of crawled pages (each URL appears once)
- `pages[].seo`: basic SEO fields (`title`, `meta description`, `h1`)
- `pages[].broken_links`: only broken links (HTTP 4xx/5xx or network error)
- `pages[].assets`: assets used on the page (image/script/style) with size and errors
- `pages[].discovered_at`: time when page was processed (ISO8601)
