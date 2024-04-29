#!/bin/bash

cat <<EOF >>bundle.Dockerfile

# Labels from hack/patch-bundle-dockerfile.sh
LABEL com.redhat.component="Multiarch Tuning Operator"
LABEL distribution-scope="public"
LABEL name="multiarch-tuning-operator-bundle"
LABEL release="0.9.0"
LABEL version="0.9.0"
LABEL url="https://github.com/openshift/multiarch-tuning-operator"
LABEL vendor="Red Hat, Inc."
LABEL description="The Multiarch Tuning Operator enhances the user experience for administrators of Openshift \\
                   clusters with multi-architecture compute nodes or Site Reliability Engineers willing to \\
                   migrate from single-arch to multi-arch OpenShift"
LABEL io.k8s.description="The Multiarch Tuning Operator enhances the user experience for administrators of Openshift \\
                   clusters with multi-architecture compute nodes or Site Reliability Engineers willing to \\
                   migrate from single-arch to multi-arch OpenShift"
LABEL summary="The Multiarch Tuning Operator enhances the user experience for administrators of Openshift \\
                   clusters with multi-architecture compute nodes or Site Reliability Engineers willing to \\
                   migrate from single-arch to multi-arch OpenShift"
LABEL io.k8s.display-name="Multiarch Tuning Operator"
LABEL io.openshift.tags="openshift,operator,multiarch,scheduling"
EOF
