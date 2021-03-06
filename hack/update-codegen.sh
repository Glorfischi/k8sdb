#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
echo $SCRIPT_ROOT
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}


${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/glorfischi/k8sdb/pkg/client github.com/glorfischi/k8sdb/pkg/apis \
  k8sdb:v1alpha1 \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt
