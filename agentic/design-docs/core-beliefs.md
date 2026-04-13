# Core Beliefs - multiarch-tuning-operator

## Operating Principles

### 1. Fail-Safe by Default
Errors in pod placement should not prevent workloads from scheduling. If image inspection fails, pods proceed without architecture constraints.

**Implications**:
- Max retries mechanism prevents infinite loops
- Errors logged but don't block scheduling
- Metrics track failure rates for monitoring

**Example**: If image inspection fails after max retries, pod is ungated without nodeAffinity modification (controllers/podplacement/pod_reconciler.go)

### 2. Minimize Time-to-Schedule
Pods should spend minimal time waiting for architecture determination. Optimize for fast image inspection and parallel processing.

**Implications**:
- High concurrency (NumCPU * 4) for pod reconciliation
- Caching of image inspection results
- Efficient field selectors to watch only Pending pods

**Example**: PodReconciler uses MaxConcurrentReconciles = NumCPU * 4 because image inspection is I/O bound

### 3. Platform Components Are Untouchable
System namespaces (openshift-*, kube-*, hypershift-*) must never be processed by pod placement operands.

**Implications**:
- Hardcoded namespace exclusions cannot be overridden
- Webhook and controller skip system namespaces
- Prevents interference with platform stability

**Example**: Namespace selector always excludes openshift-*, kube-*, hypershift-* (controllers/podplacement/scheduling_gate_mutating_webhook.go)

### 4. Configuration is Singleton
Only one ClusterPodPlacementConfig resource is allowed, named "cluster".

**Implications**:
- Validating webhook rejects other names
- Simplified configuration model
- Single source of truth for cluster-wide behavior

**Example**: Webhook validation enforces name == "cluster" (apis/multiarch/v1beta1/clusterpodplacementconfig_webhook.go)

## Non-Negotiable Constraints

### Security
- ✅ Pull secrets must be handled securely (never logged)
- ✅ RBAC limits controller permissions to necessary resources
- ✅ Webhook certificates managed via cert-manager
- ❌ NEVER allow arbitrary container execution for image inspection

### Reliability
- ✅ Controllers must use leader election
- ✅ Metrics must track all operations for observability
- ✅ Degraded operands must report via status conditions

### Correctness
- ✅ NodeAffinity must accurately reflect supported architectures
- ✅ Scheduling gates must be removed only after successful processing
- ✅ API conversions must be lossless (v1alpha1 ↔ v1beta1)

## Patterns We Use

### Verify Before Implementing Pattern
**What**: Always verify actual data structures, file paths, and output formats before making assumptions

**When to use**: Before writing any code that processes or generates data from the system

**How to verify**:
1. Check reference documentation (e.g., API specs, CRD definitions)
2. Use grep to search for actual usage patterns in codebase
3. Look at similar implementations (e.g., existing controllers)
4. Test assumptions with actual resources

**Example in this repo**: Before modifying pod specs, verify pod structure via apis/multiarch tests

**Why important**: Prevents implementing based on incorrect assumptions about Kubernetes API structure

See: [Verify pattern details](./patterns/verify-before-implementing.md)

### Ordered Deletion Pattern
**What**: When deprovisioning, ungating pods must happen before removing webhook

**When to use**: ClusterPodPlacementConfig deletion

**Example in this repo**: Operator sets Deprovisioning condition, waits for pods to be ungated, then removes operands (controllers/operator/deprovisioning.go)

See: [Ordered deletion ADR](../decisions/adr-0001-ordered-deletion.md)

### Image Inspection Caching Pattern
**What**: Cache image manifest inspection results to reduce registry API calls

**When to use**: Processing multiple pods with same images

**Example in this repo**: pkg/image/inspector.go maintains manifest cache

## Deprecated Patterns

### ❌ Synchronous Webhook Mutation
**Don't**: Perform image inspection in mutating webhook
**Do**: Use scheduling gates + async controller
**Why**: Image inspection requires external API calls (slow), webhooks must respond quickly

### ❌ Global Pull Secret in ConfigMap
**Don't**: Store pull secret reference in ConfigMap
**Do**: Sync from openshift-config/pull-secret to operand namespace
**Why**: Security - limit exposure of pull secret

## When to Break These Rules

1. Document in [agentic/decisions/](../decisions/)
2. Get consensus from team/maintainers
3. Add to tech debt tracker if temporary
