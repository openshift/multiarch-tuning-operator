---
id: ADR-0001
title: Use Scheduling Gates for Async Pod Modification
date: 2024-01-15
status: accepted
deciders: [openshift-multiarch-team]
supersedes: null
superseded-by: null
---

# Use Scheduling Gates for Async Pod Modification

## Status

Accepted (implemented)

## Context

We need to modify pod nodeAffinity based on container image architectures before the pod is scheduled. Image inspection requires external API calls to container registries, which can take several seconds.

Traditional mutating webhooks must respond synchronously (within ~10s timeout), but we cannot perform reliable image inspection within that timeframe due to:
- Registry network latency
- Rate limiting on registry APIs
- Potential authentication failures requiring retry
- Multiple images per pod requiring sequential inspection

## Decision

Use Kubernetes Scheduling Gates (KEP-3521, GA in v1.27) to hold pods in Pending state while an asynchronous controller inspects images and modifies pod specs.

**Flow**:
1. Mutating webhook adds `multiarch.openshift.io/scheduling-gate` to new pods (fast, <100ms)
2. Pod enters Pending phase but scheduler ignores gated pods
3. PodReconciler watches gated pods, performs image inspection (async, can take seconds)
4. Controller sets nodeAffinity and removes scheduling gate
5. Scheduler places pod on appropriate node

## Rationale

### Why This?
- **Decouples mutation from webhook timeout**: Image inspection happens async in controller with retries
- **Kubernetes-native**: Uses standard scheduling gate feature, no custom state management
- **Reliable**: Controller can retry failed inspections without blocking pod creation
- **Observable**: Metrics track time-to-ungate and inspection failures

### Why Not Alternatives?
- **Synchronous webhook mutation**: Cannot perform async operations (registry API calls) within webhook timeout
- **Custom resource for state tracking**: Adds complexity, requires garbage collection, not standard Kubernetes pattern
- **Manual annotation by users**: Poor user experience, error-prone

## Consequences

### Positive
- ✅ Reliable image inspection with retries and proper error handling
- ✅ No webhook timeouts blocking pod creation
- ✅ Clear separation of concerns (webhook gates, controller processes)
- ✅ Metrics visibility into processing time

### Negative
- ❌ Requires Kubernetes v1.27+ (not available on older clusters)
- ❌ Adds latency to pod scheduling (pods wait for image inspection)
- ❌ Potential for gated pods to be orphaned if controller crashes

### Neutral
- ℹ️ Need finalizer on ClusterPodPlacementConfig to ungate pods before operand deletion

## Implementation

- **Webhook**: controllers/podplacement/scheduling_gate_mutating_webhook.go
- **Controller**: controllers/podplacement/pod_reconciler.go
- **Gate removal**: controllers/podplacement/pod_model.go:removeSchedulingGate()
- **Status**: Fully implemented and deployed

## Alternatives Considered

### Alternative 1: Synchronous Webhook with Fast Timeout
**Pros**: Simpler architecture, no controller needed
**Cons**: Cannot perform reliable image inspection within webhook timeout
**Why rejected**: Image inspection is inherently async (network I/O), webhook timeouts would cause pod creation failures

### Alternative 2: Custom "PodPlacementRequest" CRD
**Pros**: Full control over state machine
**Cons**: Adds complexity, non-standard pattern, requires garbage collection
**Why rejected**: Kubernetes scheduling gates provide same functionality with native support

### Alternative 3: Require Manual Pod Annotation
**Pros**: No automatic processing needed
**Cons**: Poor UX, requires users to determine image architectures manually
**Why rejected**: Goal is automatic, transparent architecture-aware scheduling

## References

- [KEP-3521: Pod Scheduling Readiness](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/3521-pod-scheduling-readiness)
- [Concept doc](../domain/concepts/scheduling-gate.md)
- [OpenShift Enhancement Proposal](https://github.com/openshift/enhancements/blob/master/enhancements/multi-arch/multiarch-manager-operator.md)

## Notes

Originally considered for Kubernetes v1.26 (beta), but delayed adoption until v1.27 (GA) to ensure stability.
