---
concept: SchedulingGate
type: Kubernetes Feature
related: [PodSchedulingReadiness, KEP-3521, PodPlacement]
---

# Scheduling Gate

## Definition

Kubernetes v1.27+ feature that prevents the scheduler from considering a pod for scheduling until all gates are removed, enabling asynchronous pod modification before scheduling.

## Purpose

Allows the pod placement controller to inspect container images and modify pod nodeAffinity without racing against the scheduler.

## Location in Code

- **Webhook adds gate**: controllers/podplacement/scheduling_gate_mutating_webhook.go
- **Controller removes gate**: controllers/podplacement/pod_reconciler.go
- **Gate constant**: pkg/common/constants.go (SchedulingGateName = "multiarch.openshift.io/scheduling-gate")
- **Tests**: controllers/podplacement/pod_reconciler_test.go

## Lifecycle

```
1. Pod created by user/controller
2. Webhook adds schedulingGates[].name = "multiarch.openshift.io/scheduling-gate"
3. Pod enters Pending phase but scheduler ignores it
4. PodReconciler watches pods with scheduling gate
5. Image inspection completes
6. NodeAffinity set on pod spec
7. Scheduling gate removed from pod
8. Scheduler places pod on appropriate node
```

## Key Fields / Properties

### pod.spec.schedulingGates
**Type**: []PodSchedulingGate
**Purpose**: List of gates blocking scheduling
**Example**:
```yaml
spec:
  schedulingGates:
    - name: "multiarch.openshift.io/scheduling-gate"
```

## Common Patterns

### Adding Gate in Webhook
```go
gate := corev1.PodSchedulingGate{
    Name: common.SchedulingGateName,
}
pod.Spec.SchedulingGates = append(pod.Spec.SchedulingGates, gate)
```

**When to use**: When pod enters cluster and needs architecture determination

### Removing Gate in Controller
```go
gates := []corev1.PodSchedulingGate{}
for _, gate := range pod.Spec.SchedulingGates {
    if gate.Name != common.SchedulingGateName {
        gates = append(gates, gate)
    }
}
pod.Spec.SchedulingGates = gates
```

**When to use**: After successfully setting nodeAffinity or on max retries

## Related Concepts

- [ImageInspection](./image-inspection.md) - Performed while pod is gated
- [NodeAffinity](./node-affinity.md) - Set before removing gate
- [PodPlacement](./pod-placement-operand.md) - Uses gates for async processing

## Implementation Details

- **Gate addition**: controllers/podplacement/scheduling_gate_mutating_webhook.go:138
- **Gate removal**: controllers/podplacement/pod_model.go:removeSchedulingGate()
- **Validation**: Kubernetes API server enforces scheduling gate semantics

## References

- [KEP-3521: Pod Scheduling Readiness](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/3521-pod-scheduling-readiness)
- [Kubernetes Documentation](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-scheduling-readiness/)
