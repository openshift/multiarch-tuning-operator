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
	unhealthyReplica := testPrometheusTarget(namespace, services[0], freshTime, "down", "connection refused")
	unhealthyReplica.Labels["instance"] = "replica-2:8443"
	unhealthyReplica.ScrapeURL = "https://replica-2:8443/metrics"
	healthyReplica := testPrometheusTarget(namespace, services[0], initialTime, "up", "")
	healthyReplica.Labels["instance"] = "replica-2:8443"
	healthyReplica.ScrapeURL = "https://replica-2:8443/metrics"
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
			name:     "fresh healthy replica followed by unhealthy replica fails",
			current:  []prometheusTarget{testPrometheusTarget(namespace, services[0], freshTime, "up", ""), unhealthyReplica, testPrometheusTarget(namespace, services[1], freshTime, "up", "")},
			wantKind: targetUnhealthy,
		},
		{
			name:     "unhealthy replica followed by fresh healthy replica fails",
			current:  []prometheusTarget{unhealthyReplica, testPrometheusTarget(namespace, services[0], freshTime, "up", ""), testPrometheusTarget(namespace, services[1], freshTime, "up", "")},
			wantKind: targetUnhealthy,
		},
		{
			name:          "all healthy replicas with one advancing scrape per service pass",
			current:       []prometheusTarget{healthyReplica, testPrometheusTarget(namespace, services[0], freshTime, "up", ""), testPrometheusTarget(namespace, services[1], freshTime, "up", "")},
			wantNoFailure: true,
		},
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
			if tt.wantKind == targetUnhealthy {
				for _, target := range tt.current {
					if target.Health == "up" {
						continue
					}
					for _, detail := range []string{target.DiscoveredLabels["__meta_kubernetes_service_name"], target.Labels["instance"], target.LastScrape, target.LastError} {
						if !strings.Contains(validationErr.Error(), detail) {
							t.Errorf("unhealthy diagnostic %q missing %q", validationErr, detail)
						}
					}
				}
			}
		})
	}
}

func TestPortForwardOutput(t *testing.T) {
	output := &portForwardOutput{ready: make(chan string, 1)}
	// oc output may arrive in separate writes. An incomplete readiness message
	// must not be interpreted as an allocated port.
	if _, err := output.Write([]byte("Forwarding from 127.0.0.1:45")); err != nil {
		t.Fatal(err)
	}
	select {
	case address := <-output.ready:
		t.Fatalf("reported ready before the complete message: %s", address)
	default:
	}
	if _, err := output.Write([]byte("123 -> 9090\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case address := <-output.ready:
		if address != "http://127.0.0.1:45123" {
			t.Fatalf("allocated address = %q", address)
		}
	default:
		t.Fatal("complete oc readiness message did not report an address")
	}

	diagnostic := &portForwardOutput{ready: make(chan string, 1)}
	message := "error: pods/portforward is forbidden\n"
	if _, err := diagnostic.Write([]byte(message)); err != nil {
		t.Fatal(err)
	}
	if diagnostic.String() != message {
		t.Fatal("oc error output was not preserved")
	}
	select {
	case address := <-diagnostic.ready:
		t.Fatalf("error output reported readiness: %s", address)
	default:
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
