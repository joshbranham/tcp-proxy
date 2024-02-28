package tcpproxy

import (
	"errors"
	"sync"
)

// LeastConnectionBalancer is a load balancer implementation configured to favor
// upstreams with the least amount of connections when opening new connections.
type LeastConnectionBalancer struct {
	activeConnections map[string]int
	mutex             sync.RWMutex
	targets           []string
}

// NewLeastConnectionBalancer constructs a configured LeastConnectionBalancer.
func NewLeastConnectionBalancer(targets []string) (*LeastConnectionBalancer, error) {
	if len(targets) == 0 {
		return nil, errors.New("cannot initialize a LeastConnectionBalancer LoadBalancer with zero targets")
	}
	connectionsMap := make(map[string]int)
	for _, target := range targets {
		connectionsMap[target] = 0
	}

	return &LeastConnectionBalancer{
		targets:           targets,
		activeConnections: connectionsMap,
	}, nil
}

// FetchTarget provides a target upstream, which in this case, is the one with the least amount of connections.
// In addition, this function acts as a "checkout" of an upstream, and the caller should clean up when done with the
// upstream connection by calling ReleaseTarget().
func (l *LeastConnectionBalancer) FetchTarget() string {
	l.mutex.RLock()
	target := l.leastActiveUpstream()
	l.activeConnections[target] += 1
	l.mutex.RUnlock()

	return target
}

func (l *LeastConnectionBalancer) ReleaseTarget(target string) {
	l.mutex.Lock()

	if l.activeConnections[target] > 0 {
		l.activeConnections[target] -= 1
	}

	l.mutex.Unlock()
}

func (l *LeastConnectionBalancer) GetConnections() map[string]int {
	l.mutex.RLock()
	activeConnections := l.activeConnections
	l.mutex.RUnlock()
	return activeConnections
}

// leastActiveUpstream will iterate upstreams until it finds one with either 0 or the least amount
// of connections. This is a naive implementation that could be improved if performance was a concern.
func (l *LeastConnectionBalancer) leastActiveUpstream() string {
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
			continue
		}

		if l.activeConnections[leastActiveUpstream] > connectionCount {
			leastActiveUpstream = target
		}
	}

	return leastActiveUpstream
}
