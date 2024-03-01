package tcpproxy

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// TokenFillRate is the amount of tokens added to the bucket at the given FillRate.
const TokenFillRate = 1

// RateLimiter is an instance of rate limiting configuration, used for a single client.
type RateLimiter struct {
	capacity int64
	fillRate time.Duration
	tokens   atomic.Int64
	closed   atomic.Bool

	shutdownC chan struct{}
	wg        sync.WaitGroup
}

// NewRateLimiter returns a RateLimiter and spawns a goroutine to add tokens up until the capacity.
// Use Close() to cleanup the goroutine and stock token accumulation.
func NewRateLimiter(capacity int64, fillRate time.Duration) *RateLimiter {
	rl := &RateLimiter{
		capacity: capacity,
		fillRate: fillRate,

		shutdownC: make(chan struct{}),
	}
	rl.tokens.Add(capacity)

	rl.wg.Add(1)
	go rl.fillTokens()

	return rl
}

// Close stops the accumulation of tokens for the RateLimiter.
func (r *RateLimiter) Close() error {
	if r.closed.Load() {
		return errors.New("RateLimiter is already closed")
	} else {
		close(r.shutdownC)
		r.closed.Store(true)
		r.wg.Wait()
	}

	return nil
}

// ConnectionAllowed validates the RateLimiter isn't at the limit. If allowed, this returns true and decrements
// the tokens by 1. If false, it returns false and leaves the tokens as is.
func (r *RateLimiter) ConnectionAllowed() bool {
	if r.tokens.Load() > 0 {
		r.tokens.Add(-1)
		return true
	}

	return false
}

func (r *RateLimiter) fillTokens() {
	ticker := time.NewTicker(r.fillRate)
	for {
		select {
		case <-r.shutdownC:
			ticker.Stop()
			r.wg.Done()
			return
		case <-ticker.C:
			tokens := r.tokens.Load()
			if tokens != 0 && tokens < r.capacity {
				r.tokens.Add(TokenFillRate)
			}
		}
	}
}
