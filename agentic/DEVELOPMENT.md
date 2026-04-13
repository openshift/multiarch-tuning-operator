# Development Guide

## Prerequisites

- **Go**: 1.22+
- **Docker or Podman**: For containerized builds
- **make**: GNU Make
- **gpgme-devel** (RHEL/Fedora) or **libgpgme-dev** (Debian/Ubuntu): Required for containers/image library
- **OpenShift or Kubernetes cluster**: For E2E testing
- **operator-sdk**: Optional, for operator lifecycle management

## Initial Setup

1. Clone repository
```bash
git clone https://github.com/openshift/multiarch-tuning-operator.git
cd multiarch-tuning-operator
```

2. Install dependencies
```bash
# Dependencies are vendored
make vendor
```

3. Build
```bash
# Local build (requires CGO, gpgme-devel/libgpgme-dev)
make build

# Containerized build (uses BUILD_IMAGE)
make docker-build IMG=quay.io/myrepo/multiarch-tuning-operator:latest
```

## Development Workflow

### Making Changes

1. Create a feature branch
2. Make your changes
3. Run tests locally: `make test`
4. Commit and push
5. Create pull request

### Running Tests

```bash
# All checks and tests (recommended before PR)
make test

# Unit tests only
make unit

# Specific test
GINKGO_ARGS="-v --focus='pod reconciler'" make unit

# E2E tests (requires deployed operator)
KUBECONFIG=/path/to/kubeconfig NAMESPACE=openshift-multiarch-tuning-operator make e2e

# Individual checks
make lint        # golangci-lint
make gosec       # Security analysis
make vet         # go vet
make goimports   # goimports check
make fmt         # gofmt
```

### Running Tests Locally vs Containerized

By default, tests run in containerized environment using BUILD_IMAGE.

To run locally:
```bash
# One-time
NO_DOCKER=1 make test

# Persistent via .env file
echo "NO_DOCKER=1" > .env
make test
```

See dotenv.example for other configuration options.

### Local Testing

Deploy operator to local cluster:

```bash
# Build image
make docker-build IMG=quay.io/myrepo/multiarch-tuning-operator:dev

# Push to registry
make docker-push IMG=quay.io/myrepo/multiarch-tuning-operator:dev

# Install CRDs
make install

# Deploy operator
make deploy IMG=quay.io/myrepo/multiarch-tuning-operator:dev

# Create ClusterPodPlacementConfig
kubectl create -f - <<EOF
apiVersion: multiarch.openshift.io/v1beta1
kind: ClusterPodPlacementConfig
metadata:
  name: cluster
spec:
  logVerbosityLevel: Debug
  namespaceSelector:
    matchExpressions:
      - key: multiarch.openshift.io/exclude-pod-placement
        operator: DoesNotExist
EOF

# Watch operator logs
kubectl logs -f -n openshift-multiarch-tuning-operator deployment/multiarch-tuning-operator
```

## Debugging

### Debugging Operator Controller
```bash
# Check operator logs
kubectl logs -n openshift-multiarch-tuning-operator deployment/multiarch-tuning-operator

# Check CPPC status
kubectl get clusterpodplacementconfig cluster -o yaml

# Check events
kubectl get events -n openshift-multiarch-tuning-operator --sort-by='.lastTimestamp'
```

### Debugging Pod Placement
```bash
# Check pod for scheduling gate
kubectl get pod <pod-name> -o jsonpath='{.spec.schedulingGates}'

# Check pod nodeAffinity
kubectl get pod <pod-name> -o jsonpath='{.spec.affinity.nodeAffinity}'

# Check pod placement controller logs
kubectl logs -n openshift-multiarch-tuning-operator deployment/pod-placement-controller

# Check webhook logs
kubectl logs -n openshift-multiarch-tuning-operator deployment/pod-placement-webhook
```

### Common Issues

**Issue**: Build fails with "gpgme.h: No such file or directory"
**Cause**: Missing gpgme development headers
**Fix**: Install gpgme-devel (RHEL/Fedora) or libgpgme-dev (Debian/Ubuntu)

**Issue**: Image inspection fails with "unauthorized"
**Cause**: Missing or invalid pull secret
**Fix**: Verify pull-secret synced to operator namespace: `kubectl get secret -n openshift-multiarch-tuning-operator pull-secret`

**Issue**: E2E tests fail with "context deadline exceeded"
**Cause**: Tests timeout waiting for resources
**Fix**: Check cluster resources, increase timeout in test code

## Code Organization

```
cmd/                    # Entry points
├── main-binary/        # Operator, controllers, webhook (multi-mode)
└── enoexec-daemon/     # eBPF monitoring daemon

pkg/                    # Libraries
├── controllers/        # Controller implementations
├── image/              # Image inspection
├── informers/          # CPPC singleton cache
└── utils/              # Shared utilities

apis/multiarch/         # CRD definitions
├── v1alpha1/           # Alpha API with conversion
└── v1beta1/            # Beta API (storage version)

test/e2e/               # End-to-end tests
```

See [ARCHITECTURE.md](../ARCHITECTURE.md) for details.

## Making a Pull Request

1. Ensure all tests pass: `make test`
2. Update documentation if needed
3. Create PR with description referencing issue
4. Address review feedback
5. Squash commits if requested

See [CONTRIBUTING.md](../CONTRIBUTING.md) for full process.

## Useful Commands

```bash
# Generate CRDs and manifests
make manifests

# Generate DeepCopy implementations
make generate

# Build multi-arch image
make docker-buildx IMG=quay.io/myrepo/multiarch-tuning-operator:latest

# Generate bundle
make bundle VERSION=1.0.0

# Undeploy operator
make undeploy

# Uninstall CRDs
make uninstall
```

## Related Documentation

- [ARCHITECTURE.md](../ARCHITECTURE.md) - System structure
- [TESTING.md](./TESTING.md) - Test strategy
- [Core Beliefs](./design-docs/core-beliefs.md) - Coding principles
