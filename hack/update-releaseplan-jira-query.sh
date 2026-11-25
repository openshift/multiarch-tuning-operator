#!/bin/bash
# Updates the Jira collector fixVersion on a ReleasePlan in multiarch-tuning-ope-tenant

set -euo pipefail

FIX_VERSION="${1:?Usage: $0 <fix-version, e.g. mto-1.3.2>}"

if [[ ! "${FIX_VERSION}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "Error: fix-version contains invalid characters: ${FIX_VERSION}" >&2
  exit 1
fi

NAMESPACE="multiarch-tuning-ope-tenant"
RELEASEPLAN="multiarch-tuning-operator-v1-x-release-as-operator"
PROJECT="MULTIARCH"
COMPONENT="Multiarch-Tuning-Operator"

QUERY="project = \"${PROJECT}\" AND fixVersion = \"${FIX_VERSION}\" AND component = \"${COMPONENT}\""

echo "Patching ${RELEASEPLAN} in ${NAMESPACE}"
echo "  fixVersion -> ${FIX_VERSION}"

oc patch releaseplan "${RELEASEPLAN}" \
  -n "${NAMESPACE}" \
  --type=json \
  -p "[{\"op\":\"replace\",\"path\":\"/spec/collectors/items/0/params/1/value\",\"value\":\"${QUERY}\"}]"

echo "Done. Verifying:"
ACTUAL=$(oc get releaseplan "${RELEASEPLAN}" -n "${NAMESPACE}" -o jsonpath='{.spec.collectors.items[0].params[1].value}')
echo "${ACTUAL}"
if [[ "${ACTUAL}" != "${QUERY}" ]]; then
  echo "Error: patched value does not match expected query" >&2
  exit 1
fi