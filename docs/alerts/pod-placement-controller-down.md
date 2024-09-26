# PodPlacementControllerDown

## Meaning

The `PodPlacementControllerDown` alert is triggered when the Pod Placement Controller deployment has no available 
and ready replicas for more than 1 minute to serve the reconciliation of pods and remove the scheduling gate from them.

## Impact

Any newly created pods that have the `multiarch.openshift.io/scheduling-gate` scheduling gate will be stuck in the
`Pending` phase with reason `SchedulingGated`. 

## Diagnosis

### Pod placement controller deployment

Check the status of the pod placement controller deployment.

```shell
oc describe deployment -n openshift-multiarch-tuning-operator pod-placement-controller
```

Check the logs of the pod placement controller leader.

```shell
oc logs -n openshift-multiarch-tuning-operator pod-placement-controller-<hash>
```

Check for the pods that are stuck in the `Pending` phase with reason `SchedulingGated`.

```shell
oc get pods -A --field-selector status.phase=Pending | grep SchedulingGated # All the scheduling gated pods
oc get pods -A -l "multiarch.openshift.io/scheduling-gate=gated" # Alternative: all pods with the multiarch-tuning-operator scheduling gate
```

Manually remove the scheduling gate from the pods that are stuck in the `Pending` phase with reason `SchedulingGated`.

```shell
oc get pods -A -l multiarch.openshift.io/scheduling-gate=gated -o json  | jq 'del(.items[].spec.schedulingGates[] | select(.name=="multiarch.openshift.io/scheduling-gate"))' | oc apply -f -
```

### Mitigation

If the pod placement controller deployment does not back up given the previous checks and diagnosis,
you can temporarily remove the ClusterPodPlacementConfig/cluster object to disable the architecture-aware pod scheduling:
```shell
oc delete clusterpodplacementconfigs/cluster
```
