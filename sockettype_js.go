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

func newSocketType(websocket js.Value) socketType {
	return newSocketTypeString(websocket.Get("binaryType").String())
}

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

func (st socketType) Set(websocket js.Value) {
	websocket.Set("binaryType", st.String())
	if debugVerbose {
		println("Websocket: Set websocket binaryType (mode) to", st.String())
	}
}
