/*
Copyright 2024 The Kubernetes Authors.

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

package cloud

import (
	"context"
	"errors"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-fsx-csi-driver/pkg/metrics"
)

// RecordRequestsMiddleware instruments FSx API calls with Prometheus metrics.
// It is a no-op when the metrics recorder is not initialized.
func RecordRequestsMiddleware() func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("RecordRequestsMiddleware",
			func(ctx context.Context, input middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
				start := time.Now()
				output, metadata, err := next.HandleFinalize(ctx, input)
				duration := time.Since(start).Seconds()
				labels := createLabels(ctx)
				metrics.Recorder().ObserveHistogram(metrics.APIRequestDuration, metrics.APIRequestDurationHelpText, duration, labels, nil)
				if err != nil {
					var apiErr smithy.APIError
					if errors.As(err, &apiErr) {
						if _, isThrottle := retry.DefaultThrottleErrorCodes[apiErr.ErrorCode()]; isThrottle {
							metrics.Recorder().IncreaseCount(metrics.APIRequestThrottles, metrics.APIRequestThrottlesHelpText, labels)
						} else {
							labels["code"] = apiErr.ErrorCode()
							metrics.Recorder().IncreaseCount(metrics.APIRequestErrors, metrics.APIRequestErrorsHelpText, labels)
						}
					}
				}
				return output, metadata, err
			}), middleware.After)
	}
}

// LogServerErrorsMiddleware logs server errors and throttles received from the FSx API.
// Throttle errors are logged at a higher verbosity to avoid flooding logs under normal bursty workloads.
func LogServerErrorsMiddleware() func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("LogServerErrorsMiddleware",
			func(ctx context.Context, input middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
				output, metadata, err := next.HandleFinalize(ctx, input)
				if err != nil {
					var apiErr smithy.APIError
					if errors.As(err, &apiErr) {
						if _, isThrottle := retry.DefaultThrottleErrorCodes[apiErr.ErrorCode()]; isThrottle {
							klog.V(4).ErrorS(apiErr, "Throttle error from AWS API")
						} else {
							klog.V(3).ErrorS(apiErr, "Error from AWS API")
						}
					} else {
						klog.ErrorS(err, "Unknown error attempting to contact AWS API")
					}
				}
				return output, metadata, err
			}), middleware.After)
	}
}

func createLabels(ctx context.Context) map[string]string {
	op := awsmiddleware.GetOperationName(ctx)
	if op == "" {
		op = "Unknown"
	}
	return map[string]string{"request": op}
}
