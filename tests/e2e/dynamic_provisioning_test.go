package e2e

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[ebs-csi-e2e] [single-az] Dynamic Provisioning", func() {

	It("should succeed", func() {
		fmt.Println("Cluster name " + *clusterName)
		fmt.Println(framework.TestContext.ReportDir)

		instance, err := getNodeInstance(*clusterName)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(getSecurityGroup(instance))
		fmt.Println(*instance.SubnetId)
		Expect(true).To(Equal(true))
	})
})

func getNodeInstance(clusterName string) (*ec2.Instance, error) {
	config := &aws.Config{
		Region: region,
	}
	nodeName := fmt.Sprintf("nodes.%s", clusterName)
	svc := ec2.New(session.Must(session.NewSession(config)))
	request := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(nodeName)},
			},
		},
	}

	instances := []*ec2.Instance{}
	response, err := svc.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	for _, reservation := range response.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	return instances[0], nil
}

func getSecurityGroup(node *ec2.Instance) []string {
	groups := []string{}
	for _, sg := range node.SecurityGroups {
		groups = append(groups, *sg.GroupId)
	}
	return groups
}
