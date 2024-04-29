FROM quay.io/operator-framework/operator-sdk:v1.31.0 as osdk
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_1.21 as builder
ARG IMG=registry.redhat.io/multiarch-tuning/multiarch-tuning-rhel9-operator@sha256:6eb3e10671b6bb8f54139312e4acf291a7078e2912bec75105c84e65ab495460
COPY . /code
COPY --from=osdk /usr/local/bin/operator-sdk /usr/local/bin/
RUN chmod -R g+rwX /code
WORKDIR /code

# VERSION is set in the base image to the golang version. However, we want to default to the one set in the Makefile.
RUN unset VERSION; test -n "${IMG}" && make bundle IMG="${IMG}"

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
