/*
Copyright 2019 The Kubernetes Authors.

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

package driver

import (
	"context"
	"fmt"
	"net"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/util"
)

// Mode is the operating mode of the CSI driver.
type Mode string

const (
	// ControllerMode is the mode that only starts the controller service.
	ControllerMode = "controller"
	// NodeMode is the mode that only starts the node service.
	NodeMode = "node"
	// AllMode is the mode that only starts both the controller and the node service.
	AllMode = "all"
)

const (
	DriverName = "fsx.csi.aws.com"
)

type Driver struct {
	controllerService
	nodeService

	srv     *grpc.Server
	options *DriverOptions
	csi.UnimplementedIdentityServer
}

type DriverOptions struct {
	endpoint  string
	mode      string
	extraTags string
}

func NewDriver(options ...func(*DriverOptions)) (*Driver, error) {
	klog.InfoS("Driver Information", "Driver", DriverName, "Version", driverVersion)

	driverOptions := DriverOptions{
		endpoint: DefaultCSIEndpoint,
		mode:     AllMode,
	}
	for _, option := range options {
		option(&driverOptions)
	}

	driver := Driver{
		options: &driverOptions,
	}

	switch driverOptions.mode {
	case ControllerMode:
		driver.controllerService = newControllerService(&driverOptions)
	case NodeMode:
		driver.nodeService = newNodeService(&driverOptions)
	case AllMode:
		driver.controllerService = newControllerService(&driverOptions)
		driver.nodeService = newNodeService(&driverOptions)
	default:
		return nil, fmt.Errorf("unknown mode: %s", driverOptions.mode)
	}

	return &driver, nil
}

func (d *Driver) Run() error {
	scheme, addr, err := util.ParseEndpoint(d.options.endpoint)
	if err != nil {
		return err
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		return err
	}

	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			klog.ErrorS(err, "GRPC error")
		}
		return resp, err
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}
	d.srv = grpc.NewServer(opts...)

	csi.RegisterIdentityServer(d.srv, d)

	switch d.options.mode {
	case ControllerMode:
		csi.RegisterControllerServer(d.srv, d)
	case NodeMode:
		csi.RegisterNodeServer(d.srv, d)
	case AllMode:
		csi.RegisterControllerServer(d.srv, d)
		csi.RegisterNodeServer(d.srv, d)
	default:
		return fmt.Errorf("unknown mode: %s", d.options.mode)
	}

	klog.V(4).InfoS("Listening for connections", "address", listener.Addr())
	return d.srv.Serve(listener)
}

func (d *Driver) Stop() {
	klog.InfoS("Stopping server")
	d.srv.Stop()
}

func WithEndpoint(endpoint string) func(*DriverOptions) {
	return func(o *DriverOptions) {
		o.endpoint = endpoint
	}
}

func WithMode(mode string) func(*DriverOptions) {
	return func(o *DriverOptions) {
		o.mode = mode
	}
}

func WithExtraTags(extraTags string) func(*DriverOptions) {
	return func(o *DriverOptions) {
		o.extraTags = extraTags
	}
}
