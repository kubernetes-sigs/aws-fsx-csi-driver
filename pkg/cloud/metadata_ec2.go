package cloud

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"k8s.io/klog/v2"
)

type EC2MetadataClient func() (EC2Metadata, error)

type imdsClient struct {
	*imds.Client
}

var DefaultEC2MetadataClient = func() (EC2Metadata, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	svc := imds.NewFromConfig(cfg)
	return &imdsClient{svc}, nil
}

func EC2MetadataInstanceInfo(svc EC2Metadata, regionFromSession string) (*Metadata, error) {
	doc, err := svc.GetInstanceIdentityDocument(context.Background(), nil)
	klog.InfoS("Retrieving EC2 instance identity Metadata", "regionFromSession", regionFromSession)
	if err != nil {
		return nil, fmt.Errorf("could not get EC2 instance identity metadata: %w", err)
	}

	if len(doc.InstanceID) == 0 {
		return nil, fmt.Errorf("could not get valid EC2 instance ID")
	}

	if len(doc.InstanceType) == 0 {
		return nil, fmt.Errorf("could not get valid EC2 instance type")
	}

	if len(doc.Region) == 0 {
		if len(regionFromSession) != 0 {
			doc.Region = regionFromSession
		} else {
			return nil, fmt.Errorf("could not get valid EC2 Region")
		}
	}

	if len(doc.AvailabilityZone) == 0 {
		return nil, fmt.Errorf("could not get valid EC2 availability zone")
	}

	instanceInfo := Metadata{
		InstanceID:       doc.InstanceID,
		InstanceType:     doc.InstanceType,
		Region:           doc.Region,
		AvailabilityZone: doc.AvailabilityZone,
	}

	return &instanceInfo, nil
}
