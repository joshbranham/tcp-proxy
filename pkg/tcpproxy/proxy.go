package tcpproxy

import (
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DialTimeout is how long the proxy will wait when connecting to an upstream before giving up.
const DialTimeout = 5 * time.Second

// Proxy is an instance of the TCP proxy. Use New() with a Config to construct a proper Proxy.
type Proxy struct {
	loadBalancer   *LeastConnectionBalancer
	listenerConfig *ListenerConfig
	logger         *slog.Logger
	upstreamConfig *UpstreamConfig

	listener  net.Listener
	shutdownC chan struct{}

	serving atomic.Bool
}

// New constructs a Proxy for a given Config. It will validate the configuration, and if valid, begin listening
// on the configured ListenAddr.
func New(conf *Config) (*Proxy, error) {
	err := conf.Validate()
	if err != nil {
		return nil, err
	}

	proxy := &Proxy{
		loadBalancer:   conf.LoadBalancer,
		listenerConfig: conf.ListenerConfig,
		logger:         conf.Logger,
		upstreamConfig: conf.UpstreamConfig,

		shutdownC: make(chan struct{}),
	}

	tlsConfig, err := conf.TLSConfig()
	if err != nil {
		proxy.logger.Error("failure loading TLS configuration", "error", err)
		return nil, err
	}

	if proxy.listener, err = tls.Listen("tcp", proxy.listenerConfig.ListenerAddr, tlsConfig); err != nil {
		proxy.logger.Error("error listening", slog.String("error", err.Error()))
		return nil, err
	}
	proxy.logger.Info(
		"proxy ready",
		slog.String("listening", proxy.listener.Addr().String()),
		slog.String("targets", strings.Join(proxy.upstreamConfig.Targets, ",")),
	)

	return proxy, nil
}

// Serve blocks and starts receiving connections on our listener, spawning goroutines to handle individual connections.
func (p *Proxy) Serve() error {
	// Ensure we cannot call Serve more than once.
	if p.serving.Load() {
		return errors.New("cannot Serve as proxy is already serving")
	}
	p.serving.Store(true)

	wg := &sync.WaitGroup{}
	for {
		select {
		case <-p.shutdownC:
			wg.Wait()
			return nil
		default:
			conn, err := p.listener.Accept()
			if err != nil {
				continue
			}

			// Ensure we have a TLS connection, if not close it and continue.
			tlsConn, ok := conn.(*tls.Conn)
			if !ok {
				p.logger.Warn("non TLS connection detected")
				_ = conn.Close()
				continue
			}

			// Force a handshake so we can inspect x509 data. This would happen normally
			// when the first IO occurs, but we need to validate the user before accepting.
			if err = tlsConn.Handshake(); err != nil {
				p.logger.Warn("could not run handshake protocol for TLS connection, closing")
				_ = conn.Close()
				continue
			}

			// Check if the user is in the AuthorizedGroups, otherwise close the connection.
			if p.connectionAuthorized(tlsConn) {
				wg.Add(1)
				go func() {
					p.handleConnection(conn)
					wg.Done()
				}()
			} else {
				p.logger.Warn("user is not authorized to access upstream")
				_ = conn.Close()
			}
		}
	}
}

// Address returns full address and port the proxy is serving on. Eg: 127.0.0.1:5000
func (p *Proxy) Address() string {
	return p.listener.Addr().String()
}

// Close will clean up connections and close the listener, if it is listening.
func (p *Proxy) Close() error {
	// TODO: This prevents a panic if someone calls Close() twice on a Proxy instance. This is a hack,
	// in that you could in theory close and re-listen at the call site, however the API exposed here prefers
	// New() to return a new Proxy that is listening.
	if !p.serving.Load() {
		return errors.New("cannot close a proxy that is not serving")
	}
	close(p.shutdownC)

	err := p.listener.Close()
	if err != nil {
		return err
	}

	p.serving.Store(false)

	return nil
}

func (p *Proxy) handleConnection(clientConn net.Conn) {
	// Fetch a target based on our load balancing strategy. Ensure to clean up when we are done with the upstream.
	upstream := p.loadBalancer.FetchUpstream()
	defer upstream.Release()

	targetConn, err := net.DialTimeout("tcp", upstream.Address, DialTimeout)
	if err != nil {
		p.logger.Error("connecting to target", "error", err)
		return
	}

	defer func() {
		if err = targetConn.Close(); err != nil {
			p.logger.Error("closing connection", "error", err)
		}
	}()

	// Create a WaitGroup to handle nested goroutines that copy data
	wg := &sync.WaitGroup{}

	// Copy data from the client to the target
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.copyData(targetConn, clientConn)
	}()

	// Copy data from target back to the client
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.copyData(clientConn, targetConn)

		// For added safety, close the target connection once data transfer is complete to ensure the other
		// goroutine can't get stuck.
		p.closeConnection(targetConn)
	}()

	wg.Wait()

	// Close connections once EOF has been met and data transfer is complete
	p.closeConnection(clientConn)
	p.closeConnection(targetConn)
}

func (p *Proxy) copyData(dst net.Conn, src net.Conn) {
	_, err := io.Copy(dst, src)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			p.logger.Error("deadline exceeded", "error", err)
		}
		p.logger.Error("copying data", "error", err)
	}
}

func (p *Proxy) closeConnection(conn net.Conn) {
	if err := conn.Close(); err != nil {
		p.logger.Error("closing connection", "error", err)
	}
}

// connectionAuthorized will look for our authorization stored in a certificates CN, in the format "user@group",
// and extract that to verify the user is a member of the AuthorizedGroups configured.
func (p *Proxy) connectionAuthorized(conn *tls.Conn) bool {
	for _, cert := range conn.ConnectionState().PeerCertificates {
		s := strings.Split(cert.Subject.CommonName, "@")
		if len(s) != 2 {
			return false
		}
		group := s[1]
		if slices.Contains(p.upstreamConfig.AuthorizedGroups, group) {
			return true
		}
	}

	return false
}
