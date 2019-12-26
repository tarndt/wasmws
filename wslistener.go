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

//WebSockListener implements net.Listener and provides connections that are
//incoming websocket connections
type WebSockListener struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	acceptCh chan net.Conn
}

var _ net.Listener = (*WebSockListener)(nil)

//NewWebSocketListener constructs a new WebSockListener, the provided context
//is for the lifetime of the listener.
func NewWebSocketListener(ctx context.Context) *WebSockListener {
	ctx, cancel := context.WithCancel(ctx)
	wsl := &WebSockListener{
		ctx:       ctx,
		ctxCancel: cancel,
		acceptCh:  make(chan net.Conn, 8),
	}
	go func() { //Close queued connections
		<-ctx.Done()
		for {
			select {
			case conn := <-wsl.acceptCh:
				conn.Close()
				continue
			default:
			}
			break
		}
	}()
	return wsl
}

//HTTPAccept is a method that is mean to be used as http.HandlerFunc to accept inbound HTTP requests
// that are websocket connections
func (wsl *WebSockListener) HTTPAccept(wtr http.ResponseWriter, req *http.Request) {
	select {
	case <-wsl.ctx.Done():
		http.Error(wtr, "503: Service is shutdown", http.StatusServiceUnavailable)
		log.Printf("WebSockListener: WARN: A websocket listener's HTTP Accept was called when shutdown!")
		return
	default:
	}

	ws, err := websocket.Accept(wtr, req, nil)
	if err != nil {
		log.Printf("WebSockListener: ERROR: Could not accept websocket from %q; Details: %s", req.RemoteAddr, err)
	}

	conn := websocket.NetConn(wsl.ctx, ws, websocket.MessageBinary)
	select {
	case wsl.acceptCh <- conn:
	case <-wsl.ctx.Done():
		ws.Close(websocket.StatusBadGateway, fmt.Sprintf("Failed to accept connection before websocket listener shutdown; Details: %s", wsl.ctx.Err()))
	case <-req.Context().Done():
		ws.Close(websocket.StatusBadGateway, fmt.Sprintf("Failed to accept connection before websocket HTTP request cancelation; Details: %s", req.Context().Err()))
	}
}

//Accept fulfills the net.Listener interface and returns net.Conn that are incoming
// websockets
func (wsl *WebSockListener) Accept() (net.Conn, error) {
	select {
	case conn := <-wsl.acceptCh:
		return conn, nil
	case <-wsl.ctx.Done():
		return nil, fmt.Errorf("Listener closed; Details: %w", wsl.ctx.Err())
	}
}

//Close closes the listener
func (wsl *WebSockListener) Close() error {
	wsl.ctxCancel()
	return nil
}

//RemoteAddr returns a dummy websocket address to satisfy net.Listener
func (wsl *WebSockListener) Addr() net.Addr {
	return wsAddr{}
}

type wsAddr struct{}

func (wsAddr) Network() string { return "websocket" }

func (wsAddr) String() string { return "websocket" }
