FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_1.22 as builder
ARG IMG=registry.redhat.io/multiarch-tuning/multiarch-tuning-rhel9-operator@sha256:8638a33707cafe5177093dd7aab9e7e045c913613249f887c38977db9bae1782
ARG ORIGINAL_IMG=registry.ci.openshift.org/origin/multiarch-tuning-operator:main
WORKDIR /code
COPY ./ ./

# Replace the bundle image in the repository with the one specified by the IMG build argument.
RUN chmod -R g+rwX ./ && find bundle/ && find bundle -type f -exec sed -i \
    "s|${ORIGINAL_IMG}|${IMG}|g" {} \+; \
    grep -rq "${ORIGINAL_IMG}" bundle/ && \
    { echo "Failed to replace image references"; exit 1; } || echo "Image references replaced" && \
    grep -r "${IMG}" bundle/

FROM scratch
# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=multiarch-tuning-operator
LABEL operators.operatorframework.io.bundle.channels.v1=tech-preview
LABEL operators.operatorframework.io.bundle.channel.default.v1=tech-preview
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.31.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v3

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Copy files to locations specified by labels.
COPY --from=builder /code/bundle/manifests /manifests/
COPY --from=builder /code/bundle/metadata /metadata/
COPY --from=builder /code/bundle/tests/scorecard /tests/scorecard/

# Labels from hack/patch-bundle-dockerfile.sh
LABEL com.redhat.component="Multiarch Tuning Operator"
LABEL distribution-scope="public"
LABEL name="multiarch-tuning-operator-bundle"
LABEL release="1.0.0"
LABEL version="1.0.0"
LABEL url="https://github.com/openshift/multiarch-tuning-operator"
LABEL vendor="Red Hat, Inc."
LABEL description="The Multiarch Tuning Operator enhances the user experience for administrators of Openshift \
                   clusters with multi-architecture compute nodes or Site Reliability Engineers willing to \
                   migrate from single-arch to multi-arch OpenShift"
LABEL io.k8s.description="The Multiarch Tuning Operator enhances the user experience for administrators of Openshift \
                   clusters with multi-architecture compute nodes or Site Reliability Engineers willing to \
                   migrate from single-arch to multi-arch OpenShift"
LABEL summary="The Multiarch Tuning Operator enhances the user experience for administrators of Openshift \
                   clusters with multi-architecture compute nodes or Site Reliability Engineers willing to \
                   migrate from single-arch to multi-arch OpenShift"
LABEL io.k8s.display-name="Multiarch Tuning Operator"
LABEL io.openshift.tags="openshift,operator,multiarch,scheduling"
