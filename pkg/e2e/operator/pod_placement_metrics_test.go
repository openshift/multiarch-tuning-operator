package operator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	prometheusPortForwardStartupTimeout = 30 * time.Second
	prometheusPortForwardCleanupTimeout = 10 * time.Second
	prometheusTargetsRequestTimeout     = 15 * time.Second
	prometheusScrapeCheckTimeout        = 5 * time.Minute
	prometheusScrapePollInterval        = 5 * time.Second
)

type prometheusTargetsResponse struct {
	Status string `json:"status"`
	Data   struct {
		ActiveTargets []prometheusTarget `json:"activeTargets"`
	} `json:"data"`
}

type prometheusTarget struct {
	DiscoveredLabels map[string]string `json:"discoveredLabels"`
	Labels           map[string]string `json:"labels"`
	ScrapePool       string            `json:"scrapePool"`
	ScrapeURL        string            `json:"scrapeUrl"`
	Health           string            `json:"health"`
	LastError        string            `json:"lastError"`
	LastScrape       string            `json:"lastScrape"`
}

type prometheusPortForward struct {
	podName string
	address string
	cancel  context.CancelFunc
	done    chan struct{}
	err     error
	output  *portForwardOutput

	stopOnce    sync.Once
	cleanupDone chan struct{}
	cleanupErr  error
}

// portForwardOutput captures both command streams safely and reports the
// dynamically allocated local port once oc announces that forwarding is ready.
type portForwardOutput struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	ready chan string
}

func (output *portForwardOutput) Write(p []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	n, err := output.buf.Write(p)
	for _, line := range strings.Split(output.buf.String(), "\n") {
		var port int
		if _, scanErr := fmt.Sscanf(line, "Forwarding from 127.0.0.1:%d -> 9090", &port); scanErr == nil && port > 0 && port <= 65535 {
			select {
			case output.ready <- fmt.Sprintf("http://127.0.0.1:%d", port):
			default:
			}
		}
	}
	return n, err
}

func (output *portForwardOutput) String() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.buf.String()
}

func startPrometheusPortForward(ctx context.Context, clientset *kubernetes.Clientset) (*prometheusPortForward, error) {
	pods, err := clientset.CoreV1().Pods(openshiftMonitoringNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=prometheus,app.kubernetes.io/instance=k8s",
	})
	if err != nil {
		return nil, fmt.Errorf("list cluster-Prometheus pods in %s: %w", openshiftMonitoringNamespace, err)
	}

	var readyPod *corev1.Pod
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
			continue
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				readyPod = pod
				break
			}
		}
		if readyPod != nil {
			break
		}
	}
	if readyPod == nil {
		return nil, fmt.Errorf("no Ready, non-terminating cluster-Prometheus pod found in %s with labels app.kubernetes.io/name=prometheus,app.kubernetes.io/instance=k8s", openshiftMonitoringNamespace)
	}

	ginkgo.By("Checking the CI identity's pods/portforward authorization in openshift-monitoring")
	review, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   openshiftMonitoringNamespace,
				Verb:        "create",
				Group:       "",
				Resource:    "pods",
				Subresource: "portforward",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("submit SelfSubjectAccessReview for pods/portforward in %s: %w", openshiftMonitoringNamespace, err)
	}
	if !review.Status.Allowed {
		return nil, fmt.Errorf("authorization denied for create pods/portforward in %s: reason=%q evaluationError=%q", openshiftMonitoringNamespace, review.Status.Reason, review.Status.EvaluationError)
	}

	// oc is already required by the E2E deployment script. Inherit its
	// KUBECONFIG rather than introducing client-go's streaming dependencies.
	commandCtx, cancel := context.WithCancel(ctx)
	forward := &prometheusPortForward{
		podName:     readyPod.Name,
		cancel:      cancel,
		done:        make(chan struct{}),
		output:      &portForwardOutput{ready: make(chan string, 1)},
		cleanupDone: make(chan struct{}),
	}
	command := exec.CommandContext(commandCtx, "oc", "--namespace", openshiftMonitoringNamespace,
		"port-forward", "pod/"+readyPod.Name, ":9090", "--address=127.0.0.1")
	// Use POST/SPDY consistently with the authorization check above, including
	// with newer oc clients whose default streaming transport is WebSocket.
	command.Env = append(os.Environ(), "KUBECTL_PORT_FORWARD_WEBSOCKETS=false")
	command.Stdout = forward.output
	command.Stderr = forward.output
	command.WaitDelay = prometheusPortForwardCleanupTimeout
	if err := command.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("port-forward startup failed for Prometheus pod %s: %w", readyPod.Name, err)
	}

	go func() {
		forward.err = command.Wait()
		close(forward.done)
	}()
	ginkgo.DeferCleanup(func() {
		if err := forward.stopAndWait(); err != nil {
			ginkgo.Fail(err.Error())
		}
	})

	startupTimer := time.NewTimer(prometheusPortForwardStartupTimeout)
	defer startupTimer.Stop()
	select {
	case address := <-forward.output.ready:
		forward.address = address
		return forward, nil
	case <-forward.done:
		return nil, fmt.Errorf("port-forward startup failed for Prometheus pod %s: %s", readyPod.Name, forward.terminationDetails())
	case <-startupTimer.C:
		return nil, fmt.Errorf("port-forward startup timed out after %s for Prometheus pod %s", prometheusPortForwardStartupTimeout, readyPod.Name)
	case <-ctx.Done():
		return nil, fmt.Errorf("port-forward startup canceled for Prometheus pod %s: %w", readyPod.Name, ctx.Err())
	}
}

