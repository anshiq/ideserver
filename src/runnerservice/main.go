package main

import (
	"fmt"
	"net"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
	Hostiptopodmap "github.com/anshiq/ideserver/src/runnerservice/hostiptopodMap"
	K8s "github.com/anshiq/ideserver/src/runnerservice/k8s"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type RunnerService struct {
	IDESERVER.UnimplementedRunnerServiceServer
}

var Orc = K8s.NewOrchestration()
var ActivePods = Hostiptopodmap.NewActivePods()

func main() {
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
