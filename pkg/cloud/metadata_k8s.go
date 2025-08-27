package cloud

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesAPIClient func() (kubernetes.Interface, error)

const eksHybridPrefix = "eks-hybrid:///"

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

	if strings.HasPrefix(providerID, eksHybridPrefix) {
		return metadataForHybridNode(providerID)
	}

	// if not hybrid, assume AWS EC2
	return metadataForEC2Node(providerID)
}

func metadataForEC2Node(providerID string) (*Metadata, error) {
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

func metadataForHybridNode(providerID string) (*Metadata, error) {
	// provider ID for hybrid node is in formt eks:///region/clustername/instanceid
	info := strings.TrimPrefix(providerID, eksHybridPrefix)

	parts := strings.Split(info, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid hybrid node providerID format")
	}

	instanceInfo := Metadata{
		InstanceID: parts[2],
		Region:     parts[0],
	}
	return &instanceInfo, nil
}
