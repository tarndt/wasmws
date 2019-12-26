#/bin/bash!

# You can pass any args to this script you would normally pass to go test.
# For example: -cover, -v, etc
arg1="$1"

headlessChrome="$GOPATH/bin/wasmbrowsertest"
if [ ! -f "$headlessChrome" ]; then
	echo "Install headless Chrome helper: go get github.com/agnivade/wasmbrowsertest"
	exit 1
fi

# Run Tests
GOOS=js GOARCH=wasm go test -exec="$headlessChrome" "$@"
exit