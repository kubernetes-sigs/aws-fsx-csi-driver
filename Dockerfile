# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM --platform=$BUILDPLATFORM golang:1.19.2-bullseye as builder
WORKDIR /go/src/github.com/kubernetes-sigs/aws-fsx-csi-driver
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN OS=$TARGETOS ARCH=$TARGETARCH make $TARGETOS/$TARGETARCH

FROM amazonlinux:2 AS linux-amazon
RUN yum update -y
RUN yum install util-linux libyaml -y \
    && amazon-linux-extras install -y lustre2.10
    
COPY --from=builder /go/src/github.com/kubernetes-sigs/aws-fsx-csi-driver/bin/aws-fsx-csi-driver /bin/aws-fsx-csi-driver
COPY THIRD-PARTY /

ENTRYPOINT ["/bin/aws-fsx-csi-driver"]
