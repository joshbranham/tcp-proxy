package tcpproxy

import (
	"errors"
	"sync"
)

// LoadBalancer is an interface that allows for implementing other LoadBalancing algorithms,
// such as RoundRobin.
type LoadBalancer interface {
	// FetchTarget returns an eligible upstream to receive a connection based on the given implementations selection criteria.
	FetchTarget() string
	// ReleaseTarget notifies the implementation that an upstream connection is complete.
	ReleaseTarget(string)
	// GetConnections fetches all upstreams and their active connection counts.
	GetConnections() map[string]int
}

type LeastConnection struct {
	activeConnections map[string]int
	mutex             sync.Mutex
	targets           []string
}

// NewLeastConnectionBalancer constructs an implementation of LoadBalancer, configured to favor
// upstreams with the least connections when opening new connections.
func NewLeastConnectionBalancer(targets []string) (*LeastConnection, error) {
	if len(targets) == 0 {
		return nil, errors.New("cannot initialize a LeastConnection Balancer with zero targets")
	}
	connectionsMap := make(map[string]int)
	for _, target := range targets {
		connectionsMap[target] = 0
	}

	return &LeastConnection{
		targets:           targets,
		activeConnections: connectionsMap,
		mutex:             sync.Mutex{},
	}, nil
}

// FetchTarget provides a target upstream, which in this case, is the one with the least amount of connections.
// In addition, this function acts as a "checkout" of an upstream, and the caller should clean up when done with the
// upstream connection by calling ReleaseTarget().
func (l *LeastConnection) FetchTarget() string {
	l.mutex.Lock()
	target := l.leastActiveUpstream()
	l.activeConnections[target] += 1
	l.mutex.Unlock()

	return target
}

func (l *LeastConnection) ReleaseTarget(target string) {
	l.mutex.Lock()

	if l.activeConnections[target] > 0 {
		l.activeConnections[target] -= 1
	}

	l.mutex.Unlock()
}

func (l *LeastConnection) GetConnections() map[string]int {
	l.mutex.Lock()
	activeConnections := l.activeConnections
	l.mutex.Unlock()
	return activeConnections
}

// leastActiveUpstream will iterate upstreams until it finds one with either 0 or the least amount
// of connections. This is a naive implementation that could be improved if performance was a concern.
func (l *LeastConnection) leastActiveUpstream() string {
	var leastActiveUpstream string
	for target, connectionCount := range l.activeConnections {
		// If we find any target with zero connections, return early with that as an eligible target.
		if connectionCount == 0 {
			leastActiveUpstream = target
			break
		}

		// Initialize our initial eligible upstream if unset
		if leastActiveUpstream == "" {
			leastActiveUpstream = target
		}

		if l.activeConnections[leastActiveUpstream] > connectionCount {
			leastActiveUpstream = target
		}
	}

	return leastActiveUpstream
}
