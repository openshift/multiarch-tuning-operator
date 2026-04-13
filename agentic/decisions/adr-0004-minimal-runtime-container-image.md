---
id: ADR-0004
title: Minimal Runtime Container Image
date: 2026-03-30
status: accepted
deciders: [team-name, @user]
supersedes: []
superseded-by: []
---

# Minimal Runtime Container Image

## Status

Accepted

## Context

The multiarch-tuning-operator currently uses `quay.io/centos/centos:stream9-minimal` as its runtime base image. While "minimal", this image still includes:

- A shell (`/bin/sh`, `/bin/bash`)
- Package manager (`microdnf`)
- Core utilities (`ls`, `cat`, `grep`, etc.)
- Unnecessary system libraries

This creates security concerns:

1. **eBPF Daemon Runs Privileged**: The enoexec-daemon runs in a privileged container to load eBPF programs. A shell in this container allows attackers to execute arbitrary commands with host-level access if they gain pod access.

2. **Controllers Handle Sensitive Data**: Pod placement controllers access pull secrets (container registry credentials) and have cluster-wide pod mutation capabilities. Shell access could allow credential exfiltration or malicious pod mutations.

3. **Increased Attack Surface**: Every binary and library in the container is a potential attack vector. Security vulnerabilities (CVEs) in unused utilities still expose the cluster to risk.

4. **OpenShift Security Standards**: OpenShift recommends minimal container images without shells for production operators.

## Decision

We will implement a multi-stage Docker build that creates a minimal runtime layer containing **only**:

1. The operator binaries (`manager`, `enoexec-daemon`)
2. Required shared libraries (`libgpgme`, `glibc`, dependencies)
3. CA certificates for TLS (`/etc/ssl/certs/`)
4. Minimal user/group configuration (`/etc/passwd`, `/etc/group`)

The runtime stage will be based on `scratch` (empty base image) with explicit library copying.

## Rationale

### Why This?

**Security Hardening:**
- **No shell**: Prevents shell-based exploitation even if attacker gains pod access
- **No package manager**: Prevents runtime package installation or exploitation of package manager vulnerabilities
- **Minimal libraries**: Reduces CVE exposure to only essential dependencies
- **Explicit dependencies**: Every file in the image is intentionally included and auditable

**Compliance:**
- Aligns with OpenShift security best practices
- Reduces compliance audit scope (fewer binaries to verify)
- Easier to pass security scans (fewer CVEs from unused packages)

**Principle of Least Privilege:**
- Container has only capabilities needed to run the operator
- Follows "deny by default" security model
- Explicit rather than implicit (scratch + copied files vs base image)

### Why Not Alternatives?

**Alternative A: Keep current centos:stream9-minimal**
- **Why rejected**: Contains shell and unnecessary utilities, fails security hardening goals
- **Drawback**: Leaves attack surface for privileged eBPF daemon

**Alternative B: Use Google Distroless**
- **Why rejected**: May not have compatible glibc version or all gpgme dependencies
- **Drawback**: Less control over exact libraries included
- **Note**: Could be future iteration if dependency compatibility confirmed

**Alternative C: Use UBI-minimal (Red Hat Universal Base Image)**
- **Why rejected**: Still includes shell and package manager
- **Drawback**: Not significantly better than current centos:stream9-minimal for this use case

**Alternative D: Alpine Linux**
- **Why rejected**: Uses musl libc instead of glibc, would require rebuilding binaries
- **Drawback**: Compatibility issues with containers/image library (built against glibc)

## Consequences

### Positive

- ✅ **Significantly reduced attack surface**: No shell, no package manager, no utilities
- ✅ **Better security posture for privileged eBPF daemon**: Limits exploitation even if pod compromised
- ✅ **Reduced CVE count**: Fewer libraries = fewer vulnerabilities
- ✅ **Smaller image size**: Only essential files included
- ✅ **Explicit dependencies**: Clear documentation of what's needed at runtime
- ✅ **Compliance friendly**: Easier to pass security audits and scans

### Negative

- ❌ **Harder to debug**: No shell means no `kubectl exec` debugging (must use logs/metrics)
- ❌ **More complex Dockerfile**: Multi-stage build with explicit library extraction
- ❌ **Build time increases slightly**: Additional stage for dependency extraction
- ❌ **Maintenance burden**: Must update library list if new dependencies added

