# Architecture Overview

## System Context

**External Integrations:**

| System | Direction | Interface | File |
|--------|-----------|-----------|------|
| Kubernetes API | Bidirectional | controller-runtime client | pkg/controllers/*/reconcile.go |
| Container Registries | Outbound | containers/image library | pkg/image/inspector.go |
| OpenShift API Server | Inbound | CRD validation webhooks | apis/multiarch/v1beta1/*_webhook.go |

## Domain Architecture

### Package Layering (ENFORCED)

```
vendor/github.com/openshift/api
  ↓ (types only)
apis/multiarch/{v1alpha1,v1beta1}
  ↓
pkg/
  ├── controllers/        # Core reconciliation logic
  │   ├── operator/       # ClusterPodPlacementConfig controller
  │   └── podplacement/   # Pod controllers and webhook
  ├── image/              # Image inspection (uses containers/image)
  ├── informers/          # Singleton CPPC cache
  └── utils/              # Shared utilities
cmd/
  ├── main-binary/        # Single binary, multiple modes
  └── enoexec-daemon/     # eBPF monitoring daemon
```

### Dependency Rules (ENFORCED BY LINTER)

1. `pkg/controllers/podplacement` MUST NOT import `pkg/controllers/operator`
2. `pkg/image` MUST be self-contained (no controller imports)
3. Cross-component communication via CRs and informers only
4. All modes use shared packages but different leader election IDs

## Components

| Component | Entry Point | Critical Code | Purpose | Details |
|-----------|-------------|---------------|---------|---------|
| Operator | cmd/main-binary/main.go (--enable-operator) | controllers/operator/clusterpodplacementconfig_controller.go | Manages ClusterPodPlacementConfig CR lifecycle, deploys operands | [link](./agentic/design-docs/components/operator-controller.md) |
| Pod Controller | cmd/main-binary/main.go (--enable-ppc-controllers) | controllers/podplacement/pod_reconciler.go | Inspects images, sets nodeAffinity, removes gates | [link](./agentic/design-docs/components/pod-placement-controller.md) |
| Webhook | cmd/main-binary/main.go (--enable-ppc-webhook) | controllers/podplacement/scheduling_gate_mutating_webhook.go | Adds scheduling gates to new pods | [link](./agentic/design-docs/components/pod-placement-webhook.md) |
| ENoExec Daemon | cmd/enoexec-daemon/main.go | (eBPF-based) | Monitors exec format errors on nodes | [link](./agentic/design-docs/components/enoexec-daemon.md) |

## Data Flow

```
User creates Pod
  ↓
Webhook adds schedulingGates (controllers/podplacement/scheduling_gate_mutating_webhook.go)
  ↓
Pod queued (status.phase=Pending)
  ↓
PodReconciler watches gated pods (controllers/podplacement/pod_reconciler.go)
  ↓
Inspect images for architectures (pkg/image/inspector.go)
  ↓
Set nodeAffinity (controllers/podplacement/pod_model.go)
  ↓
Remove schedulingGates
  ↓
Kubernetes Scheduler places pod
```

## Critical Code Locations

| Function | File | Why Critical |
|----------|------|--------------|
| Pod reconciliation | controllers/podplacement/pod_reconciler.go | Main pod processing loop |
| Image inspection | pkg/image/inspector.go | Architecture detection |
| NodeAffinity computation | controllers/podplacement/pod_model.go | Scheduling constraint logic |
| Operand deployment | controllers/operator/clusterpodplacementconfig_controller.go | Manages operand lifecycle |

See [complete package map](./agentic/generated/package-map.md) for details.

## Execution Modes

The operator binary (`main-binary`) runs in mutually exclusive modes controlled by flags:

| Flag | Mode | Leader Election ID | Purpose |
|------|------|-------------------|---------|
| `--enable-operator` | Operator | `clusterpodplacementconfig-operator-lock` | Manage CPPC CR |
| `--enable-ppc-controllers` | Pod Placement Controllers | `pod-placement-controller-lock` | Reconcile pods |
| `--enable-ppc-webhook` | Pod Placement Webhook | `pod-placement-webhook-lock` | Mutate pods |
| `--enable-enoexec-event-controllers` | ENoExecEvent Controllers | `enoexec-event-controller-lock` | Handle exec errors |

See [Binary Modes](./agentic/design-docs/binary-modes.md) for detailed explanation.

## Related Documentation

- [Design docs](./agentic/design-docs/)
- [Domain concepts](./agentic/domain/)
- [ADRs](./agentic/decisions/)
