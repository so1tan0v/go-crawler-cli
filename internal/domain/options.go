package domain

import (
	"net/http"
	"time"
)

/*Параметры анализа сайта*/
type Options struct {
	URL         string
	Depth       int
	Retries     int
	Delay       time.Duration
	RPS         int
	Timeout     time.Duration
	UserAgent   string
	Concurrency int
	IndentJSON  bool
	HTTPClient  *http.Client
}
