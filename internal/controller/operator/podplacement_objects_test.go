package operator

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

func TestBuildNetworkPolicyPodPlacement(t *testing.T) {
	np := buildNetworkPolicyPodPlacement()

	t.Run("metadata", func(t *testing.T) {
		if np.Name != utils.PodPlacementNetworkPolicyName {
			t.Errorf("expected name %q, got %q", utils.PodPlacementNetworkPolicyName, np.Name)
		}
		if np.Namespace != utils.Namespace() {
			t.Errorf("expected namespace %q, got %q", utils.Namespace(), np.Namespace)
		}
		if np.Labels[utils.OperandLabelKey] != operandName {
			t.Errorf("expected label %s=%s, got %s", utils.OperandLabelKey, operandName, np.Labels[utils.OperandLabelKey])
		}
		if np.Labels[utils.ControllerNameKey] != utils.PodPlacementControllerName {
			t.Errorf("expected label %s=%s, got %s", utils.ControllerNameKey, utils.PodPlacementControllerName, np.Labels[utils.ControllerNameKey])
		}
	})

	t.Run("pod selector", func(t *testing.T) {
		if np.Spec.PodSelector.MatchLabels[utils.OperandLabelKey] != operandName {
			t.Errorf("expected pod selector %s=%s, got %s", utils.OperandLabelKey, operandName, np.Spec.PodSelector.MatchLabels[utils.OperandLabelKey])
		}
	})

	t.Run("policy types", func(t *testing.T) {
		if len(np.Spec.PolicyTypes) != 2 {
			t.Fatalf("expected 2 policy types, got %d", len(np.Spec.PolicyTypes))
		}
		if np.Spec.PolicyTypes[0] != networkingv1.PolicyTypeIngress {
			t.Errorf("expected first policy type Ingress, got %s", np.Spec.PolicyTypes[0])
		}
		if np.Spec.PolicyTypes[1] != networkingv1.PolicyTypeEgress {
			t.Errorf("expected second policy type Egress, got %s", np.Spec.PolicyTypes[1])
		}
	})

	t.Run("ingress rules", func(t *testing.T) {
		if len(np.Spec.Ingress) != 3 {
			t.Fatalf("expected 3 ingress rules, got %d", len(np.Spec.Ingress))
		}

		metricsRule := np.Spec.Ingress[0]
		if len(metricsRule.Ports) != 1 || *metricsRule.Ports[0].Port != intstr.FromInt32(8443) {
			t.Error("metrics ingress rule should allow port 8443")
		}
		if len(metricsRule.From) != 1 || metricsRule.From[0].NamespaceSelector == nil {
			t.Fatal("metrics ingress should restrict to openshift-monitoring namespace")
		}
		if metricsRule.From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "openshift-monitoring" {
			t.Error("metrics ingress should select openshift-monitoring namespace")
		}

		webhookRule := np.Spec.Ingress[1]
		if len(webhookRule.Ports) != 1 || *webhookRule.Ports[0].Port != intstr.FromInt32(9443) {
			t.Error("webhook ingress rule should allow port 9443")
		}
		if len(webhookRule.From) != 0 {
			t.Error("webhook ingress should allow from all sources")
		}

		healthRule := np.Spec.Ingress[2]
		if len(healthRule.Ports) != 1 || *healthRule.Ports[0].Port != intstr.FromInt32(8081) {
			t.Error("health ingress rule should allow port 8081")
		}
		if len(healthRule.From) != 0 {
			t.Error("health ingress should allow from all sources")
		}
	})

	t.Run("egress rules", func(t *testing.T) {
		if len(np.Spec.Egress) != 3 {
			t.Fatalf("expected 3 egress rules, got %d", len(np.Spec.Egress))
		}

		dnsRule := np.Spec.Egress[0]
		if len(dnsRule.Ports) != 2 {
			t.Fatalf("DNS egress should have 2 ports (UDP+TCP), got %d", len(dnsRule.Ports))
		}
		if *dnsRule.Ports[0].Protocol != corev1.ProtocolUDP || *dnsRule.Ports[0].Port != intstr.FromInt32(5353) {
			t.Error("DNS egress first port should be UDP/5353")
		}
		if *dnsRule.Ports[1].Protocol != corev1.ProtocolTCP || *dnsRule.Ports[1].Port != intstr.FromInt32(5353) {
			t.Error("DNS egress second port should be TCP/5353")
		}
		if len(dnsRule.To) != 1 || dnsRule.To[0].PodSelector == nil || dnsRule.To[0].NamespaceSelector == nil {
			t.Fatal("DNS egress should target specific pods in openshift-dns namespace")
		}
		if dnsRule.To[0].PodSelector.MatchLabels["dns.operator.openshift.io/daemonset-dns"] != "default" {
			t.Error("DNS egress should select dns-default pods")
		}
		if dnsRule.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "openshift-dns" {
			t.Error("DNS egress should select openshift-dns namespace")
		}

		apiRule := np.Spec.Egress[1]
		if len(apiRule.Ports) != 1 || *apiRule.Ports[0].Port != intstr.FromInt32(6443) {
			t.Error("API server egress should allow port 6443")
		}
		if len(apiRule.To) != 1 || apiRule.To[0].IPBlock == nil || apiRule.To[0].IPBlock.CIDR != "0.0.0.0/0" {
			t.Error("API server egress should use ipBlock 0.0.0.0/0")
		}

		registryRule := np.Spec.Egress[2]
		if len(registryRule.Ports) != 1 || *registryRule.Ports[0].Port != intstr.FromInt32(443) {
			t.Error("registry egress should allow port 443")
		}
		if len(registryRule.To) != 1 || registryRule.To[0].IPBlock == nil || registryRule.To[0].IPBlock.CIDR != "0.0.0.0/0" {
			t.Error("registry egress should use ipBlock 0.0.0.0/0")
		}
	})
}
