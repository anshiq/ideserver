package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

type Orchestration struct {
	ClientSet *kubernetes.Clientset
}

func NewOrchestration() (*Orchestration, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println("kubeconfig file not found, attempting in-cluster configuration...")
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client configuration: %w", err)
		}
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return &Orchestration{
		ClientSet: clientSet,
	}, nil
}

func (o *Orchestration) CreateDeployment(deployment *appsv1.Deployment) error {
	if deployment == nil {
		return errors.New("deployment object cannot be nil")
	}

	deploymentClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	fmt.Println("Creating deployment...")

	result, err := deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	fmt.Printf("Successfully created deployment %q.\n", result.GetName())
	return nil
}

func (o *Orchestration) UpdateDeployment(deploymentName string) error {
	deploymentsClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	fmt.Println("updating deployment...")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := deploymentsClient.Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("Failed to get latest version of Deployment: %v", getErr)
		}

		result.Spec.Replicas = int32Ptr(1)                           // reduce replica count
		result.Spec.Template.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
		_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})

		return updateErr
	})
	if retryErr != nil {
		return retryErr
	}
	fmt.Println("Updated deployment...")
	return nil
}

func (o *Orchestration) DeleteDeployment(deploymentName string) error {
	deploymentsClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.TODO(), deploymentName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return err
	}
	return nil
}

func getDeploymentManifest(deploymentName string, environmentName string, imageDocker string, containerPort uint) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: "usercontainer",
			Labels:    map[string]string{"name": deploymentName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": environmentName},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": environmentName},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  deploymentName + "-container",
							Image: imageDocker,
							Ports: []apiv1.ContainerPort{{
								ContainerPort: int32(containerPort),
								// HostIP:        deploymentName + ".myapp.com",
							}},
						},
					},
					AutomountServiceAccountToken: returnFalseAddr(),
					ServiceAccountName:           "dynamic-development-env",
				},
			},
		},
	}
	return deployment, nil
}

func returnFalseAddr() *bool {
	k := false
	return &k
}

func int32Ptr(i int32) *int32 {
	return &i
}
