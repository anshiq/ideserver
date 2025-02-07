package main

import (
	"context"
	"fmt"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
	hostiptopodmap "github.com/anshiq/ideserver/src/runnerservice/hostiptopodMap"
)

var ActivePods = hostiptopodmap.NewActivePods()

func (s *RunnerService) CreateContainer(ctx context.Context, req IDESERVER.MakeContainerRequest) (*IDESERVER.MakeContainerResponse, error) {
	stack := req.GetTechStack()
	hostName := req.GetHostname()
	// logic here
	return &IDESERVER.MakeContainerResponse{
		Msg: fmt.Sprintf("Container for stack %s has been create successfully!!! user for this action is %s ", stack, hostName),
		Err: "",
	}, nil
}

func (s *RunnerService) GetPodDnsFromMap(ctx context.Context, req IDESERVER.ConnectContainerRequest) (*IDESERVER.ConnectContainerResponse, error) {
	x := req.GetHostname()
	fmt.Print(x)
	return &IDESERVER.ConnectContainerResponse{
		PodDns: "",
	}, nil
}
func (s *RunnerService) DeletePod(ctx context.Context, req IDESERVER.DeleteContainerRequest) (*IDESERVER.DeleteContainerResponse, error) {
	return &IDESERVER.DeleteContainerResponse{
		Msg: "Container deleted successfully for hostname" + req.GetHostname(),
		Err: "",
	}, nil
}
