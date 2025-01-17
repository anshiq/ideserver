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

// the method defined on this struct should not be already existing on main.proto as service

func main() {

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
