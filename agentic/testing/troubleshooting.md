# Test Troubleshooting Guide

## Flaky Tests

**Symptom**: Tests pass/fail non-deterministically

**Common causes**:
- Race conditions in async code
- Timeouts too short for slow environments
- Shared state between tests

**Fix**:
- Add proper synchronization (Eventually/Consistently)
- Increase timeouts
- Ensure test isolation (separate namespaces, cleanup)

## Timeout Issues

**Symptom**: "context deadline exceeded" errors

**Common causes**:
- envtest API server slow to start
- E2E cluster resources unavailable
- Controllers not reconciling

**Fix**:
- Increase timeout in Eventually() calls
- Check cluster resource availability
- Verify controller logs for errors

## Image Inspection Failures in Tests

**Symptom**: Tests fail with registry errors

**Common causes**:
- Network issues reaching real registries
- Missing mock image data

**Fix**:
- Use mock image inspector from pkg/testing/image/
- Don't call real registries in unit tests
- Use fixtures for expected responses

## Test Configuration Issues

**Environment Variables**:
- `NO_DOCKER=1` - Run tests locally (not in container)
- `KUBECONFIG` - Path to kubeconfig for E2E tests
- `NAMESPACE` - Operator namespace for E2E tests
- `GINKGO_ARGS` - Additional Ginkgo flags

**Config Files**:
- `.env` - Local test configuration (see dotenv.example)
- `.ginkgo.yml` - Ginkgo configuration (if exists)

## Test Data and Fixtures

**Location**: `pkg/testing/fixtures/`
**Format**: YAML manifests, JSON image manifests

**Examples**:
- `pkg/testing/fixtures/pod.yaml` - Sample pod definitions
- `pkg/testing/fixtures/image-manifest.json` - Mock image manifest lists

## Related

- [Testing Strategy](../TESTING.md) - Main testing guide
- [Development Guide](../DEVELOPMENT.md) - Dev setup
