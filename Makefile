.PHONY: build push manifest test verify-codegen charts
TAG?=latest

# docker manifest command will work with Docker CLI 18.03 or newer
# but for now it's still experimental feature so we need to enable that
export DOCKER_CLI_EXPERIMENTAL=enabled

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
	go test -mod=vendor -v ./...

verify-codegen:
	go get -u -d k8s.io/code-generator@v0.17.0
	./hack/verify-codegen.sh

charts:
	cd chart && helm package ingress-operator/
	mv chart/*.tgz docs/
	helm repo index docs --url https://openfaas-incubator.github.io/ingress-operator/ --merge ./docs/index.yaml

