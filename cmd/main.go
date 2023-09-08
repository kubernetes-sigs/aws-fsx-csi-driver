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

package main

import (
	flag "github.com/spf13/pflag"
	logsapi "k8s.io/component-base/logs/api/v1"
	json "k8s.io/component-base/logs/json"

	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver"
)

func main() {
	fs := flag.NewFlagSet("aws-fsx-csi-driver", flag.ExitOnError)

	if err := logsapi.RegisterLogFormat(logsapi.JSONLogFormat, json.Factory{}, logsapi.LoggingBetaOptions); err != nil {
		klog.ErrorS(err, "failed to register JSON log format")
	}

	options := GetOptions(fs)

	drv, err := driver.NewDriver(
		driver.WithEndpoint(options.ServerOptions.Endpoint),
		driver.WithMode(options.ServerOptions.DriverMode),
		driver.WithExtraTags(options.ControllerOptions.ExtraTags),
		driver.WithRetryTaintRemoval(options.NodeOptions.RetryTaintRemoval),
	)

	if err != nil {
		klog.ErrorS(err, "failed to create driver")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	if err := drv.Run(); err != nil {
		klog.ErrorS(err, "failed to run driver")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}
