package crawler

import (
	"context"
	"sync"
	"time"
)

type rateLimiter struct {
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func newRateLimiter(interval time.Duration) *rateLimiter {
	if interval <= 0 {
		return &rateLimiter{interval: 0}
	}

	return &rateLimiter{interval: interval}
}

func (l *rateLimiter) Wait(ctx context.Context) error {
	if l.interval <= 0 {
		return nil
	}

	now := time.Now()

	l.mu.Lock()
	scheduled := now
	if !l.next.IsZero() && l.next.After(scheduled) {
		scheduled = l.next
	}

	l.next = scheduled.Add(l.interval)
	l.mu.Unlock()

	wait := scheduled.Sub(now)
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
