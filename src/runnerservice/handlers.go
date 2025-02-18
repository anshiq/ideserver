package main

import (
	"context"
	"fmt"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
)

func (s *RunnerService) MakeContainer(ctx context.Context, req *IDESERVER.MakeContainerRequest) (*IDESERVER.MakeContainerResponse, error) {
	stack := req.GetTechStack()
	hostName := req.GetHostname()
	yamlCode := req.GetYamlFileCode()
	fmt.Println(yamlCode)

	// logic here
	return &IDESERVER.MakeContainerResponse{
		Msg: fmt.Sprintf("Container for stack %s has been create successfully!!! user for this action is %s ", stack, hostName),
		Err: "",
	}, nil
}

func (s *RunnerService) GetContainerDns(ctx context.Context, req *IDESERVER.ConnectContainerRequest) (*IDESERVER.ConnectContainerResponse, error) {
	x := req.GetHostname()
	dns := ActivePods.Get(x)
	fmt.Println(x, dns)
	return &IDESERVER.ConnectContainerResponse{
		PodDns: dns,
	}, nil
}
func (s *RunnerService) DeleteContainer(ctx context.Context, req *IDESERVER.DeleteContainerRequest) (*IDESERVER.DeleteContainerResponse, error) {
	return &IDESERVER.DeleteContainerResponse{
		Msg: "Container deleted successfully for hostname" + req.GetHostname(),
		Err: "",
	}, nil
}
