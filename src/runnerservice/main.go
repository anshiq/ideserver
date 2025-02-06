package main

import (
	"fmt"
	"net"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type RunnerService struct {
	IDESERVER.UnimplementedRunnerServiceServer
}

func main() {
	orc, err := NewOrchestration()
	if err != nil {
		panic(err)
	}
	fmt.Print(orc)

	listner, err := net.Listen("tcp", ":4040")
	if err != nil {
		panic(err)
	}
	srv := grpc.NewServer()
	IDESERVER.RegisterRunnerServiceServer(srv, &RunnerService{})
	reflection.Register(srv)
	fmt.Print("grpc listening on :4040")
	if e := srv.Serve(listner); e != nil {
		panic(e)
	}
}
