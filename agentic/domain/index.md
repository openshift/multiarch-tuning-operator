# Domain Documentation

## Purpose

This section contains domain concepts, terminology, and workflows specific to the multiarch-tuning-operator.

## Contents

- [glossary.md](./glossary.md) - Canonical terminology definitions
- [concepts/](./concepts/) - Detailed concept documentation
  - [cluster-pod-placement-config.md](./concepts/cluster-pod-placement-config.md) - Singleton CR controlling operand
  - [scheduling-gate.md](./concepts/scheduling-gate.md) - Kubernetes mechanism to hold pods
  - [image-inspection.md](./concepts/image-inspection.md) - Determining supported architectures
  - [node-affinity.md](./concepts/node-affinity.md) - Kubernetes scheduling constraints
  - [pod-placement-operand.md](./concepts/pod-placement-operand.md) - Controllers and webhook
- [workflows/](./workflows/) - User and system workflows

## When to Add Here

Add a document here when:
- Defining a new domain concept or term
- Documenting a user or system workflow
- Explaining relationships between domain entities
- Clarifying business logic or domain rules

## Related Sections

- [Design Docs](../design-docs/) - Architectural design
- [Decisions](../decisions/) - ADRs referencing domain concepts
- [ARCHITECTURE.md](../../ARCHITECTURE.md) - System structure