func (forward *prometheusPortForward) stopAndWait() error {
	forward.stopOnce.Do(func() {
		forward.cancel()
		timer := time.NewTimer(prometheusPortForwardCleanupTimeout)
		defer timer.Stop()
		select {
		case <-forward.done:
		case <-timer.C:
			forward.cleanupErr = fmt.Errorf("port-forward cleanup timed out after %s for Prometheus pod %s", prometheusPortForwardCleanupTimeout, forward.podName)
		}
		close(forward.cleanupDone)
	})
	<-forward.cleanupDone
	return forward.cleanupErr
}

func (forward *prometheusPortForward) terminationError() error {
	select {
	case <-forward.done:
		return fmt.Errorf("port-forward terminated unexpectedly for Prometheus pod %s: %s", forward.podName, forward.terminationDetails())
	default:
		return nil
	}
}

func (forward *prometheusPortForward) terminationDetails() string {
	// Call only after done is closed, which synchronizes access to err.
	if forward.err != nil {
		return fmt.Sprintf("%v; output=%q", forward.err, forward.output.String())
	}
	return fmt.Sprintf("forwarder exited without an error; output=%q", forward.output.String())
}

func verifyFreshPrometheusScrapes(ctx context.Context, forward *prometheusPortForward, serviceNames ...string) error {
	expected := prometheusExpectations(utils.Namespace(), serviceNames)
	deadline := time.Now().Add(prometheusScrapeCheckTimeout)
	var initialTargets []prometheusTarget
	var lastMissing []string

	for len(initialTargets) == 0 {
		targets, err := queryPrometheusTargets(ctx, forward)
		if err != nil {
			return err
		}
		lastMissing = missingExpectedTargets(utils.Namespace(), expected, targets)
		if len(lastMissing) == 0 {
			initialTargets = targets
			break
		}
		if time.Until(deadline) <= 0 {
			return fmt.Errorf("missing expected Prometheus targets before timeout: %s", strings.Join(lastMissing, ", "))
		}
		if err := waitForNextScrapePoll(ctx, deadline); err != nil {
			return fmt.Errorf("missing expected Prometheus targets before timeout: %s", strings.Join(lastMissing, ", "))
		}
	}

	for {
		targets, err := queryPrometheusTargets(ctx, forward)
		if err != nil {
			return err
		}
		initialTargets = appendNewTargetsToBaseline(utils.Namespace(), serviceNames, initialTargets, targets)
		err = validatePrometheusTargetFreshness(utils.Namespace(), serviceNames, initialTargets, targets)
		if err == nil {
			return nil
		}
		var validationErr *targetValidationError
		if !errors.As(err, &validationErr) || validationErr.kind == targetUnhealthy || validationErr.kind == targetInvalidScrapeTime {
			return err
		}
		if time.Until(deadline) <= 0 {
			if validationErr.kind == targetMissing {
				return fmt.Errorf("missing expected Prometheus targets before timeout: %s", validationErr.Error())
			}
			return fmt.Errorf("no fresh Prometheus scrape before timeout: %s", validationErr.Error())
		}
		if err := waitForNextScrapePoll(ctx, deadline); err != nil {
			if validationErr.kind == targetMissing {
				return fmt.Errorf("missing expected Prometheus targets before timeout: %s", validationErr.Error())
			}
			return fmt.Errorf("no fresh Prometheus scrape before timeout: %s", validationErr.Error())
		}
	}
}

