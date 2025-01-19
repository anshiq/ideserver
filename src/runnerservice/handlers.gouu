package main

import (
	"context"
	"fmt"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
)

func (s *RunnerService) CreateContainer(ctx context.Context, req IDESERVER.MakeContainerRequest) (*IDESERVER.MakeContainerResponse, error) {
	stack := req.GetStack()
	userId := req.GetUserId()
	// logic here
	return &IDESERVER.MakeContainerResponse{
		Status: 200,
		Msg:    fmt.Sprintf("Container for stack %s has been create successfully!!! user for this action is %s ", stack, userId),
		Err:    "",
	}, nil
}

func (s *RunnerService) ConnectUserContainer(ctx context.Context, req IDESERVER.ConnectContainerRequest) (*IDESERVER.ConnectContainerResponse, error) {
	x := req.GetContainerid()
	y := req.GetUserId()
	fmt.Print(x, y)
	return &IDESERVER.ConnectContainerResponse{
		Containerlink: "htthpp",
		Err:           "",
	}, nil
}
