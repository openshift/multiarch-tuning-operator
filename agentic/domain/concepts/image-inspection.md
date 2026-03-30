---
concept: ImageInspection
type: Pattern
related: [ContainerRegistry, PullSecret, ManifestList]
---

# Image Inspection

## Definition

Process of retrieving container image manifests from container registries to determine which CPU architectures an image supports, enabling architecture-aware pod scheduling.

## Purpose

Allows the operator to automatically configure nodeAffinity based on actual image capabilities rather than requiring manual annotation.

## Location in Code

- **Inspector**: pkg/image/inspector.go
- **Authentication**: pkg/image/auth.go
- **Caching**: pkg/image/cache.go (not implemented yet, in memory only)
- **Metrics**: controllers/podplacement/metrics/metrics.go
- **Tests**: pkg/image/inspector_test.go

## Lifecycle

```
1. PodReconciler receives pod with scheduling gate
2. Extract image references from pod.spec.containers[*].image
3. Retrieve pull secrets from pod.spec.imagePullSecrets
4. For each image:
   a. Authenticate to registry using pull secret
   b. Fetch image manifest or manifest list
   c. Extract supported architectures
5. Compute intersection of supported architectures across all images
6. Return architecture list or error
```

## Key Fields / Properties

### Image Reference
**Type**: string
**Purpose**: Container image URL
**Example**:
```
registry.redhat.io/openshift4/ose-nginx:latest
quay.io/user/app:v1.0
```

### Manifest List (OCI Index)
**Type**: application/vnd.docker.distribution.manifest.list.v2+json
**Purpose**: Multi-architecture manifest containing per-arch image digests
**Example**:
```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "manifests": [
    {
      "platform": {"architecture": "amd64", "os": "linux"},
      "digest": "sha256:abc..."
    },
    {
      "platform": {"architecture": "arm64", "os": "linux"},
      "digest": "sha256:def..."
    }
  ]
}
```

## Common Patterns

### Inspecting Single Image
```go
systemContext := &types.SystemContext{
    AuthFilePath: "/path/to/pull-secret",
}

inspector := image.NewInspector(systemContext)
architectures, err := inspector.Inspect(ctx, imageReference)
if err != nil {
    // Handle error (may be transient registry issue)
}
// architectures = []string{"amd64", "arm64"}
```

**When to use**: Determining supported architectures for pod placement

### Handling Authentication
```go
// Pull secret synced from openshift-config/pull-secret
authFile := "/var/run/secrets/multiarch-tuning-operator/pull-secret/.dockerconfigjson"
systemContext := &types.SystemContext{
    AuthFilePath: authFile,
}
```

**When to use**: Accessing private registries

## Related Concepts

- [SchedulingGate](./scheduling-gate.md) - Pod waits during inspection
- [NodeAffinity](./node-affinity.md) - Set based on inspection results
- [PullSecret](./pull-secret.md) - Required for private registry access

## Implementation Details

- **Logic**: pkg/image/inspector.go:Inspect()
- **Caching**: In-memory only (no persistent cache)
- **Metrics**: `mto_ppo_ctrl_time_to_inspect_image_seconds` histogram

## Performance Considerations

- **I/O bound**: High concurrency (NumCPU * 4) to handle parallel inspections
- **Network calls**: Can be slow, especially for remote registries
- **Caching**: Manifest results cached in memory to reduce registry API calls
- **Timeouts**: Context timeouts prevent indefinite waits

## Error Handling

- **Transient errors**: Retry via controller requeue
- **Max retries**: After max attempts, pod ungated without modification
- **Metric tracking**: `mto_ppo_ctrl_failed_image_inspection_total` counter

## References

- [containers/image library](https://github.com/containers/image)
- [OCI Image Format Specification](https://github.com/opencontainers/image-spec)
- [Docker Manifest List](https://docs.docker.com/registry/spec/manifest-v2-2/)