func appendNewTargetsToBaseline(namespace string, serviceNames []string, initialTargets, currentTargets []prometheusTarget) []prometheusTarget {
	knownTargets := make(map[string]struct{}, len(initialTargets))
	for _, target := range initialTargets {
		knownTargets[prometheusTargetIdentity(target)] = struct{}{}
	}
	for _, expectation := range prometheusExpectations(namespace, serviceNames) {
		for _, target := range matchingPrometheusTargets(namespace, expectation, currentTargets) {
			identity := prometheusTargetIdentity(target)
			if _, known := knownTargets[identity]; known {
				continue
			}
			initialTargets = append(initialTargets, target)
			knownTargets[identity] = struct{}{}
		}
	}
	return initialTargets
}

func queryPrometheusTargets(ctx context.Context, forward *prometheusPortForward) ([]prometheusTarget, error) {
	if err := forward.terminationError(); err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, prometheusTargetsRequestTimeout)
	defer cancel()
	targetsURL, err := url.Parse(forward.address + "/api/v1/targets?state=active")
	if err != nil {
		return nil, fmt.Errorf("build Prometheus targets URL: %w", err)
	}
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, targetsURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build Prometheus targets request: %w", err)
	}
	response, err := (&http.Client{Timeout: prometheusTargetsRequestTimeout}).Do(request)
	if err != nil {
		if terminationErr := forward.terminationError(); terminationErr != nil {
			return nil, terminationErr
		}
		return nil, fmt.Errorf("Prometheus targets HTTP request failed: %w", err)
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read Prometheus targets HTTP response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Prometheus targets HTTP response body: %w", closeErr)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Prometheus targets HTTP request returned status %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}
	var targetsResponse prometheusTargetsResponse
	if err := json.Unmarshal(responseBody, &targetsResponse); err != nil {
		return nil, fmt.Errorf("decode Prometheus targets JSON: %w", err)
	}
	if targetsResponse.Status != "success" {
		return nil, fmt.Errorf("Prometheus targets API returned status %q", targetsResponse.Status)
	}
	return targetsResponse.Data.ActiveTargets, nil
}

