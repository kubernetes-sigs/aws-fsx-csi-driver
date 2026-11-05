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
	"testing"

	flag "github.com/spf13/pflag"
)

func TestControllerOptions(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		want ControllerOptions
	}{
		{
			name: "defaults",
			want: ControllerOptions{ClusterTagIsolation: "disable"},
		},
		{
			name: "all flags",
			args: []string{"--extra-tags=k1=v1,k2=v2", "--cluster-name=my-cluster", "--cluster-tag-key=example.com/cluster", "--cluster-tag-isolation=enforce"},
			want: ControllerOptions{ExtraTags: "k1=v1,k2=v2", ClusterName: "my-cluster", ClusterTagKey: "example.com/cluster", ClusterTagIsolation: "enforce"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Environment variables are not read.
			t.Setenv("CLUSTER_NAME", "from-env")
			t.Setenv("CLUSTER_TAG_KEY", "from-env")
			t.Setenv("CLUSTER_TAG_ISOLATION", "backfill")
			o := &ControllerOptions{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			o.AddFlags(fs)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("parse: %v", err)
			}
			if *o != tc.want {
				t.Fatalf("got %+v, want %+v", *o, tc.want)
			}
		})
	}
}
