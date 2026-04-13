# Security - multiarch-tuning-operator

## Security Model

### Trust Boundaries

```
[External]              [API Gateway]        [Internal Services]       [Data Store]
Container Registries → OpenShift API → Operator/Controllers → Kubernetes API
^untrusted              ^auth required   ^trusted namespace     ^RBAC enforced
```

**Key Boundaries**:
1. **External registries**: Untrusted, require authentication
2. **OpenShift API**: Authenticated via ServiceAccount tokens
3. **Operator namespace**: Trusted, isolated from user workloads
4. **User pods**: Untrusted, mutated by webhook

### Threat Model

**Assets**:
1. **Pull secrets** - Container registry credentials
   - Protection: Stored in Kubernetes Secrets, not logged, synced with RBAC restrictions
2. **Cluster configuration** - ClusterPodPlacementConfig
   - Protection: RBAC limits modification to cluster-admin
3. **Pod specs** - User workload definitions
   - Protection: Webhook mutates but preserves user-defined constraints

**Threats**:

1. **Pull Secret Exposure**
   - **Attack Vector**: Logs, metrics, status fields could expose credentials
   - **Impact**: Unauthorized registry access
   - **Mitigation**: Never log pull secret contents, sanitize all outputs
   - **Risk Level**: High

2. **Privilege Escalation via Pod Mutation**
   - **Attack Vector**: Malicious pod could exploit webhook to gain unintended access
   - **Impact**: Schedule pods on unauthorized nodes
   - **Mitigation**: Webhook only adds nodeAffinity (restrictive), never removes user constraints
   - **Risk Level**: Low (additive mutation only)

3. **Denial of Service via Image Inspection**
   - **Attack Vector**: Attacker creates many pods with slow-to-inspect images
   - **Impact**: Controller overwhelmed, pod scheduling delayed
   - **Mitigation**: Max retries limit, timeout controls, metrics for detection
   - **Risk Level**: Medium

4. **Registry Credential Theft**
   - **Attack Vector**: Compromised controller pod could exfiltrate pull secrets
   - **Impact**: Registry credentials stolen
   - **Mitigation**: Minimal RBAC, network policies, audit logging
   - **Risk Level**: Medium

**Threat Modeling Framework**: STRIDE (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege)

## Authentication & Authorization

### Authentication
**Mechanism**: Kubernetes ServiceAccount tokens (projected volumes)
**Implementation**: controllers/operator/clusterpodplacementconfig_controller.go
**Token Lifetime**: Default Kubernetes token rotation (1 hour)

**ServiceAccounts**:
- `multiarch-tuning-operator` - Operator controller
- `pod-placement-controller` - Pod reconciler
- `pod-placement-webhook` - Mutating webhook

### Authorization
**Model**: RBAC (Role-Based Access Control)
**Implementation**: config/rbac/

**Permissions**:

| ServiceAccount | Resource | Verbs | Scope | Why |
|----------------|----------|-------|-------|-----|
| operator | ClusterPodPlacementConfig | * | Cluster | Manage CPPC lifecycle |
| operator | Deployments | create,update,delete | Namespace | Deploy operands |
| operator | ServiceAccounts, Roles, RoleBindings | create,update,delete | Namespace | Setup RBAC for operands |
| pod-placement-controller | Pods | get,list,watch,update | Cluster | Reconcile pods |
| pod-placement-controller | Secrets | get,list,watch | Namespace | Access pull secrets |
| pod-placement-webhook | Pods | mutate (via webhook) | Cluster | Add scheduling gates |

**Principle of Least Privilege**: Each component has only permissions required for its function.

## Data Protection

### Data Classification
- **Public**: Metrics (no sensitive data in labels/values)
- **Internal**: Pod specs (namespace, names, images)
- **Confidential**: Pull secrets (registry credentials)
- **Restricted**: N/A

### Encryption
**At Rest**: Kubernetes Secrets encryption (configured at cluster level)
**In Transit**: TLS for all API communication (Kubernetes enforced)
**Key Management**: Kubernetes handles Secret encryption keys

### Secrets Management
**Storage**: Kubernetes Secrets
- `pull-secret` - Synced from openshift-config/pull-secret
- `webhook-cert` - TLS certificate for webhook

**Rotation**:
- Pull secret: Managed by cluster admin
- Webhook cert: Managed by cert-manager (auto-rotation)

**Access Control**: RBAC limits Secret access to specific ServiceAccounts

## Input Validation

### User Input (ClusterPodPlacementConfig)

**Validated Fields**:
- `metadata.name`: Must equal "cluster" (webhook validates)
- `spec.namespaceSelector`: Valid LabelSelector (Kubernetes validates)
- `spec.logVerbosity`: Must be one of [Normal, Debug, Trace, TraceAll] (webhook validates)

**Validation Implementation**: apis/multiarch/v1beta1/clusterpodplacementconfig_webhook.go

### API Input (Pod Mutation)

