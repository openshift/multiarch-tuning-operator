---
title: introducing-an-ebpf-based-monitoring-solution-for-enoexec
authors:
  - "@aleskandro"
reviewers:
  - "@AnnaZivkovic"
  - "@jeffdyoung"
  - "@lwan-wanglin"
  - "@Prashanth684"
  - "@prb112"
approvers:
  - "@Prashanth684"
creation-date: 2025-03-27
last-updated: 2025-03-27
tracking-link:
  - https://issues.redhat.com/browse/MULTIARCH-5010
see-also: []
---

# Introducing an eBPF-based monitoring solution for ENOEXEC (aka Exec Format Error)

## Summary

The Multiarch Tuning Operator can take care of setting up the node affinities of pods that are created in the cluster to 
ensure that they are scheduled on the most appropriate nodes based on the compatible architecture of the images in the pods.

However, when a node selector is already set for the architecture, or when the pods load binaries that are not compatible with the
architecture of the node where the pod is scheduled, failures can occur and either the pod will keep restarting (CrashLoopBackOff), or
other processes will be unable to start and trigger an `ENOEXEC` error.

This is particularly critical in scenarios where production-grade clusters are migrated to a multi-architecture setup and they 
start running nodes of different architectures.

This enhancement proposes to introduce a new eBPF-based monitoring solution that will be able to detect the `ENOEXEC` errors in the nodes,
trigger alerts and events in the pod events stream and provide metrics to the monitoring stack to allow the cluster administrators to
detect and troubleshoot these issues without further investigation in the logs, easing the migration journey and ensuring continuous monitoring
for the future workloads.

### User Stories
- As a cluster administrator, I want to know when an ENOEXEC occurs during an EXECVE syscall, either when starting a pod or when the 
main process in the pod executes a binary built for an architecture mismatching the one of the node.

### Goals
- Provide a new plugin - `eNoExecMonitoring` - in the `ClusterPodPlacementConfig` CRD that will enable the eBPF-based monitoring solution
  to detect the `ENOEXEC` errors in the nodes and trigger alerts and events in the pod events stream.
- To monitor the `ENOEXEC` errors in the cluster monitoring stack
- To quickly identify pods that are failing due to `ENOEXEC` errors and troubleshoot the issue

### Non-Goals
- Automatically resolve the cause of `ENOEXEC` errors
- Identify the architecture of the binary that is causing the `ENOEXEC` error and adjust the pod's node affinity accordingly

## Proposal

We introduce a new plugin - `eNoExecMonitoring` - in the cluster-scoped `ClusterPodPlacementConfig` CRD and a new CRD, `ENoExecEvent`.

The plugin let the operator deploy a publisher-subscriber architecture consisting of:
1. `enoexec-event-daemon`: A DaemonSet that runs an eBPF-based monitoring solution on each node in the cluster
2. `enoexec-event-handler`: A Deployment that watches for `ENoExecEvent` objects and (a) records the event in the pod's events log, (b) record the event in a Prometheus counter, (c) delete the `ENoExecEvent` object.
3. An AlertRule that triggers an alert when the Prometheus counter value is different than the past 24h value

## Architecture

<img src="https://raw.githubusercontent.com/openshift/multiarch-tuning-operator/1412ef6f47c6d577041f1e3f0bce2c183a4f2886/docs/enhancements/enoexec-monitoring-architecture.svg" alt="multiarch-tuning-operator enoexec monitoring architecture" width="100%" style="max-width: 60vw;display:block"/>

The architecture of the enoexec monitoring plugin - depicted in fig. 1 - consists of a leader-elected single consumer in a publish-subscribe system that uses the Kubernetes API and the KV Store (etcd) as the message broker. Publishers are the instances
of the `enoexec-event-daemon` DaemonSet that run on each node in the cluster and are responsible for creating the `ENoExecEvent` objects when a `ENOEXEC` error is detected in the node. The consumer is the leader-elected instance of the `enoexec-event-handler` Deployment that watches for the `ENoExecEvent` objects and records the event in the pod's events log, in a Prometheus counter and, finally, deletes the `ENoExecEvent` object.

### ENoExecEvent definition

`ENoExecEvent` is a new CRD that will be used to record the `ENOEXEC` events detected by the `enoexec-event-daemon` DaemonSet. The CRD will have the following structure in yaml format:

