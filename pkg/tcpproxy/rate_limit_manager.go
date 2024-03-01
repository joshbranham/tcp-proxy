package tcpproxy

import (
	"sync"
	"time"
)

// RateLimitManager wraps many RateLimiters and provides mechanisms for getting per-client RateLimiters.
type RateLimitManager struct {
	defaultCapacity int
	defaultFillRate time.Duration
	rateLimiters    map[string]*RateLimiter
	mutex           sync.RWMutex
}

// NewRateLimitManager returns a configured RateLimitManager.
func NewRateLimitManager(capacity int, fillRate time.Duration) *RateLimitManager {
	return &RateLimitManager{
		defaultCapacity: capacity,
		defaultFillRate: fillRate,
		rateLimiters:    make(map[string]*RateLimiter),
		mutex:           sync.RWMutex{},
	}
}

// RateLimiterFor returns, or creates, a RateLimiter for a given client string.
func (r *RateLimitManager) RateLimiterFor(client string) *RateLimiter {
	var rateLimiter *RateLimiter

	r.mutex.Lock()
	if r.rateLimiters[client] == nil {
		rateLimiter = NewRateLimiter(int64(r.defaultCapacity), r.defaultFillRate)
		r.rateLimiters[client] = rateLimiter
	} else {
		rateLimiter = r.rateLimiters[client]
	}
	r.mutex.Unlock()

	return rateLimiter
}

// Close calls Close() on all known RateLimiters. Calling this multiple times will error if a RateLimiter
// has already been closed.
func (r *RateLimitManager) Close() error {
	r.mutex.RLock()
	for _, rateLimiter := range r.rateLimiters {
		if err := rateLimiter.Close(); err != nil {
			return err
		}
	}
	r.mutex.RUnlock()

	return nil
}
