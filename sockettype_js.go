package wasmws

import (
	"syscall/js"
)

const (
	socketTypeUnknown = iota
	socketTypeBlob
	socketTypeArrayBuffer
)

type socketType uint8

//newSocketType returns the socket type of the provided JavaScript websocket object
func newSocketType(websocket js.Value) socketType {
	return newSocketTypeString(websocket.Get("binaryType").String())
}

//newSocketTypeString returns a socketType from a string of the socket type that
// matches the JavaScript websock.binaryType property:
// See https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/binaryType
func newSocketTypeString(wsTypeStr string) socketType {
	switch wsTypeStr {
	case "blob":
		return socketTypeBlob
	case "arraybuffer":
		return socketTypeArrayBuffer
	default:
		return socketTypeUnknown
	}
}

//String returns the a string of the socket type that matches the JavaScript
// websock.binaryType property: See https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/binaryType
func (st socketType) String() string {
	switch st {
	case socketTypeArrayBuffer:
		return "arraybuffer"
	case socketTypeBlob:
		return "blob"
	default:
		return "unknown"
	}
}

//Set sets the type of the provided JavaScript websocket to itself
func (st socketType) Set(websocket js.Value) {
	websocket.Set("binaryType", st.String())
	if debugVerbose {
		println("Websocket: Set websocket binaryType (mode) to", st.String())
	}
}
