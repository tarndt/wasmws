package wasmws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall/js"
	"time"
)

const (
	socketStreamThresholdBytes = 1024  //If enabled, the Blob interface will be used when consecutive messages exceed this threshold
	debugVerbose               = false //Set to true if you are debugging issues, this gates many prints that would kill performance
)

var (
	//EnableBlobStreaming allows the browser provided Websocket's streaming
	// interface to be used, if supported.
	EnableBlobStreaming bool = true

	//ErrWebsocketClosed is returned when operations are performed on a closed Websocket
	ErrWebsocketClosed = errors.New("WebSocket: Web socket is closed")

	blobSupported bool //set to true by init if browser supports the Blob interface
)

//init checks to see if the browser hosting this application support the Websocket Blob interface
func init() {
	newBlob := js.Global().Get("Blob")
	if newBlob.Equal(jsUndefined) {
		blobSupported = false
		return
	}

	testBlob := js.Global().Get("Blob").New()
	blobSupported = !testBlob.Get("arrayBuffer").Equal(jsUndefined) && !testBlob.Get("stream").Equal(js.Undefined())
	if debugVerbose {
		println("Websocket: Init: EnableBlobStreaming is", EnableBlobStreaming, "and blobSupported is", blobSupported)
	}
}

//WebSocket is a Go struct that wraps the web browser's JavaScript websocket object and provides a net.Conn interface
type WebSocket struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	URL        string
	ws         js.Value
	wsType     socketType
	enableBlob bool
	openCh     chan struct{}

	readLock  sync.Mutex
	remaining io.Reader
	readCh    chan io.Reader

	readDeadlineTimer *time.Timer
	newReadDeadlineCh chan time.Time

	writeLock sync.Mutex
	errCh     chan error

	writeDeadlineTimer *time.Timer
	newWriteDeadlineCh chan time.Time

	cleanup []func()
}

//New returns a new WebSocket using the provided dial context and websocket URL.
// The URL should be in the form of "ws://host/path..." for unsecured websockets
// and "wss://host/path..." for secured websockets. If tunnel a TLS based protocol
// over a "wss://..." websocket you will get TLS twice, once on the websocket using
// the browsers TLS stack and another using the Go (or other compiled) TLS stack.
func New(dialCtx context.Context, URL string) (*WebSocket, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ws := &WebSocket{
		ctx:       ctx,
		ctxCancel: cancel,

		URL:        URL,
		ws:         js.Global().Get("WebSocket").New(URL),
		wsType:     socketTypeArrayBuffer,
		enableBlob: EnableBlobStreaming && blobSupported,
		openCh:     make(chan struct{}),

		readCh:            make(chan io.Reader, 8),
		readDeadlineTimer: time.NewTimer(time.Minute),
		newReadDeadlineCh: make(chan time.Time, 1),

		errCh:              make(chan error, 1),
		writeDeadlineTimer: time.NewTimer(time.Minute),
		newWriteDeadlineCh: make(chan time.Time, 1),

		cleanup: make([]func(), 0, 3),
	}

	ws.wsType.Set(ws.ws)
	ws.setDeadline(ws.readDeadlineTimer, time.Time{})
	ws.setDeadline(ws.writeDeadlineTimer, time.Time{})
	ws.addHandler(ws.handleOpen, "open")
	ws.addHandler(ws.handleClose, "close")
	ws.addHandler(ws.handleError, "error")
	ws.addHandler(ws.handleMessage, "message")

	go func() { //handle shutdown
		<-ws.ctx.Done()
		if debugVerbose {
			println("Websocket: Shutdown")
		}

		ws.ws.Call("close")
		for _, cleanup := range ws.cleanup {
			cleanup()
		}

		for {
			select {
			case pending := <-ws.readCh:
				if closer, hasClose := pending.(io.Closer); hasClose {
					closer.Close()
				}
				continue

			default:
			}
			break
		}
	}()

	//Wait for connection or failure
	select {
	case <-ws.ctx.Done():
		return nil, ErrWebsocketClosed

	case <-dialCtx.Done():
		ws.ctxCancel()
		return nil, dialCtx.Err()

	case err := <-ws.errCh:
		ws.ctxCancel()
		return nil, err

	case <-ws.openCh:
		if debugVerbose {
			println("Websocket: Connected!")
		}
	}

	//Find out what kind of socket we are
	if ws.wsType = newSocketType(ws.ws); ws.wsType == socketTypeUnknown {
		if debugVerbose {
			println("Websocket: Invalid socket type")
		}
		ws.ctxCancel()
		return nil, fmt.Errorf("WebSocket: %q's method 'websocket.binaryType' returned %q which is an invalid socket type!", ws.URL, ws.wsType)
	}
	return ws, nil
}

