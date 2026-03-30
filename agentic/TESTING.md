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

### E2E Tests

**Location**: `test/e2e/*/`
**Run**: `KUBECONFIG=/path/to/kubeconfig NAMESPACE=openshift-multiarch-tuning-operator make e2e`
**Framework**: Ginkgo/Gomega

**Suites**:
- `test/e2e/e2e_test.go` - Operator lifecycle tests
- `test/e2e/pod-placement/` - Pod placement workflow tests

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
2. **Fix the bug** - Modify code to make test pass
3. **Verify test passes** - Run test suite: `make test`

## Running Tests

```bash
# All tests (lint, vet, gosec, goimports, unit)
make test

# Unit tests only
make unit

# Specific test by pattern
GINKGO_ARGS="-v --focus='should set nodeAffinity'" make unit

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
- [Test Troubleshooting](./testing/troubleshooting.md) - Debugging test failures
- [ARCHITECTURE.md](../ARCHITECTURE.md) - System structure
- [Test Helpers](../pkg/testing/README.md) - Using test utilities
