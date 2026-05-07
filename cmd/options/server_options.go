/*
Copyright 2020 The Kubernetes Authors.

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

package options

import (
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/aws-fsx-csi-driver/pkg/driver"
)

// ServerOptions contains options and configuration settings for the driver server.
type ServerOptions struct {
	// DriverMode is the service mode the driver server should run in.
	DriverMode string
	// Endpoint is the endpoint that the driver server should listen on.
	Endpoint string
	// HTTPEndpoint is the TCP address for the Prometheus metrics HTTP server (e.g. :3301).
	// An empty value disables metrics.
	HTTPEndpoint string
	// MetricsCertFile is the path to the TLS certificate for the metrics server.
	MetricsCertFile string
	// MetricsKeyFile is the path to the TLS private key for the metrics server.
	MetricsKeyFile string
}

func (s *ServerOptions) AddFlags(fs *flag.FlagSet) string {
	fs.StringVar(&s.DriverMode, "mode", driver.AllMode, "Service mode the driver server should run in")
	fs.StringVar(&s.Endpoint, "endpoint", driver.DefaultCSIEndpoint, "Endpoint for the CSI driver server")
	fs.StringVar(&s.HTTPEndpoint, "http-endpoint", "", "TCP network address for the Prometheus metrics HTTP server (e.g. 0.0.0.0:3301). An empty value disables metrics.")
	fs.StringVar(&s.MetricsCertFile, "metrics-cert-file", "", "Path to the TLS certificate file for the metrics server. Requires --metrics-key-file.")
	fs.StringVar(&s.MetricsKeyFile, "metrics-key-file", "", "Path to the TLS private key file for the metrics server. Requires --metrics-cert-file.")

	return s.DriverMode
}
