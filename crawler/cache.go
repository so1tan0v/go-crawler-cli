package crawler

import (
	"code/internal/domain"
	"context"
	"errors"
	"sync"
)

/*Кэш для проверки ссылок*/
type linkCheckCacheEntry struct {
	status int
	err    string
}

/*Кэш для проверки ссылок*/
type linkCheckCache struct {
	mu       sync.Mutex
	data     map[string]linkCheckCacheEntry
	inflight map[string]chan struct{}
}

/*Кэш для проверки ассетов*/
type assetCache struct {
	mu       sync.Mutex
	data     map[string]domain.Asset
	inflight map[string]chan struct{}
}

func newLinkCheckCache() *linkCheckCache {
	return &linkCheckCache{
		data:     make(map[string]linkCheckCacheEntry),
		inflight: make(map[string]chan struct{}),
	}
}

func (c *linkCheckCache) GetOrCompute(ctx context.Context, key string, fn func() (int, error)) (int, error) {
	for {
		c.mu.Lock()
		if v, ok := c.data[key]; ok {
			c.mu.Unlock()

			if v.err == "" {
				return v.status, nil
			}

			return v.status, errors.New(v.err)
		}

		if ch, ok := c.inflight[key]; ok {
			c.mu.Unlock()
			select {
			case <-ch:
				continue
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}

		ch := make(chan struct{})
		c.inflight[key] = ch
		c.mu.Unlock()

		status, err := fn()
		entry := linkCheckCacheEntry{status: status}
		if err != nil {
			entry.err = err.Error()
		}

		c.mu.Lock()
		c.data[key] = entry

		delete(c.inflight, key)
		close(ch)
		c.mu.Unlock()

		if err != nil {
			return status, err
		}

		return status, nil
	}
}

func newAssetCache() *assetCache {
	return &assetCache{
		data:     make(map[string]domain.Asset),
		inflight: make(map[string]chan struct{}),
	}
}

func (c *assetCache) GetOrCompute(ctx context.Context, key string, fn func() domain.Asset) (domain.Asset, error) {
	for {
		c.mu.Lock()
		if v, ok := c.data[key]; ok {
			c.mu.Unlock()
			return v, nil
		}
		if ch, ok := c.inflight[key]; ok {
			c.mu.Unlock()
			select {
			case <-ch:
				continue
			case <-ctx.Done():
				return domain.Asset{}, ctx.Err()
			}
		}

		ch := make(chan struct{})
		c.inflight[key] = ch
		c.mu.Unlock()

		value := fn()

		c.mu.Lock()
		c.data[key] = value

		delete(c.inflight, key)
		close(ch)

		c.mu.Unlock()

		return value, nil
	}
}
