package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	"github.com/tarndt/wasmws"
)

func main() {
	//App context setup
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	//Dial setup
	const dialTO = time.Second
	dialCtx, dialCancel := context.WithTimeout(appCtx, dialTO)
	defer dialCancel()

	//Connect to remote gRPC server
	const websocketURL = "ws://localhost:8080/grpc-proxy"
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	conn, err := grpc.DialContext(dialCtx, "passthrough:///"+websocketURL, grpc.WithContextDialer(wasmws.GRPCDialer), grpc.WithDisableRetry(), grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Could not gRPC dial: %s; Details: %s", websocketURL, err)
	}
	defer conn.Close()

	//Test setup
	client := pb.NewGreeterClient(conn)

	//Test transactions
	start := time.Now()
	const ops = 8192
	for i := 1; i <= ops; i++ {
		_, err := testTrans(appCtx, client)

		if err != nil {
			log.Fatalf("Test transaction %d failed; Details: %s", i, err)
		}
	}
	fmt.Printf("SUCCESS running %d transactions! (average %s per operation)\n", ops, time.Duration(float64(time.Since(start))/ops))
}

func testTrans(ctx context.Context, client pb.GreeterClient) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	reply, err := client.SayHello(ctx, &pb.HelloRequest{Name: "Websocket Test Client"})
	if err != nil {
		return "", fmt.Errorf("Could not say Hello to server; Details: %w", err)
	}
	return reply.GetMessage(), nil
}
