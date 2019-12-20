// +build !js,!wasm

package wasmws

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"nhooyr.io/websocket"
)

type WebSockListener struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	acceptCh chan net.Conn
}

var _ net.Listener = (*WebSockListener)(nil)

func NewWebSocketListener(ctx context.Context) *WebSockListener {
	ctx, cancel := context.WithCancel(ctx)
	return &WebSockListener{
		ctx:       ctx,
		ctxCancel: cancel,
		acceptCh:  make(chan net.Conn, 8),
	}
}

func (wsl *WebSockListener) HTTPAccept(wtr http.ResponseWriter, req *http.Request) {
	ws, err := websocket.Accept(wtr, req, nil)
	if err != nil {
		log.Fatalf("Could not accept websocket from %q; Details: %s", req.RemoteAddr, err)
	}

	conn := websocket.NetConn(wsl.ctx, ws, websocket.MessageBinary)
	select {
	case wsl.acceptCh <- conn:
	case <-req.Context().Done():
		ws.Close(websocket.StatusBadGateway, fmt.Sprintf("Failed to accept connection; Details: %s", req.Context().Err()))
	}
}

func (wsl *WebSockListener) Accept() (net.Conn, error) {
	select {
	case conn := <-wsl.acceptCh:
		return conn, nil
	case <-wsl.ctx.Done():
		return nil, fmt.Errorf("Listener closed; Details: %w", wsl.ctx.Err())
	}
}

func (wsl *WebSockListener) Close() error {
	wsl.ctxCancel()
	return nil
}

func (wsl *WebSockListener) Addr() net.Addr {
	return wsAddr{}
}

type wsAddr struct{}

func (wsAddr) Network() string { return "websocket" }

func (wsAddr) String() string { return "websocket" }
