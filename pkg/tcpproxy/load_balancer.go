package tcpproxy

import (
	"errors"
	"sync"
)

// LeastConnectionBalancer is a load balancer implementation configured to favor
// upstreams with the least amount of connections when opening new connections.
type LeastConnectionBalancer struct {
	upstreams []*Upstream
}

// NewLeastConnectionBalancer constructs a configured LeastConnectionBalancer.
func NewLeastConnectionBalancer(targets []string) (*LeastConnectionBalancer, error) {
	if len(targets) == 0 {
		return nil, errors.New("cannot initialize a LeastConnectionBalancer LoadBalancer with zero upstreams")
	}
	var upstreams []*Upstream
	for _, target := range targets {
		upstreams = append(upstreams, &Upstream{Address: target})
	}

	return &LeastConnectionBalancer{upstreams: upstreams}, nil
}

// FetchUpstream provides a target Upstream with the least amount of connections.
func (l *LeastConnectionBalancer) FetchUpstream() *Upstream {
	upstream := l.leastActiveUpstream()
	upstream.mutex.Lock()
	upstream.connections += 1
	upstream.mutex.Unlock()

	return l.leastActiveUpstream()
}

// FetchUpstreams returns all upstreams the LeastConnectionBalancer is configured with.
func (l *LeastConnectionBalancer) FetchUpstreams() []*Upstream {
	return l.upstreams
}

// Upstream is a wrapper around an upstream Address that connections can use. Callers should use upstream.Release()
// when finished with a connection.
type Upstream struct {
	// Address is the address of the upstream, for example, 172.27.0.1:5000
	Address string

	connections int
	mutex       sync.RWMutex
}

// Release will decrement the count of current connections, used when a proxied request is complete.
func (u *Upstream) Release() {
	u.mutex.Lock()

	if u.connections > 0 {
		u.connections -= 1
	}

	u.mutex.Unlock()
}

// Connections will return the count of current connections.
func (u *Upstream) Connections() int {
	var connections int
	u.mutex.RLock()
	connections = u.connections
	u.mutex.RUnlock()

	return connections
}

// leastActiveUpstream will iterate upstreams until it finds one with either 0 or the least amount
// of connections. This is a naive implementation that could be improved if performance was a concern.
func (l *LeastConnectionBalancer) leastActiveUpstream() *Upstream {
	leastActiveUpstream := l.upstreams[0]
	leastActiveConnections := -1
	for _, upstream := range l.upstreams {
		upstream.mutex.RLock()

		if (upstream.connections < leastActiveConnections) || leastActiveConnections == -1 {
			leastActiveConnections = upstream.connections
			leastActiveUpstream = upstream
		}

		upstream.mutex.RUnlock()
	}

	return leastActiveUpstream
}
