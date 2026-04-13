# multiarch-tuning-operator - Agent Navigation

> **Purpose**: Table of contents for AI agents. Points to deeper knowledge.
> **Do not expand this file**. Keep under 150 lines. Link to details instead.
>
> **New here?** Start with [README.md](./README.md) for project overview.

## What This Repository Does

Enhances operational experience within multi-architecture OpenShift clusters by providing architecture-aware scheduling of workloads through automatic nodeAffinity configuration based on container image architectures.

## Quick Navigation by Intent

**I need to understand the system**
→ [ARCHITECTURE.md](./ARCHITECTURE.md)
→ [Core beliefs](./agentic/design-docs/core-beliefs.md)
→ [Design docs index](./agentic/design-docs/index.md)
→ [Architecture decisions](./agentic/decisions/index.md)

**I'm implementing a feature**
1. INVESTIGATE: Read [ARCHITECTURE.md](./ARCHITECTURE.md), [design guide](./agentic/DESIGN.md), verify data structures
2. CREATE plan in [active plans](./agentic/exec-plans/active/index.md) using [template](./agentic/exec-plans/template.md)
3. READ [testing guide](./agentic/TESTING.md) and patterns
4. Implement with tests
5. Update plan to completed

**I'm reviewing security**
→ [Security model](./agentic/SECURITY.md)
→ [Core beliefs](./agentic/design-docs/core-beliefs.md)

**I need reliability context**
→ [Reliability guide](./agentic/RELIABILITY.md)
→ [Testing strategy](./agentic/TESTING.md)

**I'm fixing a bug**
→ [Component map](./ARCHITECTURE.md#components)
→ [Debugging](./agentic/DEVELOPMENT.md#debugging)
→ [Tests](./agentic/TESTING.md)

**I need to understand a concept**
→ [Domain documentation index](./agentic/domain/index.md)
→ [Glossary](./agentic/domain/glossary.md)
→ [Concepts](./agentic/domain/concepts/)

## Repository Structure

```
pkg/controllers/{operator,podplacement}  # Core controllers
pkg/image/                                # Image inspection
test/e2e/                                 # E2E tests
```

## Component Boundaries

```
┌────────────────────────────────┐
│  Operator Controller           │  Manages ClusterPodPlacementConfig CR
└────────────────────────────────┘
         ↓ deploys
┌────────────────────────────────┐
│  Pod Placement Webhook         │  Adds scheduling gates to pods
└────────────────────────────────┘
         ↓ gates pod
┌────────────────────────────────┐
│  Pod Placement Controller      │  Inspects images, sets nodeAffinity
└────────────────────────────────┘
         ↓ ungates pod
┌────────────────────────────────┐
│  Kubernetes Scheduler          │  Places pod on appropriate node
└────────────────────────────────┘
```

## Core Concepts (Domain Model)

| Concept | Definition | Docs |
|---------|-----------|------|
| ClusterPodPlacementConfig | Singleton CR controlling pod placement operand | [./agentic/domain/concepts/cluster-pod-placement-config.md](./agentic/domain/concepts/cluster-pod-placement-config.md) |
| Scheduling Gate | Kubernetes mechanism to hold pods before scheduling | [./agentic/domain/concepts/scheduling-gate.md](./agentic/domain/concepts/scheduling-gate.md) |
| Image Inspection | Determining supported architectures from container images | [./agentic/domain/concepts/image-inspection.md](./agentic/domain/concepts/image-inspection.md) |
| NodeAffinity | Kubernetes constraint for node selection | [./agentic/domain/concepts/node-affinity.md](./agentic/domain/concepts/node-affinity.md) |
| Pod Placement Operand | Controllers and webhook that perform scheduling | [./agentic/domain/concepts/pod-placement-operand.md](./agentic/domain/concepts/pod-placement-operand.md) |

## Key Invariants (ENFORCE THESE)

1. **ClusterPodPlacementConfig is Singleton**: Only resource named "cluster" allowed
   - Validated by: Validating webhook
   - Why: Single point of configuration for cluster-wide behavior

2. **System Namespaces Excluded**: openshift-*, kube-*, hypershift-* always excluded
   - Validated by: Webhook namespace selector
   - Why: Prevent interference with platform components

3. **All features require execution plans**: Must create plan in agentic/exec-plans/active/ before coding
   - Validated by: Code review
   - Why: Ensures design consideration and trackable decision history

## Critical Code Locations

| Purpose | File | Why Critical |
|---------|------|--------------|
| Pod reconciliation logic | controllers/podplacement/pod_reconciler.go | Core pod processing workflow |
| Image architecture detection | pkg/image/inspector.go | Determines supported architectures |
| Scheduling gate webhook | controllers/podplacement/scheduling_gate_mutating_webhook.go | Adds gates to pods |
| Operator reconciliation | controllers/operator/clusterpodplacementconfig_controller.go | Manages operand lifecycle |

## External Dependencies

- **controller-runtime**: Operator framework | **containers/image**: Image inspection | **OpenShift API**: CRDs

## Build & Test

```bash
# Build
make build

# Unit tests
make unit

# E2E tests (requires deployed operator)
KUBECONFIG=/path/to/kubeconfig NAMESPACE=openshift-multiarch-tuning-operator make e2e

# All checks (lint, vet, gosec, goimports, tests)
make test
```

## Documentation Structure

- [Design docs](./agentic/design-docs/index.md) - Architecture, components, patterns
- [Domain](./agentic/domain/index.md) - Concepts, glossary, workflows
- [Exec plans](./agentic/exec-plans/active/) - Active work tracking
- [Product specs](./agentic/product-specs/index.md) - Feature specifications
- [Decisions](./agentic/decisions/index.md) - Architecture Decision Records (ADRs)
- [References](./agentic/references/index.md) - External knowledge, primers
- [DESIGN.md](./agentic/DESIGN.md) - Design philosophy
- [DEVELOPMENT.md](./agentic/DEVELOPMENT.md) - Development setup
- [TESTING.md](./agentic/TESTING.md) - Test strategy
- [RELIABILITY.md](./agentic/RELIABILITY.md) - SLOs, observability
- [SECURITY.md](./agentic/SECURITY.md) - Security model
- [QUALITY_SCORE.md](./agentic/QUALITY_SCORE.md) - Documentation quality metrics

## When You're Stuck

1. Check [tech debt tracker](./agentic/exec-plans/tech-debt-tracker.md)
2. Check [quality score](./agentic/QUALITY_SCORE.md)
3. File a plan in [active plans](./agentic/exec-plans/active/)

## Last Updated

This file is validated by CI on every commit.
