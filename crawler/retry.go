package crawler

import (
	"context"
	"net/http"
	"time"
)

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func retryPause(ctx context.Context) error {
	timer := time.NewTimer(defaultRetryPause)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
