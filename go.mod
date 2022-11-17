module sigs.k8s.io/aws-fsx-csi-driver

require (
	github.com/aws/aws-sdk-go v1.44.76
	github.com/container-storage-interface/spec v1.6.0
	github.com/golang/mock v1.6.0
	github.com/kubernetes-csi/csi-test v2.0.1+incompatible
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	google.golang.org/grpc v1.27.0
	k8s.io/apimachinery v0.22.3
	k8s.io/klog/v2 v2.60.1
	k8s.io/mount-utils v0.24.3
)

require (
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
)

go 1.19
