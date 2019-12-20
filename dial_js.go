package wasmws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

func Dial(network, address string) (net.Conn, error) {
	return DialContext(context.Background(), network, address)
}

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "websocket" {
		return nil, fmt.Errorf("Invalid network: %q; Details: Only \"websocket\" network is supported", network)
	}
	if !(strings.HasPrefix(address, "ws://") || strings.HasPrefix(address, "wss://")) {
		return nil, errors.New("Invalid address: websocket address should be a websocket URL that starts with ws:// or wss://")
	}
	return New(ctx, address)
}

func GRPCDialer(ctx context.Context, address string) (net.Conn, error) {
	return DialContext(ctx, "websocket", address)
}
