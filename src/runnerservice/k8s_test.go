package main

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var vite_config = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: code-server
  labels:
    app: code-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: code-server
  template:
    metadata:
      labels:
        app: code-server
    spec:
      containers:
        - name: vite-react-container
          image: anshik12/ideserver:vite_react_app
          command: 
            - code-server
          args:
            - --bind-addr
            - 0.0.0.0:8080
            - --auth
            - none
            - /home/vscode/code
            - --disable-telemetry
          ports:
            - containerPort: 8080
              name: code-server
            - containerPort: 3000
              name: react-port
          resources:
            requests:
              memory: "512Mi"
              cpu: "500m"
            limits:
              memory: "1Gi"
              cpu: "1000m"
          volumeMounts:
            - name: code-server-volume
              mountPath: /home/vscode/code
      volumes:
        - name: code-server-volume
          hostPath:
            path: /home/anshik/k8svolume
            type: DirectoryOrCreate
---
apiVersion: v1
kind: Service
metadata:
  name: code-server-service
spec:
  type: NodePort
  selector:
    app: code-server
  ports:
    - name: code-server
      protocol: TCP
      port: 8080
      targetPort: 8080
    - name: react-port
      protocol: TCP
      port: 3000
      targetPort: 3000`

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
			// Replicas: int32Ptr(1),
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
func TestEveryThing(t *testing.T) {
	orr, err := NewOrchestration()
	if err != nil {
		t.Errorf("orchestration failed to connect")
	}
	depManfaist, errr := getDeploymentManifest(vite_config)
	if errr != nil {
		t.Errorf(err.Error() + " yaml string is not correct")

	}
	err = orr.CreateDeployment(depManfaist)
	if err != nil {
		t.Errorf("failed to create deployment from mainfest " + err.Error())
	}

}

// int32Ptr is a helper function to return a pointer to an int32 value.
