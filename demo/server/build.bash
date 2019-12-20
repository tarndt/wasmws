#!/bin/bash
set -eu

GOARCH=wasm GOOS=js go build -o "./static/client.wasm" ../client/main.go
ln -fs $GOROOT/misc/wasm/wasm_exec.js ./static/wasm_exec.js
go run "$GOROOT/src/crypto/tls/generate_cert.go" --host localhost
go build
