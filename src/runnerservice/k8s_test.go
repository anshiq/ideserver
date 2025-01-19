package main

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNewOrchestration tests the NewOrchestration function.
func TestNewOrchestration(t *testing.T) {
	_, err := NewOrchestration()
	if err == nil {
		t.Errorf("This test is designed to fail without a valid kubeconfig or in-cluster config")
	}
}

// TestCreateDeployment tests the CreateDeployment method.
func TestCreateDeployment(t *testing.T) {
	orch, _ := NewOrchestration()
	// Create a sample deployment object.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	// Call the CreateDeployment method.
	err := orch.CreateDeployment(deployment)
	if err != nil {
		t.Errorf("CreateDeployment failed:")
	}

	// Verify the deployment was created.
	createdDeployment, err := orch.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault).Get(context.TODO(), "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get created deployment: %v", err)
	}

	if createdDeployment.Name != "test-deployment" {
		t.Errorf("Expected deployment name 'test-deployment', got '%s'", createdDeployment.Name)
	}
}

// int32Ptr is a helper function to return a pointer to an int32 value.
