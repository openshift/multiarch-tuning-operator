ARG RUNTIME_IMAGE=quay.io/centos/centos:stream9-minimal
FROM golang:1.23 as builder
ARG TARGETOS
ARG TARGETARCH

RUN if which apt-get; then apt-get update && apt-get install -y libgpgme-dev && apt-get -y clean autoclean; \
    elif which dnf; then dnf install -y gpgme-devel && dnf clean all -y; fi;

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY vendor/ vendor/
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main-binary/main.go
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o enoexec-daemon cmd/enoexec-daemon/main.go

# Stage 2: Extract minimal runtime dependencies
# This stage collects only the essential libraries and files needed to run the operator binaries
FROM ${RUNTIME_IMAGE} as runtime-deps
ARG TARGETARCH

# Create the directory structure for the minimal runtime
RUN mkdir -p /runtime-root/lib64 \
    /runtime-root/etc/ssl/certs \
    /runtime-root/etc/pki/tls/certs \
    /runtime-root/usr/share/pki/ca-trust-source/anchors

# Copy essential runtime libraries
# For manager binary (includes image inspection via containers/image):
#   - ld-linux-*.so.2: Dynamic linker/loader
#   - libc.so.6: GNU C Library
#   - libgpgme.so.11: GPGME library for container registry authentication
#   - libassuan.so.0: Dependency of libgpgme (IPC library)
#   - libgpg-error.so.0: Dependency of libgpgme (error handling)
#   - libresolv.so.2: DNS resolver library
# For enoexec-daemon binary:
#   - Subset of above (no gpgme dependencies needed)
RUN cp -P /lib64/ld-linux-*.so.2 /runtime-root/lib64/ || true && \
    cp -P /lib64/libc.so.6 /runtime-root/lib64/ && \
    cp -P /lib64/libc-*.so /runtime-root/lib64/ || true && \
    cp -P /lib64/libgpgme.so.11* /runtime-root/lib64/ && \
    cp -P /lib64/libassuan.so.0* /runtime-root/lib64/ && \
    cp -P /lib64/libgpg-error.so.0* /runtime-root/lib64/ && \
    cp -P /lib64/libresolv.so.2 /runtime-root/lib64/ && \
    cp -P /lib64/libresolv-*.so /runtime-root/lib64/ || true

# Copy CA certificates for TLS connections to container registries
# Container image inspection requires TLS verification
RUN if [ -d /etc/ssl/certs ]; then cp -r /etc/ssl/certs/* /runtime-root/etc/ssl/certs/ 2>/dev/null || true; fi && \
    if [ -d /etc/pki/tls/certs ]; then cp -r /etc/pki/tls/certs/* /runtime-root/etc/pki/tls/certs/ 2>/dev/null || true; fi && \
    if [ -d /usr/share/pki/ca-trust-source ]; then cp -r /usr/share/pki/ca-trust-source/* /runtime-root/usr/share/pki/ca-trust-source/anchors/ 2>/dev/null || true; fi

# Create minimal passwd and group files for non-root user (65532)
# The operator runs as non-root for security hardening
RUN echo "nonroot:x:65532:65532:nonroot:/:/sbin/nologin" > /runtime-root/etc/passwd && \
    echo "nonroot:x:65532:" > /runtime-root/etc/group

# Stage 3: Final minimal runtime image
# Using scratch (empty base image) for maximum security hardening:
#   - No shell (prevents shell-based exploitation)
#   - No package manager (prevents runtime package installation)
#   - No utilities (reduced attack surface)
#   - Only operator binaries and essential libraries
FROM scratch

LABEL com.redhat.component="Multiarch Tuning Operator"
LABEL distribution-scope="public"
LABEL name="multiarch-tuning/multiarch-tuning-operator-bundle"
LABEL release="1.2.1"
LABEL version="1.2.1"
LABEL cpe="cpe:/a:redhat:multiarch_tuning_operator:1.1::el9"
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

WORKDIR /
# Copy minimal runtime dependencies from runtime-deps stage
COPY --from=runtime-deps /runtime-root/ /
# Copy operator binaries from builder stage
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/enoexec-daemon .

# Run as non-root user for security hardening
USER 65532:65532

ENTRYPOINT ["/manager"]
