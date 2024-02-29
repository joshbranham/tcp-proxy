package tcpproxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
)

type echoServer struct {
	listenAddr string
	listener   net.Listener
	shutdownC  chan struct{}
}

func newEchoServer(listenAddr string) *echoServer {
	return &echoServer{
		listenAddr: listenAddr,
		shutdownC:  make(chan struct{}),
	}
}

func (e *echoServer) listen() error {
	var err error
	e.listener, err = net.Listen("tcp", e.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener, err: %w", err)
	}
	return nil
}

func (e *echoServer) serve() error {
	wg := sync.WaitGroup{}
	for {
		select {
		case <-e.shutdownC:
			wg.Wait()
			return nil
		default:
			conn, err := e.listener.Accept()
			if err != nil {
				continue
			}
			wg.Add(1)
			go func() {
				e.handleConnection(conn)
				wg.Done()
			}()
		}
	}
}

func (e *echoServer) close() error {
	close(e.shutdownC)
	return e.listener.Close()
}

func (e *echoServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		bytes, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			if err != io.EOF {
				fmt.Println("failed to read data, err:", err)
			}
			return
		}

		_, _ = conn.Write(bytes)
	}
}
