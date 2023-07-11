# Copyright 2023 The Kubernetes Authors.
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

FROM --platform=$BUILDPLATFORM golang:1.20 as builder
WORKDIR /go/src/github.com/kubernetes-sigs/aws-fsx-csi-driver
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN OS=$TARGETOS ARCH=$TARGETARCH make $TARGETOS/$TARGETARCH

# https://github.com/aws/eks-distro-build-tooling/blob/main/eks-distro-base/Dockerfile.minimal-base-csi-ebs#L36
FROM public.ecr.aws/eks-distro-build-tooling/eks-distro-minimal-base-csi-ebs-builder:latest-al2 as rpm-installer

# shallow install systemd and the kernel which are not needed in the final container image
# since lustre is not run as a systemd service and the kernel module is node loaded via the container
# to avoid pulling in a large tree of unnecessary dependencies
RUN set -x && \
    enable_extra lustre && \
    clean_install "kernel systemd" true true && \
    clean_install lustre-client && \
    remove_package "kernel systemd" true && \
    cleanup "fsx-csi"

FROM public.ecr.aws/eks-distro-build-tooling/eks-distro-minimal-base-csi-ebs:latest-al2 AS linux-amazon

COPY --from=rpm-installer /newroot /

COPY --from=builder /go/src/github.com/kubernetes-sigs/aws-fsx-csi-driver/bin/aws-fsx-csi-driver /bin/aws-fsx-csi-driver
COPY THIRD-PARTY /

ENTRYPOINT ["/bin/aws-fsx-csi-driver"]
