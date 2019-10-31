DOCKER_REPOSITORY=openfaas/ingress-operator
BUILD_ARGS=
ARCHS=amd64 arm32v6 arm32v7 arm64 ppc64le
TAG?=latest

# docker manifest command will work with Docker CLI 18.03 or newer
# but for now it's still experimental feature so we need to enable that
export DOCKER_CLI_EXPERIMENTAL=enabled

.PHONY: build
build: $(addprefix build-,$(ARCHS))  ## Build Docker images for all architectures 

.PHONY: build-%
build-%:
	docker build $(BUILD_ARGS) --build-arg arch="$*" --build-arg go_opts="GOARCH=$*" -t $(DOCKER_REPOSITORY):$(TAG)-$* .

.PHONY: build-arm32v%
build-arm32v%:
	docker build $(BUILD_ARGS) --build-arg arch="arm32v$*" --build-arg go_opts="GOARCH=arm GOARM=$*" -t $(DOCKER_REPOSITORY):$(TAG)-arm32v$* .

.PHONY: build-arm64
build-arm64:
	docker build $(BUILD_ARGS) --build-arg arch="arm64v8" --build-arg go_opts="GOARCH=arm64" -t $(DOCKER_REPOSITORY):$(TAG)-arm64 .

.PHONY: push
push: $(addprefix push-,$(ARCHS)) ## Push Docker images for all architectures

.PHONY: push-%
push-%:
	docker push $(DOCKER_REPOSITORY):$(TAG)-$* 

.PHONY: manifest
manifest: ## Create and push Docker manifest to combine all architectures in multi-arch Docker image
	docker manifest create --amend $(DOCKER_REPOSITORY):$(TAG) $(addprefix $(DOCKER_REPOSITORY):$(TAG)-,$(ARCHS))
	$(MAKE) $(addprefix manifest-annotate-,$(ARCHS))
	docker manifest push -p $(DOCKER_REPOSITORY):$(TAG)

.PHONY: manifest-annotate-%
manifest-annotate-%:
	docker manifest annotate $(DOCKER_REPOSITORY):$(TAG) $(DOCKER_REPOSITORY):$(TAG)-$* --os linux --arch $*

.PHONY: manifest-annotate-arm32v%
manifest-annotate-arm32v%:
	docker manifest annotate $(DOCKER_REPOSITORY):$(TAG) $(DOCKER_REPOSITORY):$(TAG)-arm32v$* --os linux --arch arm --variant v$*

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: verify-codegen
verify-codegen: ## Verify generated code
	./hack/verify-codegen.sh

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show help
	@echo "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:"
	@grep -E '^[a-zA-Z_/%\-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
