package k8s

import (
	"context"
	"fmt"
	"strings"
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
      initContainers:
        - name: init-volume
          image: busybox
          command: ['sh', '-c']
          args:
            - |
              chown -R 1001:1001 /workspace &&
              chmod -R 777 /workspace
          securityContext:
            runAsUser: 0
            runAsGroup: 0
            privileged: true
          volumeMounts:
            - name: code-server-volume
              mountPath: /workspace
        - name: init-code
          image: anshik12/ideserver:vite_react_app
          imagePullPolicy: Always
          command: ["sh", "-c"]
          args:
            - |
              cp -rf /home/temp_code/* /workspace/ || true
          securityContext:
            runAsUser: 1001
            runAsGroup: 1001
          volumeMounts:
            - name: code-server-volume
              mountPath: /workspace
      containers:
        - name: vite-react-container
          image: anshik12/ideserver:vite_react_app
          imagePullPolicy: Always
          command: ["/usr/bin/code-server"]
          args:
            - --bind-addr
            - 0.0.0.0:8080
            - --auth
            - none
            - /workspace 
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
              mountPath: /workspace
          env:
            - name: HOME
              value: /home/vscode
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
      targetPort: 3000
`

// TestNewOrchestration tests the NewOrchestration function.

// TestCreateDeployment tests the CreateDeployment method.
func TestCreateDeployment(t *testing.T) {
	orch, err := NewOrchestration()
	if err != nil {
		t.Error(err.Error())
	}
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
	err = orch.CreateDeployment(deployment)
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

// lll
// lll
func TestEveryThing(t *testing.T) {
	orr, err := NewOrchestration()
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Print(orr)
	parts := strings.Split(vite_config, "---")
	if len(parts) < 2 {
		t.Error("Incompleter yaml filee.....")
	}

	deploymentYAML := strings.TrimSpace(parts[0])
	serviceYAML := strings.TrimSpace(parts[1])
	depManfaist, errr := orr.GetDeploymentManifest(deploymentYAML, "react", "newiiiiiii0")
	servManfaist, errr := orr.GetServiceManifest(serviceYAML, "react", "newiiiiiii0")
	if errr != nil {
		t.Errorf(errr.Error() + " yaml string is not correct...")

	}

	err = orr.CreateDeployment(depManfaist)
	if err != nil {
		t.Errorf("failed to create deployment from mainfest " + err.Error())
	}
	err = orr.CreateService(servManfaist)
	if err != nil {
		t.Errorf("failed to create service of deployment from mainfest " + err.Error())
	}

}

// int32Ptr is a helper function to return a pointer to an int32 value.
