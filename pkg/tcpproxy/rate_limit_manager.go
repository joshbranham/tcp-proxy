package tcpproxy

import (
	"log/slog"
	"sync"
	"time"
)

// RateLimitManager wraps many RateLimiters and provides mechanisms for getting per-client RateLimiters.
type RateLimitManager struct {
	defaultCapacity int
	defaultFillRate time.Duration
	logger          *slog.Logger
	rateLimiters    map[string]*RateLimiter
	mutex           sync.RWMutex
}

// NewRateLimitManager returns a configured RateLimitManager.
func NewRateLimitManager(capacity int, fillRate time.Duration, logger *slog.Logger) *RateLimitManager {
	return &RateLimitManager{
		defaultCapacity: capacity,
		defaultFillRate: fillRate,
		logger:          logger,
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

// Close calls Close() on all known RateLimiters. RateLimiters can only be closed once, however this
// func will handle if a RateLimiter is already closed.
func (r *RateLimitManager) Close() {
	r.mutex.RLock()
	for _, rateLimiter := range r.rateLimiters {
		if rateLimiter != nil {
			if err := rateLimiter.Close(); err != nil {
				r.logger.Warn("error closing rate limiter", "error", err)
			}
		}
	}
	r.mutex.RUnlock()
}
