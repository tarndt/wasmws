package wasmws

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

const echoServiceWebSockURL = "ws://echo.websocket.events"

//These tests are intended to run in a headless chrome instance (see test.bash)
// and access an external public websocket testing server.

func TestWebsocketEchoSmall(t *testing.T) {
	const testTO = time.Second * 10
	testCtx, testCancel := context.WithTimeout(context.Background(), testTO)
	defer testCancel()

	ws, err := New(testCtx, echoServiceWebSockURL)
	if err != nil {
		t.Fatalf("Could not construct test websocket against %q; Details: %s", echoServiceWebSockURL, err)
	}
	defer ws.Close()

	var msgBuf bytes.Buffer
	var copyBuf []byte
	var readBuf *bytes.Buffer
	for i := byte('!'); i < '~'; i++ {
		msgBuf.WriteByte(i)
		t.Logf("Echo:  %q", msgBuf.Bytes())
		copyBuf, readBuf = echo(t, bytes.NewReader(msgBuf.Bytes()), ws, true, copyBuf, readBuf)
	}
}

func TestWebsocketEchoLarge(t *testing.T) {
	const testTO = time.Second * 10
	testCtx, testCancel := context.WithTimeout(context.Background(), testTO)
	defer testCancel()

	ws, err := New(testCtx, echoServiceWebSockURL)
	if err != nil {
		t.Fatalf("Could not construct test websocket against %q; Details: %s", echoServiceWebSockURL, err)
	}
	defer ws.Close()

	var copyBuf []byte
	var readBuf *bytes.Buffer
	for i := 0; i < 10; i++ {
		copyBuf, readBuf = echo(t, strings.NewReader(testMsg), ws, true, copyBuf, readBuf)
	}
}

func echo(t testing.TB, in io.Reader, conn net.Conn, verify bool, optCopyBuf []byte, optReadBuf *bytes.Buffer) (copyBuf []byte, readBuf *bytes.Buffer) {
	//Buffer setup
	if optCopyBuf == nil {
		optCopyBuf = make([]byte, 64*1024)
	}
	if optReadBuf == nil {
		optReadBuf = new(bytes.Buffer)
	} else {
		optReadBuf.Reset()
	}

	//Verify?
	var verifyBuf bytes.Buffer
	if verify {
		in = io.TeeReader(in, &verifyBuf)
	}

	//Write
	n, err := io.CopyBuffer(conn, in, optCopyBuf)
	if err != nil {
		t.Fatalf("Write/Copy to echo server failed; Details: %s", err)
	}

	//Read
	limRdr := io.LimitedReader{R: conn, N: n}
	if optReadBuf.ReadFrom(&limRdr); err != nil {
		t.Fatalf("Read from echo server failed; Details: %s", err)
	}

	//Verify
	if verify {
		if expected, actual := verifyBuf.String(), optReadBuf.String(); expected != actual {
			t.Fatalf("Echo server returned %q rather than %q!", actual, expected)
		}
	}

	return optCopyBuf, optReadBuf
}

const testMsg = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Senectus et netus et malesuada fames ac turpis. Eget magna fermentum iaculis eu non. Sed tempus urna et pharetra pharetra. Eu scelerisque felis imperdiet proin fermentum. Pulvinar proin gravida hendrerit lectus a. Augue ut lectus arcu bibendum. Id porta nibh venenatis cras sed felis eget velit aliquet. Viverra accumsan in nisl nisi scelerisque eu ultrices. Ut tristique et egestas quis ipsum suspendisse ultrices. Diam volutpat commodo sed egestas egestas fringilla phasellus faucibus scelerisque. Dolor sit amet consectetur adipiscing elit ut aliquam.

Bibendum neque egestas congue quisque egestas diam in arcu cursus. Consectetur adipiscing elit duis tristique sollicitudin nibh sit amet commodo. Elit eget gravida cum sociis natoque penatibus. Vitae auctor eu augue ut lectus arcu bibendum at varius. Mi tempus imperdiet nulla malesuada pellentesque elit eget gravida cum. Lectus arcu bibendum at varius vel pharetra vel turpis. Dui faucibus in ornare quam viverra orci. Aenean euismod elementum nisi quis eleifend quam adipiscing. Nunc id cursus metus aliquam eleifend. Scelerisque mauris pellentesque pulvinar pellentesque. Congue quisque egestas diam in. Sed vulputate odio ut enim blandit volutpat maecenas. Eu lobortis elementum nibh tellus molestie nunc. Rhoncus urna neque viverra justo nec ultrices.

Tincidunt praesent semper feugiat nibh sed pulvinar proin gravida hendrerit. Sit amet facilisis magna etiam. Aliquet risus feugiat in ante. Nunc mi ipsum faucibus vitae aliquet nec. Sit amet aliquam id diam maecenas. Suspendisse potenti nullam ac tortor vitae purus faucibus ornare suspendisse. Egestas pretium aenean pharetra magna. Aliquam ut porttitor leo a diam. Tempus urna et pharetra pharetra massa massa ultricies mi quis. Id faucibus nisl tincidunt eget nullam non nisi. Vel orci porta non pulvinar neque laoreet suspendisse. Nec nam aliquam sem et tortor consequat id porta. Feugiat vivamus at augue eget arcu dictum. Mattis vulputate enim nulla aliquet porttitor lacus. Diam volutpat commodo sed egestas egestas fringilla phasellus faucibus. Lectus arcu bibendum at varius vel pharetra vel turpis nunc. Vel quam elementum pulvinar etiam non quam lacus suspendisse faucibus.

Mauris rhoncus aenean vel elit scelerisque mauris. Risus in hendrerit gravida rutrum quisque non tellus. Vulputate enim nulla aliquet porttitor lacus luctus accumsan. Montes nascetur ridiculus mus mauris vitae ultricies leo. Quisque sagittis purus sit amet volutpat consequat mauris. Viverra nibh cras pulvinar mattis nunc sed blandit libero. Nibh venenatis cras sed felis eget. Varius morbi enim nunc faucibus a pellentesque sit. Fringilla phasellus faucibus scelerisque eleifend donec. Elit duis tristique sollicitudin nibh sit amet commodo nulla. Adipiscing elit pellentesque habitant morbi tristique. Congue mauris rhoncus aenean vel. Quis vel eros donec ac odio tempor orci dapibus ultrices. Turpis cursus in hac habitasse platea. Consectetur adipiscing elit ut aliquam purus. Sapien faucibus et molestie ac feugiat sed lectus. Tincidunt ornare massa eget egestas purus viverra. Cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo.

Ut faucibus pulvinar elementum integer enim neque volutpat ac. Vitae sapien pellentesque habitant morbi tristique senectus et netus et. Sit amet porttitor eget dolor morbi non. Congue quisque egestas diam in arcu cursus. Nisl nisi scelerisque eu ultrices vitae auctor. Neque volutpat ac tincidunt vitae semper quis lectus nulla. Velit laoreet id donec ultrices. Aliquet porttitor lacus luctus accumsan. Ipsum a arcu cursus vitae. Donec et odio pellentesque diam volutpat commodo sed egestas. Consectetur adipiscing elit duis tristique sollicitudin nibh.`
