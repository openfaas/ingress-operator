.PHONY: build push manifest test verify-codegen charts
TAG?=latest

# docker manifest command will work with Docker CLI 18.03 or newer
# but for now it's still experimental feature so we need to enable that
export DOCKER_CLI_EXPERIMENTAL=enabled

TOOLS_DIR := .tools

GOPATH := $(shell go env GOPATH)
CODEGEN_VERSION := $(shell hack/print-codegen-version.sh)
CODEGEN_PKG := $(GOPATH)/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION}

$(TOOLS_DIR)/code-generator.mod: go.mod
	@echo "syncing code-generator tooling version"
	@cd $(TOOLS_DIR) && go mod edit -require "k8s.io/code-generator@${CODEGEN_VERSION}"

${CODEGEN_PKG}: $(TOOLS_DIR)/code-generator.mod
	@echo "(re)installing k8s.io/code-generator-${CODEGEN_VERSION}"
	@cd $(TOOLS_DIR) && go mod download -modfile=code-generator.mod

build:
	docker build -t ghcr.io/openfaas/ingress-operator:$(TAG)-amd64 . -f Dockerfile
	docker build --build-arg OPTS="GOARCH=arm64" -t ghcr.io/openfaas/ingress-operator:$(TAG)-arm64 . -f Dockerfile
	docker build --build-arg OPTS="GOARCH=arm GOARM=6" -t ghcr.io/openfaas/ingress-operator:$(TAG)-armhf . -f Dockerfile

push:
	docker push ghcr.io/openfaas/ingress-operator:$(TAG)-amd64
	docker push ghcr.io/openfaas/ingress-operator:$(TAG)-arm64
	docker push ghcr.io/openfaas/ingress-operator:$(TAG)-armhf

manifest:
	docker manifest create --amend ghcr.io/openfaas/ingress-operator:$(TAG) \
		ghcr.io/openfaas/ingress-operator:$(TAG)-amd64 \
		ghcr.io/openfaas/ingress-operator:$(TAG)-arm64 \
		ghcr.io/openfaas/ingress-operator:$(TAG)-armhf
	docker manifest annotate ghcr.io/openfaas/ingress-operator:$(TAG) ghcr.io/openfaas/ingress-operator:$(TAG)-arm64 --os linux --arch arm64
	docker manifest annotate ghcr.io/openfaas/ingress-operator:$(TAG) ghcr.io/openfaas/ingress-operator:$(TAG)-armhf --os linux --arch arm --variant v6
	docker manifest push -p ghcr.io/openfaas/ingress-operator:$(TAG)

test:
	go test -v ./...

verify-codegen: ${CODEGEN_PKG}
	./hack/verify-codegen.sh

update-codegen: ${CODEGEN_PKG}
	./hack/update-codegen.sh

charts:
	cd chart && helm package ingress-operator/
	mv chart/*.tgz docs/
	helm repo index docs --url https://openfaas-incubator.github.io/ingress-operator/ --merge ./docs/index.yaml
