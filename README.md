### Hexlet tests and linter status:
[![Actions Status](https://github.com/so1tan0v/go-project-316/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/so1tan0v/go-project-316/actions)

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
