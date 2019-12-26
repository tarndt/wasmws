package wasmws

/*
Package wasm is a library allows you to use a websocket as net.Conn for arbitrary traffic (this includes tunneling protocols like HTTP, gRPC or any other TCP protocol over it). This is most useful for protocols that are not normally exposed to client side web applications. The specific motivation of this package was to allow Go applications targeting WASM to communicate with a gRPC server.

wasmws.WebSocket is the provided net.Conn implementation intended to be used from within Go WASM applications and wasmws.WebSockListener is the provided net.Listener intended to be used from server side native Go applications to accept client connections.

For extended details and examples please see: https://github.com/tarndt/wasmws/blob/master/README.md
*/