### Neutral

- ℹ️ **No runtime behavior change**: Operator functionality unchanged
- ℹ️ **Same build tooling**: Still uses make, docker/podman, same build args
- ℹ️ **Transparent to users**: API and behavior identical

## Implementation

**Location**: `Dockerfile` (root of repository)

**Migration**:
- Single PR with Dockerfile changes
- Existing deployments updated on next operator upgrade
- No manual migration needed (transparent image change)

**Rollout**:
1. Implement multi-stage Dockerfile
2. Test locally with `make docker-build` and `make deploy`
3. Run full test suite (`make test`, `make e2e`)
4. Verify with security scanner (trivy/grype)
5. Merge and build production images
6. Deploy via standard operator upgrade process

**Required runtime dependencies** (extracted from build stage):
```
/lib64/ld-linux-x86-64.so.2      # Dynamic linker
/lib64/libc.so.6                  # GNU C Library
/lib64/libgpgme.so.11            # GPGME library (image inspection)
/lib64/libassuan.so.0            # Dependency of libgpgme
/lib64/libgpg-error.so.0         # Dependency of libgpgme
/etc/ssl/certs/                   # CA certificates for TLS
/etc/passwd                       # User configuration (non-root)
/etc/group                        # Group configuration
```

**Dockerfile structure**:
```dockerfile
# Stage 1: Build binaries (unchanged)
FROM golang:1.23 as builder
# ... existing build steps ...

# Stage 2: Extract runtime dependencies
FROM centos:stream9-minimal as runtime-deps
# Copy and extract only needed libraries

# Stage 3: Final minimal runtime
FROM scratch
COPY --from=runtime-deps /runtime-root/ /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/enoexec-daemon .
USER 65532:65532
ENTRYPOINT ["/manager"]
```

## Alternatives Considered

### Alternative 1: Separate Dockerfiles for Each Binary
**Pros**: Could further optimize each binary's dependencies
**Cons**: More maintenance, both binaries need same libraries anyway
**Why rejected**: Single Dockerfile simpler, binaries share dependencies

### Alternative 2: Gradual Approach (Smaller Base First)
**Pros**: Lower risk, incremental improvement
**Cons**: Doesn't achieve security goal (still has shell)
**Why rejected**: Going straight to minimal achieves security goals immediately

### Alternative 3: Debug and Production Variants
**Pros**: Production minimal, debug with shell for troubleshooting
**Cons**: Two images to maintain, could accidentally deploy debug in production
**Why rejected**: Observability should rely on logs/metrics, not shell access

## References

- [SECURITY.md](../../SECURITY.md) - Security model and threat analysis
- [Core Beliefs](../../design-docs/core-beliefs.md#non-negotiable-constraints) - Security as non-negotiable
- [Execution Plan](../../exec-plans/active/minimal-runtime-container-image.md) - Implementation plan
- [Google Distroless](https://github.com/GoogleContainerTools/distroless) - Alternative minimal base
- [OpenShift Security Best Practices](https://docs.openshift.com/container-platform/latest/security/container_security/security-platform.html)

## Notes

**Debugging without shell:**

Since the runtime image has no shell, debugging must use alternative approaches:

1. **Logs**: `kubectl logs -f <pod>` - primary debugging method
2. **Metrics**: Prometheus metrics at `:8080/metrics`
3. **Events**: `kubectl get events --sort-by='.lastTimestamp'`
4. **Remote debugging**: Could add delve debugger in debug builds if needed
5. **Ephemeral containers**: Kubernetes 1.23+ allows attaching debug containers with tools

**Library dependency discovery:**

To identify required libraries, use:
```bash
# Build binary locally
make build

# Check dependencies
ldd ./_output/bin/manager
ldd ./_output/bin/enoexec-daemon

# Copy all transitive dependencies
```

**Multi-architecture considerations:**

The library paths and dependencies may vary by architecture (amd64, arm64, ppc64le, s390x). The Dockerfile must handle this with build args (`TARGETARCH`).

**Future enhancements:**

- Consider static linking Go binary to eliminate glibc dependency (would need custom containers/image build)
- Explore distroless once library compatibility verified
- Implement automated dependency tracking in CI
