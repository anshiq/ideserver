package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	IDESERVER "github.com/anshiq/ideserver/src/runnerservice/genproto"
)

func (s *RunnerService) MakeContainer(ctx context.Context, req *IDESERVER.MakeContainerRequest) (*IDESERVER.MakeContainerResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	stack := req.GetTechStack()
	hostName := req.GetHostname()
	yamlCode := req.GetYamlFileCode()

	if stack == "" {
		return nil, errors.New("tech stack cannot be empty")
	}
	if hostName == "" {
		return nil, errors.New("hostname cannot be empty")
	}
	if yamlCode == "" {
		return nil, errors.New("YAML file code cannot be empty")
	}

	parts := strings.Split(yamlCode, "---")
	if len(parts) < 2 {
		return nil, errors.New("YAML file code must contain at least two parts separated by '---'")
	}

	deploymentYAML := strings.TrimSpace(parts[0])
	serviceYAML := strings.TrimSpace(parts[1])

	deployment, err := Orc.GetDeploymentManifest(deploymentYAML, stack, hostName)
	if err != nil {
		log.Printf("Failed to get deployment manifest for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	service, err := Orc.GetServiceManifest(serviceYAML, stack, hostName)
	ActivePods.Add(hostName, service.GetName())
	if err != nil {
		log.Printf("Failed to get service manifest for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	if err := Orc.CreateDeployment(deployment); err != nil {
		log.Printf("Failed to create deployment for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	if err := Orc.CreateService(service); err != nil {
		log.Printf("Failed to create service for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	log.Printf("Successfully created container for stack %s and hostname %s", stack, hostName)
	return &IDESERVER.MakeContainerResponse{
		Msg: fmt.Sprintf("Container for stack %s has been created successfully! User for this action is %s.", stack, hostName),
		Err: "",
	}, nil
}

func (s *RunnerService) UpdateContainer(ctx context.Context, req *IDESERVER.MakeContainerRequest) (*IDESERVER.MakeContainerResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	stack := req.GetTechStack()
	hostName := req.GetHostname()
	yamlCode := req.GetYamlFileCode()

	if stack == "" {
		return nil, errors.New("tech stack cannot be empty")
	}
	if hostName == "" {
		return nil, errors.New("hostname cannot be empty")
	}
	if yamlCode == "" {
		return nil, errors.New("YAML file code cannot be empty")
	}

	parts := strings.Split(yamlCode, "---")
	if len(parts) < 2 {
		return nil, errors.New("YAML file code must contain at least two parts separated by '---'")
	}

	deploymentYAML := strings.TrimSpace(parts[0])
	serviceYAML := strings.TrimSpace(parts[1])

	deployment, err := Orc.GetDeploymentManifest(deploymentYAML, stack, hostName)
	if err != nil {
		log.Printf("Failed to get deployment manifest for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	service, err := Orc.GetServiceManifest(serviceYAML, stack, hostName)
	ActivePods.Add(hostName, service.GetName())
	if err != nil {
		log.Printf("Failed to get service manifest for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	if err := Orc.CreateDeployment(deployment); err != nil {
		log.Printf("Failed to create deployment for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	if err := Orc.CreateService(service); err != nil {
		log.Printf("Failed to create service for stack %s and hostname %s: %v", stack, hostName, err)
		return nil, err
	}

	log.Printf("Successfully created container for stack %s and hostname %s", stack, hostName)
	return &IDESERVER.MakeContainerResponse{
		Msg: fmt.Sprintf("Container for stack %s has been created successfully! User for this action is %s.", stack, hostName),
		Err: "",
	}, nil
}

func (s *RunnerService) DeleteContainer(ctx context.Context, req *IDESERVER.DeleteContainerRequest) (*IDESERVER.DeleteContainerResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	hostName := req.GetHostname()
	if hostName == "" {
		return nil, errors.New("hostname cannot be empty")
	}
	if err := Orc.DeleteDeployment(hostName); err != nil {
		log.Printf("Failed to delete deployment for hostname %s: %v", hostName, err)
		return nil, err
	}
	if err := Orc.DeleteService(hostName); err != nil {
		log.Printf("Failed to delete service for hostname %s: %v", hostName, err)
		return nil, err
	}
	log.Printf("Successfully deleted container for hostname %s", hostName)
	ActivePods.Delete(hostName)
	return &IDESERVER.DeleteContainerResponse{
		Msg: fmt.Sprintf("Container deleted successfully for hostname %s", hostName),
		Err: "",
	}, nil
}