//Close shuts the websocket down
func (ws *WebSocket) Close() error {
	if debugVerbose {
		println("Websocket: Internal close")
	}
	ws.ctxCancel()
	return nil
}

//LocalAddr returns a dummy websocket address to satisfy net.Conn, see: wsAddr
func (ws *WebSocket) LocalAddr() net.Addr {
	return wsAddr(ws.URL)
}

//RemoteAddr returns a dummy websocket address to satisfy net.Conn, see: wsAddr
func (ws *WebSocket) RemoteAddr() net.Addr {
	return wsAddr(ws.URL)
}

//Write implements the standard io.Writer interface. Due to the JavaScript writes
// being internally buffered it will never block and a write timeout from a
// previous write may not surface until a subsequent write.
func (ws *WebSocket) Write(buf []byte) (n int, err error) {
	//Check for noop
	writeCount := len(buf)
	if writeCount < 1 {
		return 0, nil
	}

	//Lock
	ws.writeLock.Lock()
	defer ws.writeLock.Unlock()

	//Check for close or new deadline
	select {
	case <-ws.ctx.Done():
		return 0, ErrWebsocketClosed

	case newWriteDeadline := <-ws.newWriteDeadlineCh:
		ws.setDeadline(ws.writeDeadlineTimer, newWriteDeadline)

	default:
	}

	//Write
	select {
	case <-ws.ctx.Done():
		return 0, ErrWebsocketClosed

	case err = <-ws.errCh:
		if debugVerbose {
			println("Websocket: Write: Outstanding error", err.Error())
		}
		ws.ctxCancel()
		return 0, fmt.Errorf("WebSocket: Previous write resulted in stream error; Details: %w", err)

	case newWriteDeadline := <-ws.newWriteDeadlineCh:
		ws.setDeadline(ws.writeDeadlineTimer, newWriteDeadline)

	case <-ws.writeDeadlineTimer.C:
		if remaining := ws.ws.Get("bufferedAmount").Int(); remaining > 0 {
			return 0, timeoutError{}
		}

	default:
		jsBuf := uint8Array.New(len(buf))
		js.CopyBytesToJS(jsBuf, buf)
		ws.ws.Call("send", jsBuf)
		if debugVerbose {
			println("Websocket: Write", writeCount, "bytes", "(content: "+fmt.Sprintf("%q", buf)+")")
		}
	}

	//Check for status updates before returning
	select {
	case err = <-ws.errCh:
		if debugVerbose {
			println("Websocket: Write: outstanding error", err.Error())
		}
		ws.ctxCancel()
		return 0, fmt.Errorf("WebSocket: Write resulted in stream error; Details: %w", err)

	case <-ws.writeDeadlineTimer.C:
		if reamining := ws.ws.Get("bufferedAmount").Int(); reamining > 0 {
			return 0, timeoutError{}
		}

	default:
	}
	return writeCount, nil
}

//Read implements the standard io.Reader interface (typical semantics)
func (ws *WebSocket) Read(buf []byte) (int, error) {
	//Check for noop
	if len(buf) < 1 {
		return 0, nil
	}

	//Lock
	ws.readLock.Lock()
	defer ws.readLock.Unlock()

	//Check for close or new deadline
	select {
	case <-ws.ctx.Done():
		return 0, ErrWebsocketClosed

	case newReadDeadline := <-ws.newReadDeadlineCh:
		if debugVerbose {
			println("Websocket: Set new read deadline (pre-read)")
		}
		ws.setDeadline(ws.readDeadlineTimer, newReadDeadline)

	default:
	}

	for {
		//Get next chunk
		if ws.remaining == nil {
			if debugVerbose {
				println("Websocket: Read wait on queue-")
			}
			select {
			case ws.remaining = <-ws.readCh:

			case <-ws.ctx.Done():
				return 0, ErrWebsocketClosed

			case newReadDeadline := <-ws.newReadDeadlineCh:
				if debugVerbose {
					println("Websocket: Set new read deadline (during read)")
				}
				ws.setDeadline(ws.readDeadlineTimer, newReadDeadline)

			case <-ws.readDeadlineTimer.C:
				if debugVerbose {
					println("Websocket: Read timeout")
				}
				return 0, timeoutError{}
			}
		}

		//Read from chunk
		if debugVerbose {
			println("Websocket: Reading")
		}
		n, err := ws.remaining.Read(buf)
		if err == io.EOF {
			if closer, hasClose := ws.remaining.(io.Closer); hasClose {
				closer.Close()
			}
			ws.remaining, err = nil, nil
			if n < 1 {
				continue
			}
		}
		if debugVerbose {
			println("Websocket: Read", n, "bytes", "(content: "+fmt.Sprintf("%q", buf[:n])+")")
		}
		return n, err
	}
}