```yaml
apiVersion: multiarch.openshift.io/v1beta1
kind: ENoExecEvent
metadata:
  name: <uuid>
  namespace: <multiarch-tuning-operator-namespace>
spec:
status:
  nodeName: <node-name>
  podName: <pod-name>
  containerID: <container-id>
  command: <command>
```

`ENoExecEvent` are internal objects that users should not deal with. In the OLM deployments, the ClusterServiceVersion will be annotated to hide the `ENoExecEvent` CRD from the user interface:
```yaml
    operators.operatorframework.io/internal-objects: '["enoexecevents.multiarch.openshift.io"]' 
```


### enoexec-event-daemon

The `enoexec-event-daemon` runs on each node in the cluster and is responsible for detecting the `ENOEXEC` errors in the nodes and creating the `ENoExecEvent` objects when an error is detected.

It runs with a dedicated service account bound to a namespace-scoped role that only allows it to create and get `ENoExecEvent` objects in the namespace where the Multiarch Tuning Operator is running. For each failure, the `ENoExecEvent` creates a new object in the namespace where the Multiarch Tuning Operator is running, named as a random string generated by FNV, and including the fields described in the previous section.

#### Structure of the payload for the ring buffer

Each event recorded in the ring buffer has the following structure consisting of a 20-byte payload including
the `real_parent->tgid`, `current->tgid`, and `current_comm` fields:

```text
  0                   1                   2                   3   
  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 
 +-------------------------------+-------------------------------+
 |                       real_parent->tgid (4 bytes)             |
 +-------------------------------+-------------------------------+
 |                       current->tgid (4 bytes)                 |
 +-------------------------------+-------------------------------+
 |                                                               |
 |                      current->comm (16 bytes)                 |
 |                                                               |
 |                                                               |
 +---------------------------------------------------------------+
```

### enoexec-event-handler

The `enoexec-event-handler` is a Deployment that watches for `ENoExecEvent` objects and (a) records the event in the pod's events log, (b) record the event in a Prometheus counter, (c) delete the `ENoExecEvent` object.

## Changes to the `ClusterPodPlacementConfig` CRD

The eNoExecMonitoring plugin will be added to the `ClusterPodPlacementConfig` CRD. The plugin will be disabled by default and will
only include the basePlugin's `enabled` field.

```yaml
apiVersion: multiarch.openshift.io/v1beta1
kind: ClusterPodPlacementConfig
metadata:
  name: my-namespace-config
spec:
  # [...]
  plugins:
    eNoExecMonitoring: ## New plugin
      enabled: true
  # [...]
```

## Changes to the Multiarch Tuning Operator

The Multiarch Tuning Operator has to be modified to support the new `eNoExecMonitoring` plugin in the `ClusterPodPlacementConfig` CRD. The operator will be responsible for deploying the `enoexec-event-daemon` DaemonSet and the `enoexec-event-handler` Deployment when the `eNoExecMonitoring` plugin is enabled in the `ClusterPodPlacementConfig` CRD.


## Changes to the Pod Placement Controller

None

## Changes to the Mutating Webhook

None

## RBAC

The service account running the `enoexec-event-daemon` DaemonSet will be bound to a namespace-scoped role that only allows it to create and get `ENoExecEvent` objects in the namespace where the Multiarch Tuning Operator is running. The `enoexec-event-handler` Deployment will run with a service account that can read, list and delete `ENoExecEvent` objects in the namespace where the Multiarch Tuning Operator is running. It will also need
to be able to patch the pods in the cluster to add the events in the pod's events log.

### enoexec-event-daemon


| Resource                             | Methods           | comments |
|--------------------------------------|-------------------|----------|
| enoexecevents.multiarch.openshift.io | create, get, list |          |

##### enoexec-event-handler

---------------

| Resource                             | Methods                  | comments         |
|--------------------------------------|--------------------------|------------------|
| pods                                 | list, watch, get         | (cluster-scoped) | 
| events                               | create, patch            | (cluster-scoped) |
| enoexecevents.multiarch.openshift.io | delete, get, list, watch |                  |


#### Considerations about uninstallation

No changes in the current process are required. The users are still recommended to delete the Multiarch Tuning Operator resources before proceeding with the uninstallation: in the case such resources are not deleted before the operator is uninstalled, the operands may continue to run correctly, but changes to the configuration objects will not guarantee the expected behavior.


