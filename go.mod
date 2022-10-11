module sigs.k8s.io/aws-fsx-csi-driver

require (
	github.com/aws/aws-sdk-go v1.40.57
	github.com/container-storage-interface/spec v1.2.0
	github.com/golang/mock v1.6.0
	github.com/kubernetes-csi/csi-test v2.0.1+incompatible
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	google.golang.org/genproto v0.0.0-20190502173448-54afdca5d873 // indirect
	google.golang.org/grpc v1.23.1
	google.golang.org/protobuf v1.28.0 // indirect
	k8s.io/apimachinery v0.17.0
	k8s.io/klog/v2 v2.60.1
	k8s.io/mount-utils v0.24.3
)

replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/apiserver => k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.0
	k8s.io/code-generator => k8s.io/code-generator v0.17.0
	k8s.io/component-base => k8s.io/component-base v0.17.0
	k8s.io/cri-api => k8s.io/cri-api v0.17.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.0
	k8s.io/kubectl => k8s.io/kubectl v0.17.0
	k8s.io/kubelet => k8s.io/kubelet v0.17.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.0
	k8s.io/metrics => k8s.io/metrics v0.17.0
	k8s.io/node-api => k8s.io/node-api v0.17.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.0
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.0
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.0
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)

go 1.16
