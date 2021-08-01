#!/bin/bash

set -e
set -o pipefail

controllergen="$(go env GOBIN)/controller-gen"
PKG=sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2

if [ ! -e "$controllergen" ]
then
echo "Getting $PKG"
    go install $PKG
fi

echo "using $controllergen"

"$controllergen" \
  crd \
  schemapatch:manifests=./artifacts/crds \
  paths=./pkg/apis/... \
  output:dir=./artifacts/crds
