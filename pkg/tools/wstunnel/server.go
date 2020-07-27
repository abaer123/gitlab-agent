package wstunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type ListenerWrapper struct {
	AcceptOptions websocket.AcceptOptions
	// ReadLimit. Optional. See websocket.Conn.SetReadLimit().
	ReadLimit int64

	// Fields below are directly passed to the constructed http.Server.
	// All of them are optional.

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ConnState         func(net.Conn, http.ConnState)
	ErrorLog          *log.Logger
	BaseContext       func(net.Listener) context.Context
	ConnContext       func(ctx context.Context, c net.Conn) context.Context
}

type Listener interface {
	net.Listener
	// Shutdown gracefully shuts down the server without interrupting any
	// active connections. See http.Server.Shutdown().
	Shutdown(context.Context) error
}

// Wrap accepts WebSocket connections and turns them into TCP connections.
func (w *ListenerWrapper) Wrap(source net.Listener) Listener {
	accepted := make(chan net.Conn)
	ctx, cancel := context.WithCancel(context.Background())
	s := &wrapperServer{
		cancelAccept: cancel,
		source: &onceCloseListener{
			Listener: source,
		},
		accepted:  accepted,
		serverErr: make(chan error, 1),
	}
	options := w.AcceptOptions
	options.Subprotocols = []string{TunnelWebSocketProtocol}
	s.server = &http.Server{
		Handler: &HttpHandler{
			Ctx:           ctx,
			AcceptOptions: options,
			Sink:          accepted,
			ReadLimit:     w.ReadLimit,
		},
		ReadTimeout:       w.ReadTimeout,
		ReadHeaderTimeout: w.ReadHeaderTimeout,
		WriteTimeout:      w.WriteTimeout,
		IdleTimeout:       w.IdleTimeout,
		MaxHeaderBytes:    w.MaxHeaderBytes,
		ConnState:         w.ConnState,
		ErrorLog:          w.ErrorLog,
		BaseContext:       w.BaseContext,
		ConnContext:       w.ConnContext,
	}
	go s.run()

	return s
}

type wrapperServer struct {
	cancelAccept context.CancelFunc
	source       net.Listener
	server       *http.Server
	accepted     <-chan net.Conn
	serverErr    chan error
}

func (s *wrapperServer) run() {
	defer s.cancelAccept()
	s.serverErr <- s.server.Serve(s.source)
}

func (s *wrapperServer) Accept() (net.Conn, error) {
	select {
	case con := <-s.accepted:
		return con, nil
	case err := <-s.serverErr:
		s.serverErr <- err // put it back for the next Accept call
		return nil, err
	}
}

func (s *wrapperServer) Close() error {
	return s.source.Close()
}

func (s *wrapperServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *wrapperServer) Addr() net.Addr {
	return s.source.Addr()
}

type HttpHandler struct {
	Ctx           context.Context
	AcceptOptions websocket.AcceptOptions
	Sink          chan<- net.Conn
	// ReadLimit. Optional. See websocket.Conn.SetReadLimit().
	ReadLimit int64
}

func (h *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &h.AcceptOptions)
	if err != nil {
		return
	}
	subprotocol := conn.Subprotocol()
	if subprotocol != TunnelWebSocketProtocol {
		conn.Close(websocket.StatusProtocolError, fmt.Sprintf("Expecting %q subprotocol, got %q", TunnelWebSocketProtocol, subprotocol)) // nolint: errcheck, gosec
		return
	}
	if h.ReadLimit != 0 {
		conn.SetReadLimit(h.ReadLimit)
	}
	netConn := websocket.NetConn(context.Background(), conn, websocket.MessageBinary)

	select {
	case <-r.Context().Done():
		netConn.Close() // nolint: errcheck, gosec
	case <-h.Ctx.Done():
		// send correct close frame
		conn.Close(websocket.StatusGoingAway, "Shutting down") // nolint: errcheck, gosec
		// free resources
		netConn.Close() // nolint: errcheck, gosec
	case h.Sink <- netConn:
	}
}

// onceCloseListener wraps a net.Listener, protecting it from
// multiple Close calls.
// We get two Close calls:
// - server.Serve(s.source) closes the listener before returning
// - Close() method
type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() {
	oc.closeErr = oc.Listener.Close()
}
