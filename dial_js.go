package wasmws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

//Dial is a standard legacy network dialer that returns a websocket-based connection.
//See: DialContext for details on the network and address.
func Dial(network, address string) (net.Conn, error) {
	return DialContext(context.Background(), network, address)
}

//DialContext is a standard context-aware network dialer that returns a websocket-based connection.
// The address is a URL that should be in the form of "ws://host/path..." for unsecured websockets
// and "wss://host/path..." for secured websockets. If tunnel a TLS based protocol
// over a "wss://..." websocket you will get TLS twice, once on the websocket using
// the browsers TLS stack and another using the Go (or other compiled) TLS stack.
func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "websocket" {
		return nil, fmt.Errorf("Invalid network: %q; Details: Only \"websocket\" network is supported", network)
	}
	if !(strings.HasPrefix(address, "ws://") || strings.HasPrefix(address, "wss://")) {
		return nil, errors.New("Invalid address: websocket address should be a websocket URL that starts with ws:// or wss://")
	}
	return New(ctx, address)
}

//GRPCDialer is a helper that can be used with grpc.WithContextDialer to call DialContext.
//The address provided to the calling grpc.Dial should be in the form "passthrough:///"+websocketURL
// where websocketURL matches the description in DialContext.
func GRPCDialer(ctx context.Context, address string) (net.Conn, error) {
	return DialContext(ctx, "websocket", address)
}