func (ws *WebSocket) SetDeadline(future time.Time) (err error) {
	select {
	case ws.newWriteDeadlineCh <- future:
		ws.newReadDeadlineCh <- future
	case ws.newReadDeadlineCh <- future:
		ws.newWriteDeadlineCh <- future
	}
	return nil
}

//SetWriteDeadline implements the Conn SetWriteDeadline method
func (ws *WebSocket) SetWriteDeadline(future time.Time) error {
	if debugVerbose {
		println("Websocket: Set write deadline for", future.String())
	}
	ws.newWriteDeadlineCh <- future
	return nil
}

//SetReadDeadline implements the Conn SetReadDeadline method
func (ws *WebSocket) SetReadDeadline(future time.Time) error {
	if debugVerbose {
		println("Websocket: Set read deadline for", future.String())
	}
	ws.newReadDeadlineCh <- future
	return nil
}

//setDeadline is used internally; Only call from New or Read!
func (ws *WebSocket) setDeadline(timer *time.Timer, future time.Time) error {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	if !future.IsZero() {
		timer.Reset(future.Sub(time.Now()))
	}
	return nil
}

//addHandler is used internall by the WebSocket constructor
func (ws *WebSocket) addHandler(handler func(this js.Value, args []js.Value), event string) {
	jsHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(this, args)
		return nil
	})
	cleanup := func() {
		ws.ws.Call("removeEventListener", event, jsHandler)
		jsHandler.Release()
	}
	ws.ws.Call("addEventListener", event, jsHandler)
	ws.cleanup = append(ws.cleanup, cleanup)
}

//handleOpen is a callback for JavaScript to notify Go when the websocket is open:
// See: https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/onopen
func (ws *WebSocket) handleOpen(_ js.Value, _ []js.Value) {
	if debugVerbose {
		println("Websocket: Open JS callback!")
	}
	close(ws.openCh)
}

//handleClose is a callback for JavaScript to notify Go when the websocket is closed:
// See: https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/onclose
func (ws *WebSocket) handleClose(_ js.Value, _ []js.Value) {
	if debugVerbose {
		println("Websocket: Close JS callback!")
	}
	ws.ctxCancel()
}

//handleError is a callback for JavaScript to notify Go when the websocket is in an error state:
// See: https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/onerror
func (ws *WebSocket) handleError(_ js.Value, args []js.Value) {
	if debugVerbose {
		println("Websocket: Error JS Callback")
	}
	errMsg := "Unknown error"
	if len(args) > 0 {
		errMsg = args[0].String()
	}

	select {
	case ws.errCh <- errors.New(errMsg):
	default:
	}
}

//handleMessage is a callback for JavaScript to notify Go when the websocket has a new message:
// See: https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/onmessage
func (ws *WebSocket) handleMessage(_ js.Value, args []js.Value) {
	if debugVerbose {
		println("Websocket: New Message JS Callback")
	}

	select {
	case <-ws.ctx.Done():
	default:
	}

	var rdr io.Reader
	var size int

	switch ws.wsType {
	case socketTypeArrayBuffer:
		rdr, size = newReaderArrayBuffer(args[0].Get("data"))
		//Should we switch to blobs for next time?
		if ws.enableBlob && size > socketStreamThresholdBytes {
			ws.wsType = socketTypeBlob
			ws.wsType.Set(ws.ws)
		}

	case socketTypeBlob:
		jsBlob := args[0].Get("data")
		if size = jsBlob.Get("size").Int(); size <= socketStreamThresholdBytes {
			rdr = newReaderArrayPromise(jsBlob.Call("arrayBuffer"))
			//switch to ArrayBuffers for next read
			ws.wsType = socketTypeArrayBuffer
			ws.wsType.Set(ws.ws)
		} else {
			rdr = newStreamReaderPromise(jsBlob.Call("stream").Call("getReader"))
		}

	default:
		panic(fmt.Sprintf("WebSocket: Unknown socket type: %s (%d)", ws.wsType, ws.wsType))
	}

	select {
	case ws.readCh <- rdr: //Try non-blocking queue first...
		if debugVerbose {
			println("Websocket: JS read callback sync enqueue")
		}

	case <-ws.ctx.Done():

	default:
		go func() { //Don't block in a callback!
			select {
			case ws.readCh <- rdr:
				if debugVerbose {
					println("Websocket: JS read callback async enqueue")
				}

			case <-ws.ctx.Done():
			}
		}()
	}
}
