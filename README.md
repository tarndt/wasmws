# wasmws: Webassembly Websocket (for Go)

## What is wasms? Why would I want to use this?

[wasmws](https://github.com/tarndt/wasmws) was written primarily to allow [Go](https://golang.org/) applications targeting [WASM](https://en.wikipedia.org/wiki/WebAssembly) to communicate with a [gRPC](https://grpc.io/) server. This is normally challenging for two reasons: 

1. In general, many internet ingress paths are not [HTTP/2](https://en.wikipedia.org/wiki/HTTP/2) end to end (gRPC uses HTTP/2). In particular, most CDN vendors do not support HTTP/2 back-haul to origin (ex. [Cloudflare](https://support.cloudflare.com/hc/en-us/articles/214534978-Are-the-HTTP-2-or-SPDY-protocols-supported-between-Cloudflare-and-the-origin-server-)).
2. Browser WASM applications cannot use [grpc-go](https://github.com/grpc/grpc-go) due to the low level networking that go-grpc requires for native operation not being available. (ex. ``dial tcp: Protocol not available`` fun...)

This library allows you to use a websocket as [net.Conn](https://golang.org/pkg/net/#Conn) for arbitrary traffic, this includes running protocols like HTTP, gRPC or any other TCP protocol over it). This is most useful for protocols that are not normally exposed to client side web applications. In our examples we will focus on gRPC since that was my use-case. 

### Aproach taken

#### Client-side (Go WASM application running in browser)
wasmws provides Go WASM specific [net.Conn](https://golang.org/pkg/net/#Conn) implementation that is backed by [a browser native websocket](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API):
```
myConn, err := wasmws.Dial("websocket", "ws://demos.kaazing.com/echo")
```
It is fairly straight forward to use this to set up a gRPC connection:
```
conn, err := grpc.DialContext(dialCtx, "passthrough:///"+websocketURL, grpc.WithContextDialer(wasmws.GRPCDialer), grpc.WithTransportCredentials(creds))
```
See the [demo client](https://github.com/tarndt/wasmws/blob/master/demo/client/main.go) for an extended example.

#### Server-side
wasmws includes websocket [net.Listener](https://golang.org/pkg/net/#Listener) that provides a [HTTP handler method](https://golang.org/pkg/net/http/#HandlerFunc) to accept HTTP websocket connections...
```
wsl := wasmws.NewWebSocketListener(appCtx)
router.HandleFunc("/grpc-proxy", wsl.HTTPAccept)
```
And a network listener to provide net.Conns to network oriented servers:
```
err := grpcServer.Serve(wsl)
```
See the [demo server](https://github.com/tarndt/wasmws/blob/master/demo/server/main.go) for an extended example. If you need more server-side helpers checkout [nhooyr.io/websocket](https://github.com/nhooyr/websocket) which these helpers use themselves.

#### Security

If you use a secure websocket and gRPC or HTTPS this means you get double TLS (once using the browser's TLS stack and once uses Go's). Unless the extra defense in depth is desirable, you may want to use a unsecured websocket.

## Performance

Due to challenges around having the sever run as a native application and the client running in a browser, I have not yet added unit benches... or tests :( yet. However, running the demo which performs 8192 gRPC hello world calls provides an idea of library's performance:

Median of 6 runs:

 * 	FireFox 71.0 (64-bit) on Linux:
     * ``SUCCESS running 8192 transactions! (average 452.88µs per operation)``
 * 	Chrome Version 79.0.3945.88 (Official Build) (64-bit) on Linux:
     * ``SUCCESS running 8192 transactions! (average 475.485µs per operation)``

This implementation tries to be intelligent about managing buffers (via [pooling](https://golang.org/pkg/sync/#Pool)) and switch from JavaScript [ArrayBuffer](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/ArrayBuffer) and streaming [Blob](https://developer.mozilla.org/en-US/docs/Web/API/Blob) based websocket reading based the size of the chunks/messages being received.

The test results above are from tests run on a local development workstation:

 * CPU: AMD Ryzen 9 3900X 12-Core Processor
 * OS: Ubuntu 18.04 LTS (Linux 4.15.X)

## Running the demo

If you do not have Go installed, you will of course need to [install it](https://golang.org/doc/install).

1. Checkout repo: ``git clone https://github.com/tarndt/wasmws.git``
2. Change to demo directory: ``cd ./wasmws/demo/server``
3. Build the client, server and run the server (which serves the client): ``./build.bash && ./server``
4. Open [http://localhost:8080/](http://localhost:8080/) in your web browser
5. Open the web console (Ctrl+Shift+K in Firefox, Ctrl+Shift+I in Chrome) and observe the output
		
## Alternatives

1. Use [gRPC-Web](https://github.com/grpc/grpc-web) as a HTTP to gRPC gateway/proxy. (If you don't want to running extra middleware..)
2. Use "[nhooyr.io/websocket](https://github.com/nhooyr/websocket)"'s implemtation which unlike "wasmws" does not use the browser provided websocket functionality. Test and bench your own use-case!
		
## Future

wasmws is actively being maintained, but that does not mean there are not things to do:

* [More code comments and godocs (WIP)](https://github.com/tarndt/wasmws/issues/1)
* [Unit tests](https://github.com/tarndt/wasmws/issues/2)/[benches](https://github.com/tarndt/wasmws/issues/3) (manual testing so far due to difficulties of testing native app and WASM app together)
* [Further optimization/profiling](https://github.com/tarndt/wasmws/issues/4)
* [Testing on browsers besides Firefox and Chrome](https://github.com/tarndt/wasmws/issues/5)
		
## Contributing

[Issues](https://github.com/tarndt/wasmws/issues), and espcially issues with [pull requests](https://github.com/tarndt/wasmws/pulls) are welcome!
