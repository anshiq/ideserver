package main

import (
	"context"
	"fmt"
	"net"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type RunnerService struct {
	IDESERVER.UnimplementedRunnerServiceServer
}
type RunnerServiceI interface {
	MakeContainer(context.Context, *IDESERVER.MakeContainerRequest) (*IDESERVER.DeleteContainerResponse, error)
	GetContainerDns(context.Context, *IDESERVER.ConnectContainerRequest) (*IDESERVER.ConnectContainerResponse, error)
	DeleteContainer(context.Context, *IDESERVER.DeleteContainerRequest) (*IDESERVER.DeleteContainerResponse, error)
}

func main() {
	orc, err := NewOrchestration()
	if err != nil {
		panic(err)
	}
	fmt.Print(orc)
	ActivePods.Add("anshik", "http://localhost:8082")
	ActivePods.Add("sarb", "https://sidhu-moosewala1.blogspot.com/2025/02/physics-pdf.html")
	listner, err := net.Listen("tcp", ":8081")
	if err != nil {
		panic(err)
	}
	srv := grpc.NewServer()
	IDESERVER.RegisterRunnerServiceServer(srv, &RunnerService{})
	reflection.Register(srv)
	fmt.Print("grpc listening on :8081")
	if e := srv.Serve(listner); e != nil {
		panic(e)
	}
}