### Implementation Details/Notes/Constraints

- The `enoexec-event-daemon` DaemonSet will be deployed in the same namespace as the Multiarch Tuning Operator. It should be possible to let the eBPF program to  run in all the supported architectures, taking care of the kernel vesions and other architecture-specific issues like endianess.
- The `enoexec-event-daemon`'s eBPF program will return the PID of the process that triggered the `ENOEXEC` error and the PID of the parent process. The `enoexec-event-daemon` will use the PID of the parent process to get the cgroup of the process and infer the pod uuid from it. The Kubernetes API does not allow to get a pod by UUID (see https://github.com/kubernetes/kubernetes/issues/20572). Therefore, the `enoexec-event-daemon` running in the node where ENOEXEC is detected will use the CRI API to get the pod name and namespace given its UUID and build the `ENoExecEvent` object with the pod name and node name. The `enoexec-event-handler` will be responsible for generating the event in the pod's events log
- The container id can be used by the `enoexec-event-handler` to infer the name of the container affected by the issue by searching for the container id in the pod's status field
- The name of the binary that triggered the `ENOEXEC` error can be obtained from the `enoexec-event-daemon`'s eBPF program by using the `bpf_get_current_comm` helper. The name may not include the full path of the binary, but it can be used to identify the binary that triggered the error.


### Risks and Mitigations

#### Security considerations

In order to run the `enoexec-event-daemon`, the pod must be run with the additional capabilities: CAP_BPF, CAP_PERFMON, and CAP_SYS_RESOURCE. However, as we
also need to attach the tracepoint and CoreOS kernels enable set perf_event_paranoid to 2, disallowing most tracepoint usage for non-root users.
We need to run the container running the eBPF program as root. This is an assessed risk.

To mitigate this risk, we might use a dedicated container and minimal container image for the eBPF program and share a FIFO pipe with the main container running as a non-root user and responsible for interacting with the Kubernetes API. The main container would be responsible for consuming events in the FIFO pipe and creating the `ENoExecEvent` objects. We will evaluate this option in the implementation phase.

### Drawbacks

### Open Questions

### Test Plan

#### Unit Testing and Integration Test Suites

- Unit Testing: Test each new function, method, and feature in isolation to ensure correctness, reliability, and
  robustness. Verify that the new code paths are covered by the unit tests and that the code behaves as expected
  under different conditions.
- Integration Test Suite: Run integration tests against a simulated control plane using the operator SDK's envtest
  facilities. We will add the necessary test cases to ensure the reconciliation loop of the new `ENoExecEvent` CRD is correctly handled

#### Functional Test Suite

- Tests should be performed in all the architectures supported by the Multiarch Tuning Operator (x86_64, aarch64, ppc64le, s390x), for the `enoexec-event-daemon` DaemonSet in particular. We should make sure no issues arise when the eBPF program is run in different architectures, for example with regard to Kernel versions and endianness.

### Graduation Criteria

The `ENoExecEvent` CRD will be introduced as `v1beta1`.

### Upgrade / Downgrade Strategy
- No special upgrade/downgrade strategy is required for this enhancement. Users will update the operator to the new version
  including the support for the new plugin.

### Version Skew Strategy

### Operational Aspects of API Extensions

#### Failure Modes
- Race condition between the kernel-space eBPF program and the user-space software tracing the cgroup of the processes to 
infer the pod uuid. It's a rare condition that can happen if the pod (and related processes) is deleted in the time between the 
generation of the payload in the kernel-space and the consumption of it in the user-space. In such cases, the `enoexec-event-daemon` might not be able to infer the pod uuid and will set the `podName` field to the empty string `""`. The `enoexec-event-handler` Deployment will be able to handle this condition by ignoring the target pod for the created event. A dedicate metric will be exposed to the monitoring stack for tracking this failure signal.

## Documentation Plan

A new section will be added to the Multiarch Tuning Operator documentation to explain the new plugin and its configuration.

## Implementation History

## Infrastructure Needed

## Open Questions

- Should we harden the security of the `enoexec-event-daemon` DaemonSet by running a non-privileged container that interacts with the Kubernetes API and a privileged one for the eBPF program, possibly using a minimal image that only includes the eBPF program and dependencies (no shell, no package manager, etc.)?