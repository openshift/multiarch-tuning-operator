# Technical Debt Tracker

> **Purpose**: Track known issues, workarounds, and improvements needed
> **Update**: Add new debt immediately, remove when resolved

## High Priority

### Image Inspection Caching
**Status**: Open
**Owner**: TBD
**Created**: 2026-03-30
**Impact**: Repeated image inspections increase registry API calls and slow pod processing
**Workaround**: In-memory cache within controller process
**Fix**: Implement persistent cache (Redis/etcd) shared across controller replicas
**Effort**: M
**Related**: pkg/image/inspector.go

## Medium Priority

### E2E Test Coverage Gaps
**Status**: Open
**Owner**: TBD
**Created**: 2026-03-30
**Impact**: Some failure scenarios not covered by automated tests
**Workaround**: Manual testing
**Fix**: Add E2E tests for multi-arch failure scenarios
**Effort**: S
**Related**: test/e2e/

## Low Priority / Nice to Have

### Metrics Dashboard Improvements
**Status**: Open
**Owner**: TBD
**Created**: 2026-03-30
**Impact**: Basic Prometheus metrics exist but no Grafana dashboards
**Workaround**: Manual Prometheus queries
**Fix**: Create Grafana dashboard templates
**Effort**: S
**Related**: docs/metrics.md

## Resolved (Recent)

---

## How to Use This

**Adding debt**:
1. Add to appropriate priority section
2. Fill all fields
3. Link to related issues/PRs

**Updating debt**:
1. Change status/owner as needed
2. Update workaround if changed
3. Move to "Resolved" when fixed

**Cleaning up**:
- Move resolved items after 30 days to archive
- Re-prioritize monthly
