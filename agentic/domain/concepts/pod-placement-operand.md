---
concept: PodPlacementOperand
type: Component
related: [ClusterPodPlacementConfig, PodReconciler, SchedulingGateWebhook]
---

# Pod Placement Operand

## Definition

Set of Kubernetes controllers and webhook deployed by the operator to perform architecture-aware pod scheduling via image inspection and nodeAffinity configuration.

## Purpose

Automatically configures pods with appropriate nodeAffinity constraints based on container image architectures, preventing exec format errors in multi-architecture clusters.

## Location in Code

- **Deployment manifests**: controllers/operator/manifests/
- **Operator deployment logic**: controllers/operator/clusterpodplacementconfig_controller.go
- **Pod reconciler**: controllers/podplacement/pod_reconciler.go
- **Webhook**: controllers/podplacement/scheduling_gate_mutating_webhook.go
- **Tests**: controllers/operator/clusterpodplacementconfig_controller_test.go

## Lifecycle

```
1. ClusterPodPlacementConfig created
2. Operator controller reconciles
3. Operator deploys:
   a. Pod placement controller deployment
   b. Pod placement webhook deployment
   c. MutatingWebhookConfiguration
   d. ServiceMonitor for metrics
   e. RBAC (ServiceAccount, Role, RoleBinding)
4. Controllers become ready
5. Operator updates CPPC status to Available
6. On CPPC deletion:
   a. Deprovisioning condition set
   b. Pods ungated
   c. Operand deployments deleted
   d. CPPC finalizer removed
```

## Components

### Pod Placement Controller
**Deployment**: pod-placement-controller
**Binary**: main-binary --enable-ppc-controllers
**Purpose**: Reconciles gated pods, inspects images, sets nodeAffinity
**Namespace**: openshift-multiarch-tuning-operator

### Pod Placement Webhook
**Deployment**: pod-placement-webhook
**Binary**: main-binary --enable-ppc-webhook
**Purpose**: Adds scheduling gates to new pods
**Namespace**: openshift-multiarch-tuning-operator

### MutatingWebhookConfiguration
**Name**: pod-placement-scheduling-gate
**Purpose**: Routes pod creation to webhook
**Scope**: Cluster-wide (except excluded namespaces)

## Key Fields / Properties

### CPPC Status Conditions
**Type**: metav1.Condition
**Purpose**: Report operand health
**Conditions**:
- PodPlacementControllerNotRolledOut: Controller deployment not ready
- PodPlacementWebhookNotRolledOut: Webhook deployment not ready
- MutatingWebhookConfigurationNotAvailable: Webhook config missing

## Common Patterns

### Health Monitoring
```go
// Check if operand deployments are ready
if deployment.Status.AvailableReplicas < *deployment.Spec.Replicas {
    setCondition(PodPlacementControllerNotRolledOut, "Not all replicas available")
}
```

**When to use**: Operator reconciliation loop

### Ordered Deletion
```go
// Before removing operands, ungate all pods
if cppc.DeletionTimestamp != nil {
    setCondition(Deprovisioning, "Ungating pods before deletion")
    ungatePods()
    // Only after ungating completes, remove operands
}
```

**When to use**: CPPC deletion to prevent orphaned gated pods

## Related Concepts

- [ClusterPodPlacementConfig](./cluster-pod-placement-config.md) - Controls operand lifecycle
- [SchedulingGate](./scheduling-gate.md) - Mechanism used by operand
- [ImageInspection](./image-inspection.md) - Core operand functionality

## Implementation Details

- **Deployment**: controllers/operator/clusterpodplacementconfig_controller.go:deployPodPlacementOperand()
- **Health checks**: controllers/operator/clusterpodplacementconfig_controller.go:updateStatus()
- **Metrics**: controllers/podplacement/metrics/metrics.go

## Metrics

### Controller Metrics
- `mto_ppo_ctrl_processed_pods_total`: Total pods processed
- `mto_ppo_ctrl_time_to_process_gated_pod_seconds`: Processing time
- `mto_ppo_ctrl_failed_image_inspection_total`: Inspection failures

### Webhook Metrics
- `mto_ppo_wh_pods_processed_total`: Total pods seen by webhook
- `mto_ppo_wh_pods_gated_total`: Total pods gated
- `mto_ppo_wh_response_time_seconds`: Webhook latency

### Shared Metrics
- `mto_ppo_pods_gated`: Current count of gated pods (gauge)

## References

- [ADR](../../decisions/adr-0001-ordered-deletion.md) - Ordered deletion pattern
- [Metrics Guide](../../../docs/metrics.md) - Complete metrics documentation
