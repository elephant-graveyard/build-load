#!/usr/bin/env bash

# Copyright © 2022 The Homeport Team
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

set -euo pipefail

if [[ "$(kubectl config current-context)" != "kind-kind" ]]; then
  echo "Error: Install script is only allowed for KinD clusters with the name kind."
  exit 1
fi

BASEDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
TEKTON_VERSION="${TEKTON_VERSION:-$(grep "github.com/tektoncd/pipeline" "$BASEDIR/go.mod" | cut -d' ' -f2)}"

echo -e "\n\033[93mDeploying Tekton Pipelines '${TEKTON_VERSION}'\033[0m"
kubectl apply --filename "https://github.com/tektoncd/pipeline/releases/download/${TEKTON_VERSION}/release.yaml"

echo -e "\n\033[93mWaiting for deployment to get ready ...\033[0m"
kubectl --namespace=tekton-pipelines rollout status deployment tekton-pipelines-controller --timeout=1m
kubectl --namespace=tekton-pipelines rollout status deployment tekton-pipelines-webhook --timeout=1m
