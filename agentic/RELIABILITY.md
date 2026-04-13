# Reliability - multiarch-tuning-operator

## Service Level Objectives (SLOs)

### Availability
**Target**: 99.9% uptime for pod placement operands
**Measurement**: Deployment availability (availableReplicas >= desired replicas)
**Error Budget**: 43 minutes downtime per month

### Latency
**Target**: p95 pod processing time < 5 seconds (gate added to gate removed)
**Measurement**: `mto_ppo_ctrl_time_to_process_gated_pod_seconds` histogram p95

### Throughput
**Target**: Process 100 pods/second cluster-wide
**Measurement**: `rate(mto_ppo_ctrl_processed_pods_total[1m])`

### Success Rate
**Target**: >99% of pods successfully processed (image inspection succeeds)
**Measurement**: `(1 - rate(mto_ppo_ctrl_failed_image_inspection_total[1m]) / rate(mto_ppo_ctrl_processed_pods_total[1m])) * 100`

## Observability

### Metrics

All components expose Prometheus metrics at `:8080/metrics`.

**Key Metrics (Pod Placement Controller)**:
- `mto_ppo_ctrl_processed_pods_total` - Total pods processed (counter)
  - Type: Counter
  - Labels: none
  - Use: Track total workload volume

- `mto_ppo_ctrl_time_to_process_gated_pod_seconds` - Time to process gated pods (histogram)
  - Type: Histogram
  - Labels: none
  - Use: Monitor processing latency, identify slow image inspections

- `mto_ppo_ctrl_time_to_inspect_image_seconds` - Image inspection duration (histogram)
  - Type: Histogram
  - Labels: none
  - Use: Track registry API performance

- `mto_ppo_ctrl_failed_image_inspection_total` - Failed image inspections (counter)
  - Type: Counter
  - Labels: none
  - Use: Alert on high failure rates

**Key Metrics (Webhook)**:
- `mto_ppo_wh_pods_processed_total` - Total pods seen by webhook (counter)
- `mto_ppo_wh_pods_gated_total` - Total pods gated (counter)
- `mto_ppo_wh_response_time_seconds` - Webhook response time (histogram)

**Shared Metrics**:
- `mto_ppo_pods_gated` - Current number of gated pods (gauge)
  - Use: Monitor pod backlog, alert on excessive buildup

**Dashboards**:
- See docs/metrics.md for Grafana dashboard examples and queries

### Logging

**Log Levels** (configured via ClusterPodPlacementConfig.spec.logVerbosity):
- **Normal**: Errors and important state changes
- **Debug**: Detailed operation logs, useful for troubleshooting
- **Trace**: Per-pod processing details
- **TraceAll**: Full verbosity including image inspection details

**Structured Logging Fields**:
- `pod`: Pod namespace/name
- `image`: Image reference
- `architectures`: Supported architectures
- `error`: Error details

**Example Queries**:
```bash
# View operator logs
kubectl logs -n openshift-multiarch-tuning-operator deployment/multiarch-tuning-operator

# View pod controller logs with Debug level
# (Set logVerbosity: Debug in CPPC first)
kubectl logs -n openshift-multiarch-tuning-operator deployment/pod-placement-controller

# Filter for errors
kubectl logs -n openshift-multiarch-tuning-operator deployment/pod-placement-controller | grep -i error
```

### Tracing

Currently not implemented. Future enhancement tracked in tech-debt-tracker.md.

## Alerts

### Critical Alerts

**Alert**: PodPlacementControllerDown
- **Condition**: `up{job="pod-placement-controller"} == 0` for 5 minutes
- **Impact**: New pods not processed, accumulate with scheduling gate
- **Response**: Check deployment health, restart if necessary
- **Runbook**: Check operator logs for crash/restart, verify CPPC status conditions

**Alert**: HighImageInspectionFailureRate
- **Condition**: `rate(mto_ppo_ctrl_failed_image_inspection_total[5m]) / rate(mto_ppo_ctrl_processed_pods_total[5m]) > 0.1` (>10% failure rate)
- **Impact**: Pods ungated without architecture constraints, may land on wrong architecture
- **Response**: Check registry availability, verify pull-secret, review logs
- **Runbook**:
  1. Check metrics: `rate(mto_ppo_ctrl_failed_image_inspection_total[5m])`
  2. View controller logs for "failed to inspect image" errors
  3. Verify pull-secret: `kubectl get secret -n openshift-multiarch-tuning-operator pull-secret`
  4. Test registry connectivity from cluster

