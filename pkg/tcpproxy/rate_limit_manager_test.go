package tcpproxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimitManager(t *testing.T) {
	rlm := NewRateLimitManager(10, 1*time.Millisecond)

	user1 := rlm.RateLimiterFor("user1")
	assert.Equal(t, user1, rlm.RateLimiterFor("user1"))

	// Ensure we clean up goroutines for the manager and any child RateLimiters
	assert.NoError(t, rlm.Close())
}
