# Design Philosophy - multiarch-tuning-operator

## Overview

The multiarch-tuning-operator embodies a fail-safe, minimally-invasive approach to architecture-aware pod scheduling in Kubernetes, prioritizing workload availability over enforcement strictness.

## Design Principles

### 1. Fail-Safe Over Fail-Secure
When in doubt, let pods schedule without constraints rather than blocking them.

**Why**: Availability is paramount. Better to schedule a pod on wrong architecture (detectable) than block a workload indefinitely (silent failure).

**Example**: If image inspection fails after max retries, pod is ungated without nodeAffinity modification. Monitoring alerts on high failure rates.

### 2. Asynchronous is Better Than Synchronous
Decouple critical path operations from pod creation to avoid blocking user workflows.

**Why**: Pod creation must be fast and reliable. External dependencies (registry APIs) introduce latency and failure modes.

**Example**: Webhook adds scheduling gate (<100ms), controller performs image inspection async (can take seconds with retries).

### 3. Kubernetes-Native Where Possible
Prefer standard Kubernetes mechanisms over custom implementations.

**Why**: Reduces complexity, improves compatibility, leverages upstream testing and documentation.

**Example**: Use scheduling gates (KEP-3521) instead of custom state CRDs for pod holding.

### 4. Explicit Over Implicit
Make behavior observable and configurable rather than magical.

**Why**: Operators need to understand and control what the system does.

**Example**: ClusterPodPlacementConfig explicitly declares namespace selector; metrics track every operation.

## Architecture Decisions

Key architectural decisions that shape this codebase:

1. **Scheduling Gates for Async Processing**: Decouple pod mutation from webhook timeout
   - See: [ADR-0001](./decisions/adr-0001-scheduling-gates-for-async-pod-modification.md)

2. **Singleton Configuration**: One ClusterPodPlacementConfig named "cluster"
   - See: [ADR-0002](./decisions/adr-0002-singleton-clusterpodplacementconfig.md)

3. **Ordered Deletion**: Ungate pods before removing operands
   - See: [ADR-0003](./decisions/adr-0003-ordered-deletion-during-deprovisioning.md)

## Design Patterns

### High Concurrency for I/O-Bound Operations
**What**: PodReconciler uses NumCPU * 4 concurrent reconciliations
**When to use**: Operations bottlenecked on network I/O (image inspection)
**Example**: controllers/podplacement/pod_reconciler.go

### In-Memory Caching
**What**: Cache image manifest results to reduce registry API calls
**When to use**: Repeated access to same data with acceptable staleness
**Example**: pkg/image/inspector.go (manifest cache)

### Metrics-First Observability
**What**: Emit Prometheus metrics for all significant operations
**When to use**: Any operation that can fail or has variable latency
**Example**: controllers/podplacement/metrics/metrics.go

## Anti-Patterns to Avoid

### ❌ Synchronous External API Calls in Webhooks
**Don't**: Call container registries from mutating webhook
**Do**: Use scheduling gates + async controller
**Why**: Webhook timeouts cause pod creation failures

### ❌ Implicit Configuration
**Don't**: Hardcode behavior without configuration option
**Do**: Make behavior configurable via ClusterPodPlacementConfig
**Why**: Different clusters have different requirements

### ❌ Orphaned Resources
**Don't**: Delete controllers without cleaning up state they manage
**Do**: Implement ordered deletion with finalizers
**Why**: Leaves workloads in broken state

## Trade-offs

### Latency vs. Correctness
**What we chose**: Accept scheduling latency for correct architecture placement
**What we gave up**: Immediate pod scheduling
**Why**: Exec format errors are harder to debug than delayed scheduling

### Flexibility vs. Simplicity
**What we chose**: Singleton configuration with namespace selector
**What we gave up**: Per-namespace configuration flexibility
**Why**: Simplicity and predictability more valuable than flexibility

### Automation vs. Control
**What we chose**: Automatic pod mutation with opt-out via namespace labels
**What we gave up**: Explicit per-pod opt-in
**Why**: Better UX - works automatically for most users, exceptions use labels

## Related Documentation

- [Core Beliefs](./design-docs/core-beliefs.md) - Detailed patterns and principles
- [Architecture](../ARCHITECTURE.md) - System structure
- [ADRs](./decisions/) - Individual design decisions
