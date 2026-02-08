package domain

import "time"

/*
Результат анализа сайта.

{
  "root_url": "https://example.com",
  "depth": 1,
  "generated_at": "2024-05-18T12:34:56Z",
  "pages": [
    {
      "url": "https://example.com",
      "depth": 0,
      "http_status": 200,
      "status": "ok",
      "error": ""
    }
  ]
}
*/
type AnalyzeResult struct {
	RootURL     string    `json:"root_url"`
	Depth       int       `json:"depth"`
	GeneratedAt time.Time `json:"generated_at"`
	Pages       []Page    `json:"pages"`
}
