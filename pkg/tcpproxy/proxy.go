package tcpproxy

import (
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

const ConnectionCloseTimeout = time.Second

type Proxy struct {
	Balancer LoadBalancer
	Config   *Config

	logger        *slog.Logger
	listener      net.Listener
	wg            *sync.WaitGroup
	shutdownChn   chan struct{}
	connectionChn chan net.Conn
}

func New(conf *Config, balancer LoadBalancer, logger *slog.Logger) *Proxy {
	return &Proxy{
		Balancer: balancer,
		Config:   conf,

		logger:        logger,
		wg:            &sync.WaitGroup{},
		shutdownChn:   make(chan struct{}),
		connectionChn: make(chan net.Conn),
	}
}

// Listen starts a tcp listener on the configured ListenAddr, spawning goroutines to handle connections.
func (p *Proxy) Listen() error {
	defer p.wg.Done()

	var err error
	if p.listener, err = net.Listen("tcp", p.Config.ListenerConfig.ListenerAddr); err != nil {
		p.logger.Error("error listening", slog.String("error", err.Error()))
		return err
	}
	p.logger.Info(
		"proxy ready",
		slog.String("listening", p.listener.Addr().String()),
		slog.String("targets", strings.Join(p.Config.UpstreamConfig.Targets, ",")),
	)

	p.wg.Add(2)
	go p.acceptConnections()
	go p.handleConnections()

	return nil
}

// Close will clean up connections and close the listener.
func (p *Proxy) Close() error {
	close(p.shutdownChn)
	err := p.listener.Close()
	if err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(ConnectionCloseTimeout):
		p.logger.Warn("timed out waiting for connections to finish")
		return nil
	}
}

func (p *Proxy) acceptConnections() {
	for {
		select {
		case <-p.shutdownChn:
			return
		default:
			conn, err := p.listener.Accept()
			if err != nil {
				continue
			}
			p.connectionChn <- conn
		}
	}
}

func (p *Proxy) handleConnections() {
	for {
		select {
		case <-p.shutdownChn:
			return
		case conn := <-p.connectionChn:
			p.wg.Add(1)
			go p.handleConnection(conn)
		}
	}
}

// TODO: Authorization
func (p *Proxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()
	defer p.wg.Done()

	// Fetch a target based on our load balancing strategy. Ensure to clea nup when we are done with the upstream.
	targetUpstream := p.Balancer.FetchTarget()
	defer p.Balancer.ReleaseTarget(targetUpstream)

	targetConn, err := net.DialTimeout("tcp", targetUpstream, p.Config.Timeout)
	if err != nil {
		p.logger.Error("connecting to target", slog.String("error", err.Error()))
		return
	}

	defer func(targetConn net.Conn) {
		err = targetConn.Close()
		if err != nil {
			p.logger.Error("closing connection", slog.String("error", err.Error()))
		}
	}(targetConn)

	// Copy data from the client to the target
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		if err != nil {
			p.logger.Error("copying from client to target", slog.String("error", err.Error()))
		}
	}()

	// Copy data from target back to the client
	_, err = io.Copy(clientConn, targetConn)
	if err != nil {
		p.logger.Error("copying from target to client", slog.String("error", err.Error()))
	}
}