**Webhook Validation**:
- Rejects pods in system namespaces (openshift-*, kube-*, hypershift-*)
- Validates pod has containers (not empty)
- Ensures scheduling gate name is correct constant

**Implementation**: controllers/podplacement/scheduling_gate_mutating_webhook.go

### External Input (Image Manifests)

**Registry Responses**:
- JSON schema validation via containers/image library
- Reject malformed manifests
- Timeout on slow responses (context deadline)

**Implementation**: pkg/image/inspector.go

## Secure Coding Practices

### Mandatory Checks
- [x] Input validation on all external inputs (CPPC, pod specs, image manifests)
- [x] Never log sensitive data (pull secrets)
- [x] Use parameterized Kubernetes API calls (no string interpolation)
- [x] Timeout all external network calls (registry API)
- [x] RBAC follows least privilege

### Code Review Focus
- Pull secret handling (ensure never logged or exposed)
- Pod mutation logic (ensure additive, not removing user constraints)
- Error messages (ensure no sensitive data)
- Network calls (ensure timeout and retry limits)

### Static Analysis
**Tool**: gosec (via `make gosec`)
**Frequency**: On every PR (CI enforced)
**Response SLA**: Critical findings block merge

## Vulnerability Management

### Dependency Scanning
**Tool**: Dependabot (GitHub)
**Frequency**: Daily
**Response SLA**:
- Critical: 7 days
- High: 30 days
- Medium: 90 days
- Low: Best effort

### Security Testing
**SAST**: gosec (static analysis)
**DAST**: Not applicable (no web UI)
**Penetration Testing**: Not regularly performed

### Incident Response

**Security Incidents**:
1. **Detection**: CVE notifications, security scanner alerts, user reports
2. **Containment**: Patch vulnerable dependencies, update operator image
3. **Investigation**: Review logs, identify affected clusters
4. **Remediation**: Release patched version, notify users
5. **Reporting**: Security advisory via GitHub, CVE if applicable

**Contact**: OpenShift security team via standard channels

## Compliance

**Standards**: Follows OpenShift security requirements
**Audit Logs**: Kubernetes audit logs capture all API operations
**Compliance Checks**: OpenShift compliance operator scans

## Security Contacts

**Security Team**: OpenShift security team
**Vulnerability Reports**: Via GitHub Security Advisories or Red Hat security
**Security Mailing List**: N/A (use GitHub issues for public reports)

## Known Security Considerations

### Pull Secret Handling
**Risk**: Pull secrets contain registry credentials
**Mitigation**:
- Never logged (verified by code review)
- Access limited to specific ServiceAccounts via RBAC
- Stored in Kubernetes Secrets with encryption at rest
- Only used for image inspection, never exposed in pod specs or status

**Code locations to audit**:
- pkg/image/inspector.go - Uses pull secrets
- controllers/podplacement/global_pull_secret_syncer.go - Syncs secrets

### Webhook Certificate Management
**Risk**: Expired or compromised webhook certificates break pod creation
**Mitigation**:
- Cert-manager auto-rotates certificates
- Webhook failure mode: Fail-open (pods created without gate if webhook unavailable)
- Monitoring via MutatingWebhookConfigurationNotAvailable condition

### Minimal Runtime Container Image
**Risk**: Container images with shells and utilities increase attack surface, especially for privileged containers
**Mitigation** (since ADR-0004):
- Runtime image based on `scratch` (empty base)
- No shell (`/bin/sh`, `/bin/bash`) - prevents shell-based exploitation
- No package manager - prevents runtime package installation
- No system utilities - minimal attack surface
- Only essential libraries: libgpgme (image inspection), glibc, CA certificates
- Runs as non-root user (65532:65532)
- Explicit library dependencies documented in Dockerfile

**Benefits**:
- **Privileged eBPF daemon**: Even if compromised, attacker cannot use shell to exploit host
- **Reduced CVE exposure**: Fewer libraries = fewer security vulnerabilities
- **Compliance**: Aligns with OpenShift security best practices for minimal containers
- **Audit**: Explicit dependencies make security audits easier

**Debugging without shell**:
- Use `kubectl logs` for log inspection
- Use metrics endpoint (`:8080/metrics`) for observability
- Use Kubernetes events for status information
- Use ephemeral debug containers (Kubernetes 1.23+) if shell access needed

**Code locations**:
- Dockerfile - Multi-stage build with minimal runtime layer

## Security Best Practices for Users

**Recommendations**:
1. Limit ClusterPodPlacementConfig modification to cluster-admin
2. Regularly rotate pull secrets
3. Monitor metrics for unusual image inspection failures (may indicate registry compromise)
4. Use namespace labels to exclude sensitive namespaces from pod placement

## Related Documentation

- [ARCHITECTURE.md](../ARCHITECTURE.md) - System structure
- [RBAC Configuration](../config/rbac/) - Role definitions
- [Threat Model](./design-docs/threat-model.md) - Detailed threat analysis (if created)
