package tcpproxy

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// ConnectionCloseTimeout is how long the proxy will wait for active connections to close before exiting when
// it receives a signal.
const ConnectionCloseTimeout = time.Second

// DialTimeout is how long the proxy will wait when connecting to an upstream before giving up.
const DialTimeout = 5 * time.Second

// Proxy is an instance of the TCP proxy. Use New() with a Config to construct a proper Proxy.
type Proxy struct {
	loadBalancer   *LeastConnectionBalancer
	listenerConfig *ListenerConfig
	logger         *slog.Logger
	idletimeout    time.Duration
	upstreamConfig *UpstreamConfig

	listener  net.Listener
	shutdownC chan struct{}
}

// New constructs a Proxy for a given Config, validating the Config beforehand.
func New(conf *Config) (*Proxy, error) {
	err := conf.Validate()
	if err != nil {
		return nil, err
	}

	return &Proxy{
		loadBalancer:   conf.LoadBalancer,
		listenerConfig: conf.ListenerConfig,
		logger:         conf.Logger,
		idletimeout:    conf.IdleTimeout,
		upstreamConfig: conf.UpstreamConfig,

		shutdownC: make(chan struct{}),
	}, nil
}

// Listen starts a TCP listener on the configured ListenAddr. Use Serve() to begin accepting connections.
func (p *Proxy) Listen() error {
	if p.listener != nil {
		return fmt.Errorf("attempted to call Listen when the proxy is already listening")
	}
	var err error
	if p.listener, err = net.Listen("tcp", p.listenerConfig.ListenerAddr); err != nil {
		p.logger.Error("error listening", slog.String("error", err.Error()))
		return err
	}
	p.logger.Info(
		"proxy ready",
		slog.String("listening", p.listener.Addr().String()),
		slog.String("targets", strings.Join(p.upstreamConfig.Targets, ",")),
	)
	return err
}

// Serve blocks and starts receiving connections on our listener, spawning goroutines to handle individual connections.
func (p *Proxy) Serve() error {
	if p.listener == nil {
		return fmt.Errorf("cannot serve requests before calling Listen()")
	}
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
			wg.Add(1)
			go p.handleConnection(conn)
		}
	}
}

// Address returns full address and port the proxy is serving on. Eg: 127.0.0.1:5000
func (p *Proxy) Address() string {
	return p.listener.Addr().String()
}

// Close will clean up connections and close the listener, if it is listening.
func (p *Proxy) Close() error {
	if p.listener == nil {
		return fmt.Errorf("cannot close proxy, not currently listening")
	}
	close(p.shutdownC)
	err := p.listener.Close()
	if err != nil {
		return err
	}

	done := make(chan struct{})
	close(done)
	select {
	case <-done:
		return nil
	case <-time.After(ConnectionCloseTimeout):
		p.logger.Warn("timed out waiting for connections to finish")
		return nil
	}
}

// TODO: Authorization
func (p *Proxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Fetch a target based on our load balancing strategy. Ensure to clean up when we are done with the upstream.
	upstream := p.loadBalancer.FetchUpstream()
	defer upstream.Release()

	targetConn, err := net.DialTimeout("tcp", upstream.Address, DialTimeout)
	if err != nil {
		p.logger.Error("connecting to target", "error", err)
		return
	}

	// Set the Read and Write Deadline for each connection to the configured IdleTimeout
	err = clientConn.SetDeadline(time.Now().Add(p.idletimeout))
	if err != nil {
		p.logger.Error("unable to extend deadline for connection")
	}
	err = targetConn.SetDeadline(time.Now().Add(p.idletimeout))
	if err != nil {
		p.logger.Error("unable to extend deadline for connection")
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
	}()

	wg.Wait()
}

func (p *Proxy) copyData(dst net.Conn, src net.Conn) {
	_, err := io.Copy(dst, src)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			p.logger.Error("idle timeout exceeded", "error", err)
		}
		p.logger.Error("copying data", "error", err)
	}
}
