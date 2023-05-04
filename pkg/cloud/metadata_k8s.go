package cloud

import (
	"context"
	"fmt"
	"os"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesAPIClient func() (kubernetes.Interface, error)

var DefaultKubernetesAPIClient = func() (kubernetes.Interface, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func KubernetesAPIInstanceInfo(clientset kubernetes.Interface) (*Metadata, error) {
	nodeName := os.Getenv("CSI_NODE_NAME")
	if nodeName == "" {
		return nil, fmt.Errorf("CSI_NODE_NAME env var not set")
	}

	// get node with k8s API
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting Node %v: %w", nodeName, err)
	}

	providerID := node.Spec.ProviderID
	if providerID == "" {
		return nil, fmt.Errorf("node providerID empty, cannot parse")
	}

	awsInstanceIDRegex := "s\\.i-[a-z0-9]+|i-[a-z0-9]+$"

	re := regexp.MustCompile(awsInstanceIDRegex)
	instanceID := re.FindString(providerID)
	if instanceID == "" {
		return nil, fmt.Errorf("did not find aws instance ID in node providerID string")
	}

	instanceInfo := Metadata{
		InstanceID: instanceID,
	}

	return &instanceInfo, nil
}
