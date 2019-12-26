package wasmws

import (
	"syscall/js"
)

var (
	jsUndefined = js.Undefined()
	uint8Array  = js.Global().Get("Uint8Array")
)

//timeoutErr is a net.Addr implementation for the websocket to use when fufilling
// the net.Conn interface
type timeoutError struct{}

func (timeoutError) Error() string { return "deadline exceeded" }

func (timeoutError) Timeout() bool { return true }

func (timeoutError) Temporary() bool { return true }

//wsAddr is a net.Addr implementation for the websocket to use when fufilling
// the net.Conn interface
type wsAddr string

func (wsAddr) Network() string { return "websocket" }

func (url wsAddr) String() string { return string(url) }
