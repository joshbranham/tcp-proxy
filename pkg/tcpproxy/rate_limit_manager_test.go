package tcpproxy

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimitManager(t *testing.T) {
	rlm := NewRateLimitManager(10, 1*time.Millisecond, slog.Default())

	user1 := rlm.RateLimiterFor("user1")
	assert.Equal(t, user1, rlm.RateLimiterFor("user1"))

	// Ensure we clean up goroutines for the manager and any child RateLimiters
	rlm.Close()
}