func waitForNextScrapePoll(ctx context.Context, deadline time.Time) error {
	wait := prometheusScrapePollInterval
	if remaining := time.Until(deadline); remaining < wait {
		wait = remaining
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type prometheusTargetExpectation struct {
	serviceName string
	scrapePool  string
	endpoint    string
}

type targetValidationKind string

const (
	targetMissing           targetValidationKind = "missing"
	targetUnhealthy         targetValidationKind = "unhealthy"
	targetStale             targetValidationKind = "stale"
	targetInvalidScrapeTime targetValidationKind = "invalid-scrape-time"
)

type targetValidationError struct {
	kind targetValidationKind
	msg  string
}

func (err *targetValidationError) Error() string {
	return err.msg
}

func prometheusExpectations(namespace string, serviceNames []string) []prometheusTargetExpectation {
	expected := make([]prometheusTargetExpectation, 0, len(serviceNames))
	for _, serviceName := range serviceNames {
		expected = append(expected, prometheusTargetExpectation{
			serviceName: serviceName,
			scrapePool:  fmt.Sprintf("serviceMonitor/%s/%s/0", namespace, serviceName),
			endpoint:    "metrics",
		})
	}
	return expected
}

func missingExpectedTargets(namespace string, expected []prometheusTargetExpectation, targets []prometheusTarget) []string {
	var missing []string
	for _, expectation := range expected {
		if len(matchingPrometheusTargets(namespace, expectation, targets)) == 0 {
			missing = append(missing, fmt.Sprintf("service=%s scrapePool=%s", expectation.serviceName, expectation.scrapePool))
		}
	}
	return missing
}

func matchingPrometheusTargets(namespace string, expectation prometheusTargetExpectation, targets []prometheusTarget) []prometheusTarget {
	var matches []prometheusTarget
	for _, target := range targets {
		targetNamespace := target.DiscoveredLabels["__meta_kubernetes_namespace"]
		if targetNamespace == "" {
			targetNamespace = target.Labels["namespace"]
		}
		targetService := target.DiscoveredLabels["__meta_kubernetes_service_name"]
		if targetService == "" {
			targetService = target.Labels["service"]
		}
		if targetNamespace != namespace || targetService != expectation.serviceName {
			continue
		}
		if target.ScrapePool != "" && target.ScrapePool != expectation.scrapePool {
			continue
		}
		if endpoint := target.Labels["endpoint"]; endpoint != "" && endpoint != expectation.endpoint {
			continue
		}
		matches = append(matches, target)
	}
	return matches
}

func validatePrometheusTargetFreshness(namespace string, serviceNames []string, initialTargets, currentTargets []prometheusTarget) error {
	for _, expectation := range prometheusExpectations(namespace, serviceNames) {
		initial := matchingPrometheusTargets(namespace, expectation, initialTargets)
		if len(initial) == 0 {
			return &targetValidationError{kind: targetMissing, msg: fmt.Sprintf("missing initial Prometheus target for service %s (expected scrapePool %s)", expectation.serviceName, expectation.scrapePool)}
		}
		baseline := make(map[string]time.Time, len(initial))
		for _, target := range initial {
			timestamp, err := parsePrometheusLastScrape(target.LastScrape)
			if err != nil {
				return &targetValidationError{kind: targetInvalidScrapeTime, msg: fmt.Sprintf("invalid initial lastScrape for service %s: %v", expectation.serviceName, err)}
			}
			baseline[prometheusTargetIdentity(target)] = timestamp
		}

		current := matchingPrometheusTargets(namespace, expectation, currentTargets)
		if len(current) == 0 {
			return &targetValidationError{kind: targetMissing, msg: fmt.Sprintf("missing expected Prometheus target for service %s (expected scrapePool %s)", expectation.serviceName, expectation.scrapePool)}
		}
		for _, target := range current {
			if target.Health != "up" {
				return &targetValidationError{kind: targetUnhealthy, msg: fmt.Sprintf("unhealthy Prometheus target for service %s: target=%q health=%q lastScrape=%q lastError=%q", expectation.serviceName, prometheusTargetIdentity(target), target.Health, target.LastScrape, target.LastError)}
			}
		}
		fresh := false
		staleDetails := make([]string, 0, len(current))
		for _, target := range current {
			previousScrape, exists := baseline[prometheusTargetIdentity(target)]
			if !exists {
				continue
			}
			lastScrape, err := parsePrometheusLastScrape(target.LastScrape)
			if err != nil {
				return &targetValidationError{kind: targetInvalidScrapeTime, msg: fmt.Sprintf("invalid lastScrape for service %s: %v", expectation.serviceName, err)}
			}
			if lastScrape.After(previousScrape) {
				fresh = true
				break
			}
			staleDetails = append(staleDetails, fmt.Sprintf("target=%q lastScrape=%q initialLastScrape=%q lastError=%q", prometheusTargetIdentity(target), target.LastScrape, previousScrape.Format(time.RFC3339Nano), target.LastError))
		}
		if !fresh {
			return &targetValidationError{kind: targetStale, msg: fmt.Sprintf("target for service %s has not advanced beyond its initial lastScrape: %s", expectation.serviceName, strings.Join(staleDetails, "; "))}
		}
	}
	return nil
}

func parsePrometheusLastScrape(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %q: %w", value, err)
	}
	return timestamp, nil
}

func prometheusTargetIdentity(target prometheusTarget) string {
	return strings.Join([]string{target.ScrapePool, target.ScrapeURL, target.Labels["instance"]}, "\x00")
}
