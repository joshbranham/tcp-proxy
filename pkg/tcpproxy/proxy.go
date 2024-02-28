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
	Config *Config

	listener    net.Listener
	wg          *sync.WaitGroup
	shutdownChn chan struct{}
}

func New(conf *Config) *Proxy {
	return &Proxy{
		Config: conf,

		wg:          &sync.WaitGroup{},
		shutdownChn: make(chan struct{}),
	}
}

// Listen starts a TCP listener on the configured ListenAddr, spawning goroutines to handle connections.
func (p *Proxy) Listen() error {
	var err error
	if p.listener, err = net.Listen("tcp", p.Config.ListenerConfig.ListenerAddr); err != nil {
		p.Config.Logger.Error("error listening", slog.String("error", err.Error()))
		return err
	}
	p.Config.Logger.Info(
		"proxy ready",
		slog.String("listening", p.listener.Addr().String()),
		slog.String("targets", strings.Join(p.Config.UpstreamConfig.Targets, ",")),
	)

	p.wg.Add(1)
	go p.acceptConnections()

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
		p.Config.Logger.Warn("timed out waiting for connections to finish")
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
			p.wg.Add(1)
			go p.handleConnection(conn)
		}
	}
}

// TODO: Authorization
func (p *Proxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()
	defer p.wg.Done()

	// Fetch a target based on our load balancing strategy. Ensure to clean up when we are done with the upstream.
	targetUpstream := p.Config.Balancer.FetchTarget()
	defer p.Config.Balancer.ReleaseTarget(targetUpstream)

	targetConn, err := net.DialTimeout("tcp", targetUpstream, p.Config.Timeout)
	if err != nil {
		p.Config.Logger.Error("connecting to target", slog.String("error", err.Error()))
		return
	}

	defer func(targetConn net.Conn) {
		err = targetConn.Close()
		if err != nil {
			p.Config.Logger.Error("closing connection", slog.String("error", err.Error()))
		}
	}(targetConn)

	// Copy data from the client to the target
	go p.copyData(targetConn, clientConn)

	// Copy data from target back to the client
	p.copyData(clientConn, targetConn)
}

func (p *Proxy) copyData(dst net.Conn, src net.Conn) {
	_, err := io.Copy(dst, src)
	if err != nil {
		p.Config.Logger.Error("copying data", slog.String("error", err.Error()))
	}
}
