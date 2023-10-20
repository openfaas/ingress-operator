TAG?=latest
SERVER?=ghcr.io
OWNER?=openfaas
IMAGE := ingress-operator

.GIT_COMMIT=$(shell git rev-parse HEAD)
.GIT_VERSION=$(shell git describe --tags 2>/dev/null || echo "$(.GIT_COMMIT)")
.GIT_UNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
ifneq ($(.GIT_UNTRACKEDCHANGES),)
	.GIT_COMMIT := $(.GIT_COMMIT)-dirty
endif

# docker manifest command will work with Docker CLI 18.03 or newer
# but for now it's still experimental feature so we need to enable that
export DOCKER_CLI_EXPERIMENTAL=enabled

TOOLS_DIR := .tools

GOPATH := $(shell go env GOPATH)
CODEGEN_VERSION := $(shell hack/print-codegen-version.sh)
CODEGEN_PKG := $(GOPATH)/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION}

ARCH?=linux/amd64
MULTIARCH?=linux/amd64,linux/arm/v7,linux/arm64

$(TOOLS_DIR)/code-generator.mod: go.mod
	@echo "syncing code-generator tooling version"
	@cd $(TOOLS_DIR) && go mod edit -require "k8s.io/code-generator@${CODEGEN_VERSION}"

${CODEGEN_PKG}: $(TOOLS_DIR)/code-generator.mod
	@echo "(re)installing k8s.io/code-generator-${CODEGEN_VERSION}"
	@cd $(TOOLS_DIR) && go mod download -modfile=code-generator.mod

.PHONY: build
build:
	@echo "building  $(SERVER)/$(OWNER)/$(IMAGE):$(TAG)"
	@docker build \
	--build-arg VERSION=$(.GIT_VERSION) \
	--build-arg GIT_COMMIT=$(.GIT_COMMIT) \
	-t  $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) .

.PHONY: build-buildx
build-buildx:
	@echo  $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) && \
	docker buildx create --use --name=multiarch --node=multiarch && \
	docker buildx build \
		--push \
		--platform $(ARCH) \
		--build-arg VERSION=$(.GIT_VERSION) \
		--build-arg GIT_COMMIT=$(.GIT_COMMIT) \
		--tag  $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) \
		.

.PHONY: build-buildx-all
build-buildx-all:
	@echo  "build $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) for $(MULTIARCH)"
	@docker buildx create --use --name=multiarch --node=multiarch && \
	docker buildx build \
		--platform $(MULTIARCH) \
		--output "type=image,push=false" \
		--build-arg VERSION=$(.GIT_VERSION) \
		--build-arg GIT_COMMIT=$(.GIT_COMMIT) \
		--tag $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) \
		.

.PHONY: publish-buildx-all
publish-buildx-all:
	@echo  "build and publish $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) for $(MULTIARCH)"
	@docker buildx create --use --name=multiarch --node=multiarch && \
	docker buildx build \
		--platform $(MULTIARCH) \
		--push=true \
		--build-arg VERSION=$(.GIT_VERSION) \
		--build-arg GIT_COMMIT=$(.GIT_COMMIT) \
		--tag $(SERVER)/$(OWNER)/$(IMAGE):$(TAG) \
		.

.PHONY: test
test:
	go test -v ./...

.PHONY: verify-codegen
verify-codegen: ${CODEGEN_PKG}
	./hack/verify-codegen.sh

.PHONY: update-codegen
update-codegen: ${CODEGEN_PKG}
	./hack/update-codegen.sh

.PHONY: charts
charts:
	cd chart && helm package ingress-operator/
	mv chart/*.tgz docs/
	helm repo index docs --url https://openfaas.github.io/ingress-operator/ --merge ./docs/index.yaml
