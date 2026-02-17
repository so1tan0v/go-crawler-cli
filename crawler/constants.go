package crawler

import "time"

const (
	defaultTimeout    = 15 * time.Second
	defaultHTMLLimit  = 2 << 20  // 2MB
	defaultAssetLimit = 10 << 20 // 10MB

	defaultRetryPause = 10 * time.Millisecond
)
