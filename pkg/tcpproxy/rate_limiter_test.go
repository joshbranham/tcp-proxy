package tcpproxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_ConnectionAllowed(t *testing.T) {
	// TODO: Set a FillRate that exceeds test runtime. In the future, we could use
	// an interface and mock add/remove tokens from the bucket without the ticker.
	rl := NewRateLimiter(1, 1*time.Minute)

	// Allowed immediately at creation with 1 token
	assert.Equal(t, true, rl.ConnectionAllowed())

	// This brings capacity to zero, so connection not allowed
	assert.Equal(t, false, rl.ConnectionAllowed())

	// Ensure we can't double close a RateLimiter
	require.NoError(t, rl.Close())
	assert.Error(t, rl.Close())
}
