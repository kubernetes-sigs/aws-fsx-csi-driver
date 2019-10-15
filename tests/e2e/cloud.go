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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
)

type cloud struct {
	ec2client *ec2.EC2
	s3client  *s3.S3
}

func NewCloud(region string) *cloud {
	config := &aws.Config{
		Region: aws.String(region),
	}
	sess := session.Must(session.NewSession(config))
	return &cloud{
		ec2client: ec2.New(sess),
		s3client:  s3.New(sess),
	}
}

func (c *cloud) getNodeInstance(clusterName string) (*ec2.Instance, error) {
	nodeName := fmt.Sprintf("nodes.%s", clusterName)
	request := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(nodeName)},
			},
		},
	}

	instances := []*ec2.Instance{}
	response, err := c.ec2client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	for _, reservation := range response.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	return instances[0], nil
}

func (c *cloud) createS3Bucket(name string) error {
	request := &s3.CreateBucketInput{
		Bucket: aws.String(name),
	}

	_, err := c.s3client.CreateBucket(request)
	if err != nil {
		return err
	}
	return nil
}

func (c *cloud) deleteS3Bucket(name string) error {
	request := &s3.DeleteBucketInput{
		Bucket: aws.String(name),
	}

	_, err := c.s3client.DeleteBucket(request)
	if err != nil {
		return err
	}
	return nil
}

func getSecurityGroupIds(node *ec2.Instance) []string {
	groups := []string{}
	for _, sg := range node.SecurityGroups {
		groups = append(groups, *sg.GroupId)
	}
	return groups
}
