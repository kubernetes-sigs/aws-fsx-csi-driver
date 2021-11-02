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

PKG=github.com/kubernetes-sigs/aws-fsx-csi-driver
IMAGE?=amazon/aws-fsx-csi-driver
VERSION=v0.6.0
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X ${PKG}/pkg/driver.driverVersion=${VERSION} -X ${PKG}/pkg/driver.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/driver.buildDate=${BUILD_DATE}"
GO111MODULE=on
GOPROXY=direct
GOPATH=$(shell go env GOPATH)
GOOS=$(shell go env GOOS)

.EXPORT_ALL_VARIABLES:

.PHONY: aws-fsx-csi-driver
aws-fsx-csi-driver:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o bin/aws-fsx-csi-driver ./cmd/

bin /tmp/helm:
	@mkdir -p $@

bin/helm: | /tmp/helm bin
	@curl -o /tmp/helm/helm.tar.gz -sSL https://get.helm.sh/helm-v3.5.3-${GOOS}-amd64.tar.gz
	@tar -zxf /tmp/helm/helm.tar.gz -C bin --strip-components=1
	@rm -rf /tmp/helm/*

.PHONY: verify
verify:
	./hack/verify-all

.PHONY: test
test:
	go test -v -race ./pkg/...
	go test -v ./tests/sanity/...

.PHONY: test-e2e
test-e2e:
	DRIVER_NAME=aws-fsx-csi-driver \
	CONTAINER_NAME=fsx-plugin \
	TEST_EXTRA_FLAGS='--cluster-name=$$CLUSTER_NAME' \
	AWS_REGION=us-west-2 \
	AWS_AVAILABILITY_ZONES=us-west-2a \
	NODE_COUNT=1 \
	TEST_PATH=./tests/e2e/... \
	GINKGO_FOCUS=".*" \
	GINKGO_SKIP="subPath.should.be.able.to.unmount.after.the.subpath.directory.is.deleted|\[Disruptive\]|\[Serial\]" \
	./hack/e2e/run.sh

.PHONY: image
image:
	docker build -t $(IMAGE):latest .

.PHONY: push
push:
	docker push $(IMAGE):latest

.PHONY: image-release
image-release:
	docker build -t $(IMAGE):$(VERSION) .

.PHONY: push-release
push-release:
	docker push $(IMAGE):$(VERSION)

generate-kustomize: bin/helm
	cd charts/aws-fsx-csi-driver && ../../bin/helm template kustomize . -s templates/csidriver.yaml > ../../deploy/kubernetes/base/csidriver.yaml
	cd charts/aws-fsx-csi-driver && ../../bin/helm template kustomize . -s templates/node-daemonset.yaml > ../../deploy/kubernetes/base/node-daemonset.yaml
	cd charts/aws-fsx-csi-driver && ../../bin/helm template kustomize . -s templates/node-serviceaccount.yaml > ../../deploy/kubernetes/base/node-serviceaccount.yaml
	cd charts/aws-fsx-csi-driver && ../../bin/helm template kustomize . -s templates/controller-deployment.yaml > ../../deploy/kubernetes/base/controller-deployment.yaml
	cd charts/aws-fsx-csi-driver && ../../bin/helm template kustomize . -s templates/controller-serviceaccount.yaml > ../../deploy/kubernetes/base/controller-serviceaccount.yaml