### Warning Alerts

**Alert**: HighPodProcessingLatency
- **Condition**: `histogram_quantile(0.95, mto_ppo_ctrl_time_to_process_gated_pod_seconds) > 10` (p95 > 10s)
- **Impact**: Slow pod scheduling, may indicate registry performance issues
- **Response**: Check image inspection latency, registry health
- **Runbook**: Review `mto_ppo_ctrl_time_to_inspect_image_seconds` metric, check registry API rate limits

**Alert**: PodBacklogBuildup
- **Condition**: `mto_ppo_pods_gated > 100` for 10 minutes
- **Impact**: Large number of pods waiting for processing
- **Response**: Check controller throughput, scale controller replicas if needed
- **Runbook**: Check controller CPU/memory usage, review processing rate metrics

## Runbooks

### High Image Inspection Failure Rate

**Symptoms**: Alert "HighImageInspectionFailureRate" firing, pods scheduling without architecture constraints

**Diagnosis**:
1. Check controller logs:
   ```bash
   kubectl logs -n openshift-multiarch-tuning-operator deployment/pod-placement-controller | grep "failed to inspect"
   ```
2. Verify pull-secret exists:
   ```bash
   kubectl get secret -n openshift-multiarch-tuning-operator pull-secret
   ```
3. Check registry connectivity:
   ```bash
   # From node
   curl -I https://registry.redhat.io/v2/
   ```

**Resolution**:
- If pull-secret missing: Verify GlobalPullSecretSyncer is running
- If registry unreachable: Check network policies, DNS resolution
- If rate-limited: Increase controller memory limit to enable larger cache
- If transient: Monitor, failures should self-recover via retry

### Controller Crash Loop

**Symptoms**: pod-placement-controller deployment not ready, pods not being processed

**Diagnosis**:
1. Check pod status:
   ```bash
   kubectl get pods -n openshift-multiarch-tuning-operator -l app=pod-placement-controller
   ```
2. View crash logs:
   ```bash
   kubectl logs -n openshift-multiarch-tuning-operator deployment/pod-placement-controller --previous
   ```

**Resolution**:
- Check for OOM: Increase memory limit
- Check for panic: File issue with stack trace
- Check RBAC: Verify ServiceAccount has required permissions

## Incident Response

1. **Detection**: Alerts fire via Prometheus Alertmanager
2. **Triage**: Check CPPC status conditions, review metrics dashboard
3. **Mitigation**: Follow runbook for specific alert
4. **Resolution**: Apply fix, verify metrics return to normal
5. **Post-mortem**: Document incident, update runbooks if needed

## Capacity Planning

**Current Capacity** (per controller replica):
- Pod processing: ~100 pods/second (limited by image inspection)
- Concurrent reconciliations: NumCPU * 4
- Memory: ~200MB baseline + cache overhead

**Growth Rate**: Linear with pod creation rate

**Bottlenecks**:
- Image inspection (external registry API calls)
- Cache size (limited by memory)
- API server throughput (list/watch)

**Scaling**:
- Horizontal: Increase controller replicas for higher throughput
- Vertical: Increase memory for larger cache, reduce registry calls

## Disaster Recovery

**Backup**: Not applicable (stateless operator, configuration in CPPC CR)

**Recovery Time Objective (RTO)**: <5 minutes
- Redeploy operator from bundle
- CPPC re-created from backup

**Recovery Point Objective (RPO)**: 0 (no data loss, stateless)

**Recovery Procedure**:
1. Reinstall operator: `kubectl apply -f operator.yaml`
2. Recreate CPPC: `kubectl apply -f clusterpodplacementconfig.yaml`
3. Verify operands deployed: `kubectl get deployments -n openshift-multiarch-tuning-operator`
4. Monitor metrics for normal operation

## Related Documentation

- [ARCHITECTURE.md](../ARCHITECTURE.md) - System structure
- [Metrics Guide](../docs/metrics.md) - Complete metrics catalog
- [DEVELOPMENT.md](./DEVELOPMENT.md#debugging) - Debugging procedures
