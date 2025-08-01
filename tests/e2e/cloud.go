/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	fsx "sigs.k8s.io/aws-fsx-csi-driver/pkg/cloud"
)

type cloud struct {
	ec2client *ec2.Client
	s3client  *s3.Client
	fsx.Cloud
}

func NewCloud(region string) *cloud {
	config, _ := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))

	c, err := fsx.NewCloud(region)
	if err != nil {
		fmt.Sprintf("could not get NewCloud: %v", err)
		return nil
	}

	return &cloud{
		ec2.NewFromConfig(config),
		s3.NewFromConfig(config),
		c,
	}
}

func (c *cloud) getNodeInstance(clusterName string) (*ec2types.Instance, error) {
	request := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:KubernetesCluster"),
				Values: []string{clusterName},
			},
		},
	}

	instances := []ec2types.Instance{}
	ctx := context.Background()
	response, err := c.ec2client.DescribeInstances(ctx, request)
	if err != nil {
		return nil, err
	}
	for _, reservation := range response.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances in cluster %q found", clusterName)
	}

	return &instances[0], nil
}

func (c *cloud) createS3Bucket(name string) error {
	request := &s3.CreateBucketInput{
		Bucket: aws.String(name),
	}

	ctx := context.Background()
	_, err := c.s3client.CreateBucket(ctx, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *cloud) deleteS3Bucket(name string) error {
	request := &s3.DeleteBucketInput{
		Bucket: aws.String(name),
	}

	ctx := context.Background()
	_, err := c.s3client.DeleteBucket(ctx, request)
	if err != nil {
		return err
	}
	return nil
}

func getSecurityGroupIds(node *ec2types.Instance) []string {
	groups := []string{}
	for _, sg := range node.SecurityGroups {
		groups = append(groups, *sg.GroupId)
	}
	return groups
}
