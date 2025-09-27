#!/bin/bash
# Copyright 2018 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

# shellcheck disable=SC2128
REPO_ROOT=$(dirname "${BASH_SOURCE}")/..

OPENSHIFT_CI=${OPENSHIFT_CI:-""}
export ARTIFACT_DIR=${ARTIFACT_DIR:-$(mktemp -d)}
GINKGO=${GINKGO:-"go run ${REPO_ROOT}/vendor/github.com/onsi/ginkgo/v2/ginkgo"}
GINKGO_ARGS=${GINKGO_ARGS:-"-vv --randomize-all --randomize-suites -p -race -trace --keep-going --timeout=60m "}
TEST_LABEL=${TEST_LABEL:-"integration"}
GINKGO_ARGS="${GINKGO_ARGS} --label-filter ${TEST_LABEL}"
GINKGO_EXTRA_ARGS=${GINKGO_EXTRA_ARGS:-""}
SKIP_COVERAGE=${SKIP_COVERAGE:-"false"}

# Ensure that some home var is set and that it's not the root.
# This is required for the kubebuilder cache.
export HOME=${HOME:=/tmp/kubebuilder-testing}
if [ $HOME == "/" ]; then
  export HOME=/tmp/kubebuilder-testing
fi

if [ "$OPENSHIFT_CI" == "true" ] && [ -n "$ARTIFACT_DIR" ] && [ -d "$ARTIFACT_DIR" ]; then # detect ci environment there
  GINKGO_ARGS="${GINKGO_ARGS} --junit-report=junit_multiarch_tuning_operator${JUNIT_SUFFIX:-}.xml --output-dir=${ARTIFACT_DIR}"
fi

if [ "$SKIP_COVERAGE" != "true" ] && [ -n "$ARTIFACT_DIR" ] && [ -d "$ARTIFACT_DIR" ]; then
  GINKGO_ARGS="${GINKGO_ARGS} --cover --coverprofile=test-unit-coverage.out"
fi

if [ "${TEST_LABEL}" == "e2e" ]; then
  export NAMESPACE="openshift-multiarch-tuning-operator"
fi

# Print the command we are going to run as Make would.
echo "${GINKGO} ${GINKGO_ARGS} ${GINKGO_EXTRA_ARGS} ./..."
# shellcheck disable=SC2086
${GINKGO} ${GINKGO_ARGS} ${GINKGO_EXTRA_ARGS} ./...
# Capture the test result to exit on error after coverage.
TEST_RESULT=$?

if [ -f "${ARTIFACT_DIR}/test-unit-coverage.out" ]; then
  # Convert the coverage to html for spyglass.
  go tool cover -html="${ARTIFACT_DIR}/test-unit-coverage.out" -o "${ARTIFACT_DIR}/test-unit-coverage.html"

  # Report the coverage at the end of the test output.
  echo -n "Coverage "
  go tool cover -func="${ARTIFACT_DIR}/test-unit-coverage.out" | tail -n 1
  # Blank new line after the coverage output to make it easier to read when there is an error.
  echo
fi

# Ensure we exit based on the test result, coverage results are supplementary.
exit ${TEST_RESULT}
