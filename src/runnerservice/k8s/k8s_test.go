package k8s

import (
	"fmt"
	"strings"
	"testing"
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

func TestEveryThingCreate(t *testing.T) {
	orr, err := NewOrchestration()
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Print(orr)
	parts := strings.Split(vite_config, "---")
	if len(parts) < 2 {
		t.Error("Incompleter yaml filee....")
	}

	deploymentYAML := strings.TrimSpace(parts[0])
	serviceYAML := strings.TrimSpace(parts[1])
	depManfaist, errr := orr.GetDeploymentManifest(deploymentYAML, "react", "newiiiiiii0")
	if errr != nil {
		t.Errorf(errr.Error() + "deployment in yaml string is not correct...")

	}
	servManfaist, errr := orr.GetServiceManifest(serviceYAML, "react", "newiiiiiii0")
	if errr != nil {
		t.Errorf(errr.Error() + "service in yaml string is not correct...")

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

func TestEveryThingDelete(t *testing.T) {
	orr, err := NewOrchestration()

	err = orr.DeleteDeployment("newiiiiiii0")
	if err != nil {
		t.Errorf("failed to delete deployment " + err.Error())
	}
	err = orr.DeleteService("newiiiiiii0")
	if err != nil {
		t.Errorf("failed to delete service of deployment " + err.Error())
	}

}
