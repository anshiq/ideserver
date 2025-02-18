package k8s

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

type Orchestration struct {
	ClientSet *kubernetes.Clientset
}

func NewOrchestration() *Orchestration {
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
			log.Panic("Failed to get k8s config incluster and kubeconfig file")
		}
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic("failed to create Kubernetes clientset: %w", err)
		return nil
	}

	return &Orchestration{
		ClientSet: clientSet,
	}
}

func (o *Orchestration) CreateDeployment(deployment *appsv1.Deployment) error {
	if deployment == nil {
		return errors.New("deployment object cannot be nil")
	}

	namespaceClient := o.ClientSet.CoreV1().Namespaces()
	deploymentClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	serviceClient := o.ClientSet.CoreV1().Services(apiv1.NamespaceDefault)
	fmt.Println("Creating deployment...", namespaceClient, deploymentClient, serviceClient)

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

		// result.Spec.Replicas =&int32(1)                           // reduce replica count
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
func getServiceManifest(serviceString string, serviceuniqueId string) (*apiv1.Service, error) {
	service := &apiv1.Service{}
	err := yaml.Unmarshal([]byte(serviceString), service)
	service.ObjectMeta.Name = serviceuniqueId
	service.Spec.Selector = map[string]string{
		"app": serviceuniqueId,
	}
	if err != nil {
		return nil, err
	}
	return service, nil
}
func getDeploymentManifest(deploymentString string, deploymentuniqueId string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}

	err := yaml.Unmarshal([]byte(deploymentString), deployment)
	if err != nil {
		return nil, err
	}
	deployment.ObjectMeta.Name = deploymentuniqueId
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": deploymentuniqueId,
		},
	}

	deployment.Spec.Template.Spec.AutomountServiceAccountToken = returnFalseAddr()
	return deployment, nil
}

func returnFalseAddr() *bool {
	k := false
	return &k
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}
