package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	"github.com/tarndt/wasmws"
)

//go:generate ./build.bash

type helloServer struct{}

func (*helloServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	//App context setup
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	//Setup HTTP / Websocket server
	router := http.NewServeMux()
	wsl := wasmws.NewWebSocketListener(appCtx)
	router.HandleFunc("/grpc-proxy", wsl.HTTPAccept)
	router.Handle("/", http.FileServer(http.Dir("./static")))
	httpServer := &http.Server{Addr: ":8080", Handler: router}
	//Run HTTP server
	go func() {
		defer appCancel()
		log.Printf("ERROR: HTTP Listen and Server failed; Details: %s", httpServer.ListenAndServe())
	}()

	//gRPC setup
	creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
	if err != nil {
		log.Fatalf("Failed to contruct gRPC TSL credentials from {cert,key}.pem: %s", err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterGreeterServer(grpcServer, new(helloServer))
	//Run gRPC server
	go func() {
		defer appCancel()

		if err := grpcServer.Serve(wsl); err != nil {
			log.Printf("ERROR: Failed to serve gRPC connections; Details: %s", err)
		}
	}()

	//Handle signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Printf("INFO: Received shutdown signal: %s", <-sigs)
		appCancel()
	}()

	//Shutdown
	<-appCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*2)
	defer shutdownCancel()

	grpcShutdown := make(chan struct{}, 1)
	go func() {
		grpcServer.GracefulStop()
		grpcShutdown <- struct{}{}
	}()

	httpServer.Shutdown(shutdownCtx)
	select {
	case <-grpcShutdown:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
}
