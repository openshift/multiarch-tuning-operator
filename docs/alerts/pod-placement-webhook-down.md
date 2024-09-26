# PodPlacementWebhookDown

## Meaning

The `PodPlacementWebhookDown` alert is triggered when the Pod Placement Webhook deployment has no available
and ready replicas for more than 5 minute to patch the pods with the scheduling gate and request the architecture-aware
pod placement.

## Impact

Any newly created pods will be scheduled without any automated set architecture-specific constraints in the node affinity.
Therefore, pods may be scheduled on nodes that do not have the required architecture and either fail to start, 
leading to CrashLoopBackOff, or will be stuck in ImagePullBackOff because no matching image is available for the architecture
of the node where the pod is scheduled.

## Diagnosis

### Pod placement webhook deployment

Check the status of the pod placement controller deployment.

```shell
oc describe deployment -n openshift-multiarch-tuning-operator pod-placement-webhook
```

Check the logs of the pod placement webhook pods

```shell
oc logs -n openshift-multiarch-tuning-operator pod-placement-webhook-<hash>
```

Check the mutating webhook configurations:

```shell
oc get mutatingwebhookconfigurations
```

### Mitigation

A mitigation depends on the root cause of the issue, as investigated in the diagnosis steps.
Moreover, since this alert is triggered when the Pod Placement Webhook deployment has no available and ready replicas,
if the Pod Placement Controller is not down, there is no risk of scheduling gated pods being stuck longer than expected
in the `Pending` phase and unable to be scheduled and run.