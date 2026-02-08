package domain

/*
Страница сайта

JSON:

	{
	  "url": "https://example.com",
	  "depth": 0,
	  "http_status": 200,
	  "status": "ok",
	  "error": ""
	}
*/
type Page struct {
	URL        string `json:"url"`
	Depth      int    `json:"depth"`
	HTTPStatus int    `json:"http_status"`
	Status     string `json:"status"`
	Error      string `json:"error"`
}
