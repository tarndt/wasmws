package wasmws

import (
	"errors"
	"io"
	"sync"
	"syscall/js"
)

//streamReader is an io.ReadCloser implementation for Javascript's ReadableStream
// See: https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
type streamReader struct {
	remaining []byte
	jsPromise js.Value
	err       error
}

var streamReaderPool = sync.Pool{
	New: func() interface{} {
		return new(streamReader)
	},
}

func newStreamReaderPromise(streamPromise js.Value) *streamReader {
	sr := streamReaderPool.Get().(*streamReader)
	sr.jsPromise = streamPromise
	return sr
}

func (sr *streamReader) Close() error {
	sr.Reset()
	streamReaderPool.Put(sr)
	return nil
}

func (sr *streamReader) Reset() {
	const bufMax = socketStreamThresholdBytes
	sr.jsPromise, sr.err = js.Value{}, nil
	if cap(sr.remaining) < bufMax {
		sr.remaining = sr.remaining[:0]
	} else {
		sr.remaining = nil
	}
}

func (sr *streamReader) Read(p []byte) (n int, err error) {
	if sr.err != nil {
		return 0, sr.err
	}
	if len(sr.remaining) == 0 {
		readCh, errCh := make(chan []byte, 1), make(chan error, 1)

		successCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].Get("done").Bool() {
				errCh <- io.EOF
				return nil
			}
			jsBuf := args[0].Get("value")
			count := jsBuf.Get("byteLength").Int()

			var goBuf []byte
			if count <= cap(sr.remaining) {
				goBuf = sr.remaining[:count]
			} else {
				goBuf = make([]byte, count)
			}
			js.CopyBytesToGo(goBuf, jsBuf)
			readCh <- goBuf
			return nil
		})
		defer successCallback.Release()

		failureCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errCh <- errors.New(args[0].Get("message").String()) //Send TypeError
			return nil
		})
		defer failureCallback.Release()

		//Wait for callback
		sr.jsPromise.Call("read").Call("then", successCallback, failureCallback)
		select {
		case sr.remaining = <-readCh:
		case err := <-errCh:
			sr.err = err
			return 0, err
		}
	}

	n = copy(p, sr.remaining)
	sr.remaining = sr.remaining[n:]
	return n, nil
}
