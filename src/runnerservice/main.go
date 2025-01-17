package main

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	fmt.Print("Hello, World!")

	listner, err := net.Listen("tcp", ":4040")
	if err != nil {
		panic(err)

	}

	srv := grpc.NewServer()
	reflection.Register(srv)
	if e := srv.Serve(listner); e != nil {
		panic(e)
	}
}
