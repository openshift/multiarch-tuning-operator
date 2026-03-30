---
id: ADR-0002
title: Singleton ClusterPodPlacementConfig Resource
date: 2024-01-15
status: accepted
deciders: [openshift-multiarch-team]
supersedes: null
superseded-by: null
---

# Singleton ClusterPodPlacementConfig Resource

## Status

Accepted (implemented)

## Context

We need a custom resource to configure pod placement behavior cluster-wide. Design choices include:
- Allow multiple ClusterPodPlacementConfig resources with different selectors
- Allow single resource with name flexibility
- Enforce singleton with fixed name "cluster"

Multi-resource approach would enable different configurations per namespace group, but adds complexity in determining precedence and conflict resolution.

## Decision

Enforce singleton ClusterPodPlacementConfig with mandatory name "cluster". Only one instance allowed cluster-wide, validated by admission webhook.

## Rationale

### Why This?
- **Simple mental model**: One configuration for entire cluster, easy to understand and debug
- **No precedence conflicts**: Cannot have overlapping namespace selectors with different configs
- **Consistent with OpenShift patterns**: Other cluster-scoped singletons use this pattern (e.g., cluster operator configs)
- **Single source of truth**: All pod placement behavior controlled from one place

### Why Not Alternatives?
- **Multiple resources**: Requires complex precedence rules, conflict detection, and merging logic
- **Free naming**: No benefit to allowing arbitrary names for singleton resource

## Consequences

### Positive
- ✅ Simple, predictable configuration model
- ✅ No conflict resolution needed
- ✅ Easy to locate configuration (always named "cluster")
- ✅ Matches OpenShift conventions

### Negative
- ❌ Cannot have different configurations for different namespace groups
- ❌ Less flexible than multi-resource approach

### Neutral
- ℹ️ Namespace selector provides sufficient flexibility for most use cases

## Implementation

- **Validation**: apis/multiarch/v1beta1/clusterpodplacementconfig_webhook.go (validates name == "cluster")
- **Controller**: controllers/operator/clusterpodplacementconfig_controller.go (watches singleton)
- **Status**: Fully implemented, webhook rejects non-"cluster" names

## Alternatives Considered

### Alternative 1: Multiple ClusterPodPlacementConfig Resources
**Pros**: More flexible, can have different configs per namespace group
**Cons**: Requires precedence rules, conflict detection, complex to debug
**Why rejected**: Complexity not justified by use cases; namespace selector sufficient

### Alternative 2: ConfigMap Instead of CRD
**Pros**: No CRD installation needed
**Cons**: No schema validation, no status reporting, not declarative
**Why rejected**: CRD provides better UX with validation and status conditions

### Alternative 3: Free Naming for Singleton
**Pros**: Users can choose meaningful names
**Cons**: No benefit for singleton, adds validation complexity
**Why rejected**: Fixed name "cluster" is conventional for cluster-scoped config

## References

- [ClusterPodPlacementConfig CRD](../../apis/multiarch/v1beta1/clusterpodplacementconfig_types.go)
- [Concept doc](../domain/concepts/cluster-pod-placement-config.md)
- [Validation webhook](../../apis/multiarch/v1beta1/clusterpodplacementconfig_webhook.go)

## Notes

Early design iterations considered multi-resource approach, but testing revealed complexity in precedence rules outweighed flexibility benefits.
