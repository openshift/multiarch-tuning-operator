---
id: ADR-0003
title: Ordered Deletion During Deprovisioning
date: 2024-02-10
status: accepted
deciders: [openshift-multiarch-team]
supersedes: null
superseded-by: null
---

# Ordered Deletion During Deprovisioning

## Status

Accepted (implemented)

## Context

When ClusterPodPlacementConfig is deleted, the operator must clean up deployed operands (webhook, controller). However, if we delete operands immediately, gated pods will be orphaned (stuck with scheduling gate but no controller to remove it).

**Problem**: Gated pods become unschedulable permanently if controller is deleted before ungating them.

## Decision

Implement ordered deletion with explicit deprovisioning phase:

1. ClusterPodPlacementConfig receives deletion timestamp
2. Operator sets `Deprovisioning` status condition
3. Controller ungates all pods with `multiarch.openshift.io/scheduling-gate`
4. Only after all pods ungated, operator deletes operand deployments
5. Finalizer removed, ClusterPodPlacementConfig deleted

## Rationale

### Why This?
- **Prevents orphaned pods**: Ensures no pods left permanently gated
- **Safe cleanup**: Operands remain available until cleanup completes
- **Observable**: Deprovisioning condition visible to users
- **Idempotent**: Rerunning ungating is safe

### Why Not Alternatives?
- **Immediate deletion**: Leaves gated pods orphaned
- **Owner references**: Doesn't help with ungating (pods aren't owned by CPPC)
- **Background cleanup job**: Adds complexity, may fail to complete

## Consequences

### Positive
- ✅ No orphaned gated pods
- ✅ Deletion is safe and observable
- ✅ Can monitor deprovisioning progress via status

### Negative
- ❌ Deletion takes longer (waits for ungating)
- ❌ Adds complexity to operator controller

### Neutral
- ℹ️ Finalizer on ClusterPodPlacementConfig ensures deprovisioning runs

## Implementation

- **Finalizer**: controllers/operator/clusterpodplacementconfig_controller.go (adds finalizer on create)
- **Deprovisioning**: controllers/operator/deprovisioning.go
- **Ungating**: controllers/operator/clusterpodplacementconfig_controller.go:ungatePods()
- **Status**: Fully implemented

**Flow**:
```go
// Simplified logic
if cppc.DeletionTimestamp != nil {
    setCondition("Deprovisioning", "Ungating pods")

    // Ungate all pods
    podList := listGatedPods()
    for pod := range podList {
        removeSchedulingGate(pod)
    }

    // Only after ungating
    deleteOperands()
    removeFinalizer()
}
```

## Alternatives Considered

### Alternative 1: Immediate Operand Deletion
**Pros**: Fast cleanup
**Cons**: Orphans gated pods permanently
**Why rejected**: Unacceptable to leave workloads stuck

### Alternative 2: Background Cleanup Job
**Pros**: Deletion completes immediately, job handles cleanup
**Cons**: Job may fail, harder to observe, requires RBAC for job
**Why rejected**: Inline cleanup is simpler and more reliable

### Alternative 3: Owner References on Pods
**Pros**: Kubernetes handles cleanup automatically
**Cons**: Cannot add owner references to pods (we don't own them), doesn't help with ungating
**Why rejected**: Not applicable - we mutate pods but don't own them

## References

- [Deprovisioning implementation](../../controllers/operator/deprovisioning.go)
- [Kubernetes Finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
- [Core beliefs](../design-docs/core-beliefs.md) - Ordered deletion pattern

## Notes

This pattern was added after initial implementation when testing revealed orphaned pods during operator uninstall.
