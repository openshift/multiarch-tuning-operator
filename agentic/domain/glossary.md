# Glossary - multiarch-tuning-operator

> **Purpose**: Canonical definitions for all domain concepts.
> **Format**: Alphabetical order. Link to detailed docs.

## C

### ClusterPodPlacementConfig

**Definition**: Singleton custom resource (name "cluster") that controls pod placement operand lifecycle and configuration.

**Type**: CRD

**Related**: PodPlacementOperand, NamespaceSelector

**Details**: [./concepts/cluster-pod-placement-config.md](./concepts/cluster-pod-placement-config.md)

## E

### ENoExecEvent

**Definition**: Custom resource created when eBPF daemon detects exec format errors on nodes, indicating architecture mismatch.

**Type**: CRD

**Related**: MultiarchDaemon

**Details**: [./concepts/enoexec-event.md](./concepts/enoexec-event.md)

## I

### Image Inspection

**Definition**: Process of retrieving container image manifests from registries to determine supported CPU architectures.

**Type**: Concept

**Related**: ImageManifest, PullSecret

**Details**: [./concepts/image-inspection.md](./concepts/image-inspection.md)

## M

### Multi-Architecture Cluster

**Definition**: OpenShift cluster with compute nodes of different CPU architectures (e.g., amd64 and arm64).

**Type**: Concept

**Related**: NodeArchitecture

**Details**: [./concepts/multi-architecture-cluster.md](./concepts/multi-architecture-cluster.md)

## N

### Namespace Selector

**Definition**: Label selector in ClusterPodPlacementConfig that determines which namespaces have pod placement enabled.

**Type**: Concept

**Related**: ClusterPodPlacementConfig

**Details**: [./concepts/namespace-selector.md](./concepts/namespace-selector.md)

### NodeAffinity

**Definition**: Kubernetes scheduling constraint that limits which nodes a pod can be scheduled on based on node labels.

**Type**: Kubernetes Concept

**Related**: SchedulingGate, PodPlacement

**Details**: [./concepts/node-affinity.md](./concepts/node-affinity.md)

## P

### Pod Placement Operand

**Definition**: Set of controllers and webhook deployed by operator to perform architecture-aware pod scheduling.

**Type**: Component

**Related**: PodReconciler, SchedulingGateWebhook

**Details**: [./concepts/pod-placement-operand.md](./concepts/pod-placement-operand.md)

## S

### Scheduling Gate

**Definition**: Kubernetes v1.27+ feature that prevents pod scheduling until gate is removed, enabling async pod modification.

**Type**: Kubernetes Feature

**Related**: PodSchedulingReadiness, KEP-3521

**Details**: [./concepts/scheduling-gate.md](./concepts/scheduling-gate.md)

---

## See Also

- [Domain concepts](./concepts/) - Detailed explanations
- [Workflows](./workflows/) - How concepts interact
- [ARCHITECTURE.md](../../ARCHITECTURE.md) - System structure
