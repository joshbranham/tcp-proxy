package tcpproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LeastConnectionLoadBalancer(t *testing.T) {
	balancer, err := NewLeastConnectionBalancer([]string{":5000", ":5001"})
	require.NoError(t, err)

	// Mark some targets as being connected, then cleanup
	target1 := balancer.FetchUpstream()
	target2 := balancer.FetchUpstream()
	assert.NotEqual(t, target1.Address, target2.Address)

	// Assert for each target, we have one connection at this time
	for _, upstream := range balancer.FetchUpstreams() {
		assert.Equal(t, 1, upstream.Connections())
	}

	// Cleanup and assert zero connections
	target1.Release()
	target2.Release()
	for _, upstream := range balancer.FetchUpstreams() {
		assert.Equal(t, 0, upstream.Connections())
	}
}

func Test_UpstreamConnectionsCannotBeNegative(t *testing.T) {
	balancer, err := NewLeastConnectionBalancer([]string{":5000"})
	require.NoError(t, err)

	// Release an upstream twice, ensure we can't have a negative counter
	target := balancer.FetchUpstream()
	target.Release()
	target.Release()
	assert.Equal(t, 0, target.Connections())
}
