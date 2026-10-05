ARG BUILD_IMAGE=registry.access.redhat.com/ubi9/go-toolset:1.26
ARG RUNTIME_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:latest
FROM ${BUILD_IMAGE} as builder
ARG TARGETOS
ARG TARGETARCH

# Switch to root to install gpgme-devel, which is required for CGO compilation of the
# containers/image library used for registry authentication and image inspection.
# This only affects the builder stage (used during compilation) and does not impact the
# security of the final runtime image, which runs as USER 65532:65532 (non-root).
USER 0
RUN if ! pkg-config --exists gpgme 2>/dev/null; then \
        if which apt-get; then apt-get update && apt-get install -y libgpgme-dev && apt-get -y clean autoclean; \
        elif which dnf; then dnf install -y gpgme-devel && dnf clean all -y; fi; \
    fi

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY vendor/ vendor/

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN GOGC=75 CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go
RUN GOGC=75 CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o enoexec-daemon cmd/enoexec-daemon/main.go

# Trimmer stage: use the runtime image (which has ldd) to discover shared library
# dependencies and assemble a minimal root filesystem.
FROM ${RUNTIME_IMAGE} as trimmer
COPY --from=builder /workspace/manager /workspace/enoexec-daemon /usr/local/bin/
RUN mkdir -p /runtime/usr/local/bin /runtime/etc/pki/tls/certs /runtime/etc /runtime/lib64 /runtime/lib && \
    cp /usr/local/bin/manager /usr/local/bin/enoexec-daemon /runtime/usr/local/bin/ && \
    for bin in /usr/local/bin/manager /usr/local/bin/enoexec-daemon; do \
        ldd "$bin" 2>/dev/null | grep -oP '(?<==> )\S+' | while read lib; do \
            dir="/runtime$(dirname "$lib")" && \
            mkdir -p "$dir" && \
            cp -L "$lib" "$dir/"; \
        done; \
    done && \
    cp -rL /etc/pki/tls/certs/ca-bundle.crt /runtime/etc/pki/tls/certs/ && \
    if [ -d /usr/share/zoneinfo ]; then \
        mkdir -p /runtime/usr/share && \
        cp -rL /usr/share/zoneinfo /runtime/usr/share/zoneinfo; \
    fi && \
    echo "65532:x:65532:65532:nonroot:/:" > /runtime/etc/passwd && \
    echo "65532:x:65532:" > /runtime/etc/group && \
    cp -L /lib64/ld-linux-*.so.* /runtime/lib64/ 2>/dev/null; \
    cp -L /lib/ld-linux-*.so.* /runtime/lib/ 2>/dev/null; \
    true

# Final minimal image with no shell, no package manager, no unnecessary files.
FROM scratch
COPY --from=trimmer /runtime/ /
WORKDIR /
USER 65532:65532
LABEL com.redhat.component="Multiarch Tuning Operator"
LABEL distribution-scope="public"
LABEL name="multiarch-tuning/multiarch-tuning-operator"
LABEL release="1.3.4"
LABEL version="1.3.4"
LABEL cpe="cpe:/a:redhat:multiarch_tuning_operator:1.3::el9"
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

ENTRYPOINT ["/usr/local/bin/manager"]
