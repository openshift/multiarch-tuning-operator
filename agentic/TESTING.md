# Testing Strategy

## Test Pyramid

```
       /\
      /E2E\          Small number, full system, slow
     /------\
    / Integ  \       Medium number, component integration
   /----------\
  / Unit Tests \     Large number, fast, isolated
 /--------------\
```

## Test Organization

### Unit Tests

**Location**: `*_test.go` files alongside source code
**Run**: `make unit`
**Coverage Target**: >80%
**Framework**: Ginkgo/Gomega

**Pattern**:
```go
var _ = Describe("PodReconciler", func() {
    Context("when pod has scheduling gate", func() {
        It("should inspect images and set nodeAffinity", func() {
            // Arrange
            pod := builder.NewPod().WithSchedulingGate().Build()

            // Act
            result, err := reconciler.Reconcile(ctx, req)

            // Assert
            Expect(err).ToNot(HaveOccurred())
            Expect(pod.Spec.Affinity.NodeAffinity).ToNot(BeNil())
        })
    })
})
```

**Key Test Locations**:
- `controllers/podplacement/pod_reconciler_test.go` - Pod reconciliation logic
- `controllers/operator/clusterpodplacementconfig_controller_test.go` - Operator lifecycle
- `pkg/image/inspector_test.go` - Image inspection
- `apis/multiarch/v1beta1/*_webhook_test.go` - Webhook validation

**Test Helpers**:
- `pkg/testing/builder/` - Fluent builders for Kubernetes objects
- `pkg/testing/framework/` - Test framework utilities
- `pkg/testing/image/` - Mock image registry

### Integration Tests

**Location**: Included in unit test suite (use envtest)
**Run**: `make unit` (runs with unit tests)
**Framework**: controller-runtime envtest

**Purpose**: Test controller interactions with Kubernetes API (using fake API server)

**Example**:
```go
// envtest provides fake API server
testEnv := &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
}
cfg, err := testEnv.Start()

// Create manager with fake API server
mgr, err := ctrl.NewManager(cfg, ctrl.Options{})

// Test reconciler against fake API
reconciler := &PodReconciler{Client: mgr.GetClient()}
```

### E2E Tests

**Location**: `test/e2e/*/`
**Run**: `KUBECONFIG=/path/to/kubeconfig NAMESPACE=openshift-multiarch-tuning-operator make e2e`
**Framework**: Ginkgo/Gomega

**Suites**:
- `test/e2e/e2e_test.go` - Operator lifecycle tests
- `test/e2e/pod-placement/` - Pod placement workflow tests

**Purpose**: Test full system in real cluster

**Example Scenarios**:
- Deploy operator, create CPPC, verify operands deployed
- Create pod, verify scheduling gate added
- Verify pod nodeAffinity set based on image architectures
- Delete CPPC, verify pods ungated before operand deletion

## Writing Tests

### For New Features

1. **Write unit tests for new code**
   - Test each function/method independently
   - Mock external dependencies (image registry, etc.)
   - Use test helpers from pkg/testing/

2. **Add integration tests for component interactions**
   - Test controller reconciliation against fake API
   - Verify CRD updates, status conditions

3. **Add E2E tests for user-facing changes**
   - Test complete user workflows
   - Verify behavior in real cluster

### For Bug Fixes

1. **Write a failing test that reproduces the bug**
   - Demonstrate broken behavior
   - Make test as minimal as possible

2. **Fix the bug**
   - Modify code to make test pass

3. **Verify test passes**
   - Run test suite: `make test`

## Running Tests Locally

```bash
# All tests (unit + E2E if cluster available)
make test

# Unit tests only
make unit

# Specific test by pattern
GINKGO_ARGS="-v --focus='should set nodeAffinity'" make unit

# With coverage report
make unit
# Coverage report: test-unit-coverage.out

# E2E tests (requires deployed operator)
export KUBECONFIG=/path/to/kubeconfig
export NAMESPACE=openshift-multiarch-tuning-operator
make e2e

# Run specific E2E suite
GINKGO_ARGS="-v --focus='pod placement'" make e2e
```

## CI Test Execution

**GitHub Actions / CI Pipeline**:
- Triggered on pull requests and merges
- Runs: `make test` (lint, vet, gosec, goimports, unit tests)
- E2E tests run on merge to main (requires cluster)

**Pre-merge Checks**:
- All linters pass (golangci-lint, gosec)
- All unit tests pass
- Code coverage maintained (>80%)
- No new gosec warnings

## Test Data

**Location**: `pkg/testing/fixtures/`
**Format**: YAML manifests, JSON image manifests

**Examples**:
- `pkg/testing/fixtures/pod.yaml` - Sample pod definitions
- `pkg/testing/fixtures/image-manifest.json` - Mock image manifest lists

## Test Configuration

**Environment Variables**:
- `NO_DOCKER=1` - Run tests locally (not in container)
- `KUBECONFIG` - Path to kubeconfig for E2E tests
- `NAMESPACE` - Operator namespace for E2E tests
- `GINKGO_ARGS` - Additional Ginkgo flags

**Config Files**:
- `.env` - Local test configuration (see dotenv.example)
- `.ginkgo.yml` - Ginkgo configuration (if exists)

## Troubleshooting Test Failures

### Flaky tests
**Symptom**: Tests pass/fail non-deterministically
**Common causes**:
- Race conditions in async code
- Timeouts too short for slow environments
- Shared state between tests

**Fix**:
- Add proper synchronization (Eventually/Consistently)
- Increase timeouts
- Ensure test isolation (separate namespaces, cleanup)

### Timeout issues
**Symptom**: "context deadline exceeded" errors
**Common causes**:
- envtest API server slow to start
- E2E cluster resources unavailable
- Controllers not reconciling

**Fix**:
- Increase timeout in Eventually() calls
- Check cluster resource availability
- Verify controller logs for errors

### Image inspection failures in tests
**Symptom**: Tests fail with registry errors
**Common causes**:
- Network issues reaching real registries
- Missing mock image data

**Fix**:
- Use mock image inspector from pkg/testing/image/
- Don't call real registries in unit tests
- Use fixtures for expected responses

## Test Coverage

Current coverage targets:
- **Overall**: >80%
- **Controllers**: >85%
- **Core logic (pkg/image)**: >90%
- **Webhooks**: >80%

View coverage report:
```bash
make unit
go tool cover -html=test-unit-coverage.out
```

## Related Documentation

- [DEVELOPMENT.md](./DEVELOPMENT.md) - Dev setup and workflow
- [ARCHITECTURE.md](../ARCHITECTURE.md) - System structure
- [Test Helpers](../pkg/testing/README.md) - Using test utilities
