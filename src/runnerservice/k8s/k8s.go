package k8s

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	// informers "k8s.io/client-go/informers/apps/v1"
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
		// fmt.Println("kubeconfig file not found, attempting in-cluster configuration: %v\n", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			// fmt.Println("Failed to get k8s config in-cluster and kubeconfig file: %v\n", err.Error())
			return nil, err
		}
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		// fmt.Println("Failed to create Kubernetes clientset: %v", err.Error())
		return nil, err
	}
	fmt.Print("\n\nKube client and client sets created successfully and stable and about to start gprc server\n\n")
	return &Orchestration{
		ClientSet: clientSet,
	}, nil
}

func (o *Orchestration) CreateDeployment(deployment *appsv1.Deployment) error {
	if deployment == nil {
		return errors.New("deployment object cannot be nil")
	}

	if deployment.ObjectMeta.Name == "" {
		return errors.New("deployment name cannot be empty")
	}

	deploymentClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	log.Printf("Creating deployment %q in namespace %q...", deployment.ObjectMeta.Name, apiv1.NamespaceDefault)

	result, err := deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment %q: %w", deployment.ObjectMeta.Name, err)
	}

	log.Printf("Successfully created deployment %q.\n", result.GetName())
	return nil
}
func (o *Orchestration) CreateService(service *apiv1.Service) error {
	if service == nil {
		return errors.New("service object cannot be nil")
	}

	if service.ObjectMeta.Name == "" {
		return errors.New("service name cannot be empty")
	}

	serviceClient := o.ClientSet.CoreV1().Services(apiv1.NamespaceDefault)
	log.Printf("Creating service %q in namespace %q...", service.ObjectMeta.Name, apiv1.NamespaceDefault)

	result, err := serviceClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service %q: %w", service.ObjectMeta.Name, err)
	}

	log.Printf("Successfully created service %q.\n", result.GetName())
	return nil
}

func (o *Orchestration) UpdateDeployment(deploymentuniqueId string, ports []int32) error {
	if deploymentuniqueId == "" {
		return errors.New("deployment ID cannot be empty")
	}

	if len(ports) == 0 {
		return errors.New("ports cannot be empty")
	}

	deploymentsClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	log.Printf("Updating deployment %q in namespace %q...", deploymentuniqueId, apiv1.NamespaceDefault)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := deploymentsClient.Get(context.TODO(), deploymentuniqueId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get latest version of Deployment %q: %v", deploymentuniqueId, getErr)
		}

		for _, port := range ports {
			result.Spec.Template.Spec.Containers[0].Ports = append(result.Spec.Template.Spec.Containers[0].Ports, apiv1.ContainerPort{ContainerPort: port})
		}

		_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return fmt.Errorf("failed to update deployment %q: %w", deploymentuniqueId, retryErr)
	}

	log.Printf("Successfully updated deployment %q.\n", deploymentuniqueId)
	return nil
}
func (o *Orchestration) UpdateService(serviceuniqueId string, ports []int32) error {
	if serviceuniqueId == "" {
		return errors.New("service ID cannot be empty")
	}
	if len(ports) == 0 {
		return errors.New("ports cannot be empty")
	}

	serviceClient := o.ClientSet.CoreV1().Services(apiv1.NamespaceDefault)
	log.Printf("Updating service %q in namespace %q...", serviceuniqueId, apiv1.NamespaceDefault)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := serviceClient.Get(context.TODO(), serviceuniqueId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get latest version of Service %q: %v", serviceuniqueId, getErr)
		}

		for _, port := range ports {

			result.Spec.Ports = append(result.Spec.Ports, apiv1.ServicePort{
				TargetPort: intstr.FromInt32(port),
				Protocol:   apiv1.ProtocolTCP,
			})
		}

		_, updateErr := serviceClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return fmt.Errorf("failed to update service %q: %w", serviceuniqueId, retryErr)
	}

	log.Printf("Successfully updated service %q.\n", serviceuniqueId)
	return nil
}

func (o *Orchestration) DeleteDeployment(deploymentuniqueId string) error {
	if deploymentuniqueId == "" {
		return errors.New("deployment ID cannot be empty")
	}

	deploymentsClient := o.ClientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
	deletePolicy := metav1.DeletePropagationForeground
	log.Printf("Deleting deployment %q in namespace %q...", deploymentuniqueId, apiv1.NamespaceDefault)

	if err := deploymentsClient.Delete(context.TODO(), deploymentuniqueId, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return fmt.Errorf("failed to delete deployment %q: %w", deploymentuniqueId, err)
	}

	log.Printf("Successfully deleted deployment %q.\n", deploymentuniqueId)
	return nil
}

