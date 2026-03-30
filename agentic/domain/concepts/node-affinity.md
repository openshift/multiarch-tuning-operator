---
concept: NodeAffinity
type: Kubernetes Concept
related: [SchedulingGate, ImageInspection, PodPlacement]
---

# NodeAffinity

## Definition

Kubernetes scheduling constraint that limits which nodes a pod can be scheduled on based on node labels, used by this operator to match pods to nodes with compatible CPU architectures.

## Purpose

Ensures pods are only scheduled on nodes with architectures supported by their container images, preventing exec format errors.

## Location in Code

- **Computation**: controllers/podplacement/pod_model.go:computeNodeAffinity()
- **Application**: controllers/podplacement/pod_model.go:setNodeAffinityForArchAndSchedulingGateRemoval()
- **Plugin integration**: apis/multiarch/common/plugins/nodeaffinityscoring_plugin.go
- **Tests**: controllers/podplacement/pod_model_test.go

## Lifecycle

```
1. Image inspection determines supported architectures (e.g., ["amd64", "arm64"])
2. Pod has existing nodeAffinity (or none)
3. Operator computes required nodeAffinity for kubernetes.io/arch
4. Merge with existing affinity (preserving user constraints)
5. Apply to pod.spec.affinity.nodeAffinity
6. Remove scheduling gate
7. Scheduler honors nodeAffinity when placing pod
```

## Key Fields / Properties

### pod.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
**Type**: NodeSelector
**Purpose**: Hard constraint - pod MUST match
**Example**:
```yaml
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
              - key: kubernetes.io/arch
                operator: In
                values:
                  - amd64
                  - arm64
```

### pod.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution
**Type**: []PreferredSchedulingTerm
**Purpose**: Soft preference - scheduler tries to honor
**Example**:
```yaml
spec:
  affinity:
    nodeAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          preference:
            matchExpressions:
              - key: kubernetes.io/arch
                operator: In
                values:
                  - amd64
```

## Common Patterns

### Required NodeAffinity (Default)
```go
requirement := corev1.NodeSelectorRequirement{
    Key:      "kubernetes.io/arch",
    Operator: corev1.NodeSelectorOpIn,
    Values:   supportedArchs, // ["amd64", "arm64"]
}
```

**When to use**: Always applied to ensure compatibility

### Preferred NodeAffinity (Plugin)
```go
// NodeAffinityScoring plugin adds preferences based on cluster distribution
weight := computeWeightBasedOnNodeDistribution(arch)
preferred := corev1.PreferredSchedulingTerm{
    Weight: weight,
    Preference: corev1.NodeSelectorTerm{...},
}
```

**When to use**: When NodeAffinityScoring plugin is enabled in CPPC

## Related Concepts

- [SchedulingGate](./scheduling-gate.md) - Applied before removing gate
- [ImageInspection](./image-inspection.md) - Provides architecture list
- [NodeAffinityScoring](./node-affinity-scoring.md) - Plugin for preferred scheduling

## Implementation Details

- **Logic**: controllers/podplacement/pod_model.go:computeNodeAffinity()
- **Merging**: Combines with existing user-defined nodeAffinity
- **Validation**: Kubernetes API server validates affinity syntax

## Edge Cases

### User-Defined Affinity Conflicts
If user already specified kubernetes.io/arch with conflicting values, operator merges using AND logic. Result may be unsatisfiable.

**Example**:
```yaml
# User wants only amd64, but image supports amd64+arm64
# Operator adds: values: [amd64, arm64]
# Result: Both constraints apply (intersection = amd64)
```

### No Supported Architectures
If image inspection returns empty architecture list, pod is ungated without modification to allow manual intervention.

## References

- [Kubernetes Affinity Documentation](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)
- [KEP-3838: Pod Mutable Scheduling Directives](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/3838-pod-mutable-scheduling-directives)
