package wasmws

import (
	"syscall/js"
)

var (
	jsUndefined = js.Undefined()
	uint8Array  = js.Global().Get("Uint8Array")
)

type timeoutError struct{}

func (timeoutError) Error() string { return "deadline exceeded" }

func (timeoutError) Timeout() bool { return true }

func (timeoutError) Temporary() bool { return true }

type wsAddr string

func (wsAddr) Network() string { return "websocket" }

func (url wsAddr) String() string { return string(url) }
