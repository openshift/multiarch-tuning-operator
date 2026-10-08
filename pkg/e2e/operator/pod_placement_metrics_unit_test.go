package operator_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestValidatePrometheusTargetFreshness(t *testing.T) {
	const namespace = "multiarch-test"
	services := []string{"pod-placement-controller", "pod-placement-web-hook"}
	initialTime := time.Date(2026, time.October, 8, 12, 0, 0, 0, time.UTC)
	freshTime := initialTime.Add(time.Minute)
	initial := []prometheusTarget{
		testPrometheusTarget(namespace, services[0], initialTime, "up", ""),
		testPrometheusTarget(namespace, services[1], initialTime, "up", ""),
	}

	tests := []struct {
		name          string
		current       []prometheusTarget
		wantKind      targetValidationKind
		wantNoFailure bool
	}{
		{
			name:     "missing expected target fails",
			current:  []prometheusTarget{testPrometheusTarget(namespace, services[0], freshTime, "up", "")},
			wantKind: targetMissing,
		},
		{
			name: "stale healthy scrape fails",
			current: []prometheusTarget{
				testPrometheusTarget(namespace, services[0], initialTime, "up", ""),
				testPrometheusTarget(namespace, services[1], initialTime, "up", ""),
			},
			wantKind: targetStale,
		},
		{
			name: "fresh unhealthy scrape fails",
			current: []prometheusTarget{
				testPrometheusTarget(namespace, services[0], freshTime, "up", ""),
				testPrometheusTarget(namespace, services[1], freshTime, "down", "connection refused"),
			},
			wantKind: targetUnhealthy,
		},
		{
			name: "fresh healthy controller and webhook scrapes pass",
			current: []prometheusTarget{
				testPrometheusTarget(namespace, services[0], freshTime, "up", ""),
				testPrometheusTarget(namespace, services[1], freshTime, "up", ""),
			},
			wantNoFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrometheusTargetFreshness(namespace, services, initial, tt.current)
			if tt.wantNoFailure {
				if err != nil {
					t.Fatalf("validatePrometheusTargetFreshness() error = %v, want nil", err)
				}
				return
			}

			var validationErr *targetValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("validatePrometheusTargetFreshness() error = %v, want target validation error", err)
			}
			if validationErr.kind != tt.wantKind {
				t.Errorf("validation error kind = %q, want %q", validationErr.kind, tt.wantKind)
			}
			if tt.wantKind == targetUnhealthy && !strings.Contains(validationErr.Error(), "connection refused") {
				t.Errorf("unhealthy target error %q does not include lastError", validationErr)
			}
		})
	}
}

func testPrometheusTarget(namespace, serviceName string, lastScrape time.Time, health, lastError string) prometheusTarget {
	return prometheusTarget{
		DiscoveredLabels: map[string]string{
			"__meta_kubernetes_namespace":    namespace,
			"__meta_kubernetes_service_name": serviceName,
		},
		Labels: map[string]string{
			"endpoint": "metrics",
			"instance": serviceName + ":8443",
		},
		ScrapePool: fmt.Sprintf("serviceMonitor/%s/%s/0", namespace, serviceName),
		ScrapeURL:  fmt.Sprintf("https://%s.%s.svc:8443/metrics", serviceName, namespace),
		Health:     health,
		LastError:  lastError,
		LastScrape: lastScrape.Format(time.RFC3339Nano),
	}
}
