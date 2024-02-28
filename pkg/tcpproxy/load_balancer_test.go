package tcpproxy

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_LeastConnectionLoadBalancer(t *testing.T) {
	balancer, err := NewLeastConnectionBalancer([]string{":5000", ":5001"})
	require.NoError(t, err)

	// Mark some targets as being connected, then cleanup
	target1 := balancer.FetchTarget()
	target2 := balancer.FetchTarget()
	assert.NotEqual(t, target1, target2)

	// Assert for each target, we have one connection at this time
	for _, connections := range balancer.GetConnections() {
		assert.Equal(t, 1, connections)
	}

	// Cleanup and assert zero connections
	balancer.ReleaseTarget(target1)
	balancer.ReleaseTarget(target2)
	for _, connections := range balancer.GetConnections() {
		assert.Equal(t, 0, connections)
	}
}
