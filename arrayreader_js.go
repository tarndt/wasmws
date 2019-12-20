package wasmws

import (
	"errors"
	"io"
	"sync"
	"syscall/js"
)

//arrayReader is an io.ReadCloser implementation for Javascript ArrayBuffers
// See: https://developer.mozilla.org/en-US/docs/Web/API/Body/arrayBuffer
type arrayReader struct {
	jsPromise js.Value
	remaining []byte

	read bool
	err  error
}

var arrayReaderPool = sync.Pool{
	New: func() interface{} {
		return new(arrayReader)
	},
}

func newReaderArrayPromise(arrayPromise js.Value) *arrayReader {
	ar := arrayReaderPool.Get().(*arrayReader)
	ar.jsPromise = arrayPromise
	return ar
}

func newReaderArrayBuffer(arrayBuffer js.Value) (*arrayReader, int) {
	ar := arrayReaderPool.Get().(*arrayReader)
	ar.remaining, ar.read = ar.fromArray(arrayBuffer), true
	return ar, len(ar.remaining)
}

func (ar *arrayReader) Close() error {
	ar.Reset()
	arrayReaderPool.Put(ar)
	return nil
}

func (ar *arrayReader) Reset() {
	const bufMax = socketStreamThresholdBytes
	ar.jsPromise, ar.read, ar.err = js.Value{}, false, nil
	if cap(ar.remaining) < bufMax {
		ar.remaining = ar.remaining[:0]
	} else {
		ar.remaining = nil
	}
}

func (ar *arrayReader) Read(buf []byte) (n int, err error) {
	if ar.err != nil {
		return 0, ar.err
	}

	if !ar.read {
		ar.read = true
		readCh, errCh := make(chan []byte, 1), make(chan error, 1)

		successCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			readCh <- ar.fromArray(args[0])
			return nil
		})
		defer successCallback.Release()

		failureCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errCh <- errors.New(args[0].Get("message").String()) //Send TypeError
			return nil
		})
		defer failureCallback.Release()

		//Wait for callback
		ar.jsPromise.Call("then", successCallback, failureCallback)
		select {
		case ar.remaining = <-readCh:
		case err := <-errCh:
			return 0, err
		}
	}

	if len(ar.remaining) < 1 {
		return 0, io.EOF
	}
	n = copy(buf, ar.remaining)
	ar.remaining = ar.remaining[n:]
	return n, nil
}

func (ar *arrayReader) fromArray(arrayBuffer js.Value) []byte {
	jsBuf := uint8Array.New(arrayBuffer)
	count := jsBuf.Get("byteLength").Int()

	var goBuf []byte
	if count <= cap(ar.remaining) {
		goBuf = ar.remaining[:count]
	} else {
		goBuf = make([]byte, count)
	}
	js.CopyBytesToGo(goBuf, jsBuf)
	return goBuf
}