func (o *Orchestration) DeleteService(serviceuniqueId string) error {
	if serviceuniqueId == "" {
		return errors.New("service ID cannot be empty")
	}

	serviceClient := o.ClientSet.CoreV1().Services(apiv1.NamespaceDefault)
	deletePolicy := metav1.DeletePropagationForeground
	log.Printf("Deleting service %q in namespace %q...", serviceuniqueId, apiv1.NamespaceDefault)

	if err := serviceClient.Delete(context.TODO(), serviceuniqueId, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return fmt.Errorf("failed to delete service %q: %w", serviceuniqueId, err)
	}

	log.Printf("Successfully deleted service %q.\n", serviceuniqueId)
	return nil
}
func (o *Orchestration) GetServiceManifest(serviceString string, stack string, serviceuniqueId string) (*apiv1.Service, error) {
	if serviceString == "" {
		return nil, errors.New("service manifest string cannot be empty")
	}

	if serviceuniqueId == "" {
		return nil, errors.New("service unique ID cannot be empty")
	}

	service := &apiv1.Service{}
	if err := yaml.Unmarshal([]byte(serviceString), service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service manifest: %w", err)
	}

	service.ObjectMeta.Name = serviceuniqueId
	service.Spec.Selector = map[string]string{
		"app":   serviceuniqueId,
		"stack": stack,
	}

	return service, nil
}

func (o *Orchestration) GetDeploymentManifest(deploymentString string, stack string, deploymentuniqueId string) (*appsv1.Deployment, error) {
	if deploymentString == "" {
		return nil, errors.New("deployment manifest string cannot be empty")
	}

	if deploymentuniqueId == "" {
		return nil, errors.New("deployment unique ID cannot be empty")
	}

	deployment := &appsv1.Deployment{}
	if err := yaml.Unmarshal([]byte(deploymentString), deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment manifest: %w", err)
	}

	deployment.ObjectMeta.Name = deploymentuniqueId
	deployment.Spec.Replicas = int32Ptr(1)
	deployment.Spec.Selector.MatchLabels = map[string]string{
		"app": deploymentuniqueId,
	}
	deployment.Spec.Template.ObjectMeta.Labels = map[string]string{
		"app":   deploymentuniqueId,
		"stack": stack,
	}
	deployment.Spec.Template.Spec.Volumes[0].Name = deploymentuniqueId
	deployment.Spec.Template.Spec.Volumes[0].HostPath.Path = "/home/" + deploymentuniqueId
	deployment.Spec.Template.Spec.InitContainers[1].VolumeMounts[0].Name = deploymentuniqueId
	deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts[0].Name = deploymentuniqueId
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name = deploymentuniqueId

	deployment.Spec.Template.Spec.AutomountServiceAccountToken = returnFalseAddr()
	return deployment, nil
}
func (o *Orchestration) GetDeploymentLiveStatus(deploymentuniqueId string, namespace string, depStatus chan int) {
	labelOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.FieldSelector = "metadata.name=" + deploymentuniqueId
	})
	factory := informers.NewSharedInformerFactoryWithOptions(o.ClientSet, 2*time.Second, informers.WithNamespace(namespace), labelOptions)
	informer := factory.Apps().V1().Deployments().Informer()

	stopChan := make(chan struct{})

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Handle add event if needed
		},
		DeleteFunc: func(obj interface{}) {
			// Handle delete event if needed
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			dep, ok := newObj.(*appsv1.Deployment)
			if !ok {
				return
			}

			// Look for an Available condition with status True
			for _, condition := range dep.Status.Conditions {
				if condition.Type == "Available" && condition.Status == "True" {
					depStatus <- 1
					return
				}
			}

			// Check if the deployment is ready
			if dep.Status.ReadyReplicas == dep.Status.Replicas &&
				dep.Status.Replicas > 0 {
				depStatus <- 1
				return
			}
		},
	})

	// Initializes all active informers and starts the internal goroutine
	factory.Start(stopChan)
	factory.WaitForCacheSync(stopChan)

	// Set a timeout for 4 minutes
	timeout := time.After(4 * time.Minute)
	fmt.Print("at both")
	select {
	case <-depStatus:
		// If a signal is received on depStatus, stop the informer
		fmt.Print("hit chan first")
		close(stopChan)
		return
	case <-timeout:
		// If 4 minutes have passed, stop the informer
		depStatus <- 2 // failed to retrive info
		fmt.Print("hit time out")
		close(stopChan)
		return
	}
}

// func getDeployment(namespace string, name string, labelKey string, client *kubernetes.Clientset) *appsv1.Deployment {
// 	d, err := client.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
// 	if err != nil {
// 		return nil
// 	}

// 	if _, ok := d.GetLabels()[labelKey]; !ok {
// 		return nil
// 	}

// 	return d
// }

func returnFalseAddr() *bool {
	k := false
	return &k
}

func int32Ptr(i int32) *int32 {
	return &i
}

// func int64Ptr(i int64) *int64 {
// 	return &i
// }
