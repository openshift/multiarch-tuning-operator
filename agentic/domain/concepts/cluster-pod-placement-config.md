---
concept: ClusterPodPlacementConfig
type: CRD
related: [PodPlacementOperand, NamespaceSelector, LogVerbosity]
---

# ClusterPodPlacementConfig

## Definition

Singleton custom resource that controls the lifecycle and configuration of the pod placement operand, including namespace selection and log verbosity.

## Purpose

Provides cluster administrators a single configuration point to enable/disable architecture-aware pod scheduling and control which namespaces are affected.

## Location in Code

- **API Definition**: apis/multiarch/v1beta1/clusterpodplacementconfig_types.go
- **Conversion**: apis/multiarch/v1alpha1/clusterpodplacementconfig_conversion.go
- **Controller**: controllers/operator/clusterpodplacementconfig_controller.go
- **Webhook**: apis/multiarch/v1beta1/clusterpodplacementconfig_webhook.go
- **Tests**: apis/multiarch/v1beta1/clusterpodplacementconfig_webhook_test.go

## Lifecycle

```
1. Created by cluster admin (must be named "cluster")
2. Validated by validating webhook
3. Reconciled by operator controller
4. Operator deploys pod placement operands (controller, webhook)
5. Status conditions updated (Available, Progressing, Degraded)
6. On deletion: Deprovisioning condition set, pods ungated, operands removed
```

## Key Fields / Properties

### spec.namespaceSelector
**Type**: metav1.LabelSelector
**Purpose**: Determines which namespaces have pod placement enabled
**Example**:
```yaml
spec:
  namespaceSelector:
    matchExpressions:
      - key: multiarch.openshift.io/exclude-pod-placement
        operator: DoesNotExist
```

### spec.logVerbosity
**Type**: string (enum)
**Purpose**: Controls log verbosity for operands
**Values**: Normal, Debug, Trace, TraceAll
**Example**:
```yaml
spec:
  logVerbosity: Normal
```

## State Machine

```yaml
status.conditions:
  - Available: Operands deployed and ready
  - Progressing: Deployment in progress
  - Degraded: Operands unhealthy
  - Deprovisioning: Deletion in progress, ungating pods
  - PodPlacementControllerNotRolledOut: Controller deployment not ready
  - PodPlacementWebhookNotRolledOut: Webhook deployment not ready
  - MutatingWebhookConfigurationNotAvailable: Webhook config missing

transitions:
  - Created → Progressing: Operator begins deployment
  - Progressing → Available: All operands ready
  - Available → Degraded: Operand fails health check
  - Deleting → Deprovisioning: Finalizer triggers ungating
  - Deprovisioning → (deleted): All pods ungated, finalizer removed
```

## Common Patterns

### Minimal Configuration
```yaml
apiVersion: multiarch.openshift.io/v1beta1
kind: ClusterPodPlacementConfig
metadata:
  name: cluster
spec:
  logVerbosityLevel: Normal
  namespaceSelector:
    matchExpressions:
      - key: multiarch.openshift.io/exclude-pod-placement
        operator: DoesNotExist
```

**When to use**: Default setup for architecture-aware scheduling

## Related Concepts

- [PodPlacementOperand](./pod-placement-operand.md) - Deployed by this controller
- [NamespaceSelector](./namespace-selector.md) - Controls scope of pod placement

## Implementation Details

- **Logic**: controllers/operator/clusterpodplacementconfig_controller.go
- **Validation**: apis/multiarch/v1beta1/clusterpodplacementconfig_webhook.go (name must be "cluster")
- **Tests**: controllers/operator/clusterpodplacementconfig_controller_test.go

## References

- [ADR](../../decisions/adr-0002-singleton-config.md) - Why singleton design
- [OpenShift Enhancement Proposal](https://github.com/openshift/enhancements/blob/master/enhancements/multi-arch/multiarch-manager-operator.md)
