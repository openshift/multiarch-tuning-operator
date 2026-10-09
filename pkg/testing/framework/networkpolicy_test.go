package framework

import (
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

func TestNetworkPolicyAssertions(t *testing.T) {
	type assertionCase struct {
		name   string
		mutate func(*networkingv1.NetworkPolicySpec)
		valid  bool
	}
	for name, fixture := range expectedNetworkPolicySpecs {
		t.Run(name, func(t *testing.T) {
			var expected networkingv1.NetworkPolicySpec
			if err := json.Unmarshal([]byte(fixture), &expected); err != nil {
				t.Fatal(err)
			}
			tests := []assertionCase{
				{"expected", func(*networkingv1.NetworkPolicySpec) {}, true},
				{"allow all ingress", func(s *networkingv1.NetworkPolicySpec) {
					s.Ingress = append(s.Ingress, networkingv1.NetworkPolicyIngressRule{})
				}, false},
				{"extra unrestricted port", func(s *networkingv1.NetworkPolicySpec) {
					p := intstr.FromInt32(1234)
					s.Ingress = append(s.Ingress, networkingv1.NetworkPolicyIngressRule{Ports: []networkingv1.NetworkPolicyPort{{Port: &p}}})
				}, false},
				{"workload labels", func(s *networkingv1.NetworkPolicySpec) { s.PodSelector.MatchLabels["extra"] = "constraint" }, false},
				{"workload expressions", func(s *networkingv1.NetworkPolicySpec) {
					s.PodSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{Key: "extra", Operator: metav1.LabelSelectorOpExists}}
				}, false},
				{"end port", func(s *networkingv1.NetworkPolicySpec) { s.Egress[0].Ports[0].EndPort = utils.NewPtr(int32(65535)) }, false},
				{"wrong protocol", func(s *networkingv1.NetworkPolicySpec) {
					s.Egress[0].Ports[0].Protocol = utils.NewPtr(corev1.ProtocolSCTP)
				}, false},
				{"wrong port", func(s *networkingv1.NetworkPolicySpec) {
					s.Egress[0].Ports[0].Port = utils.NewPtr(intstr.FromInt32(1234))
				}, false},
				{"extra egress", func(s *networkingv1.NetworkPolicySpec) {
					s.Egress = append(s.Egress, networkingv1.NetworkPolicyEgressRule{})
				}, false},
				{"policy type", func(s *networkingv1.NetworkPolicySpec) {
					s.PolicyTypes = append(s.PolicyTypes, networkingv1.PolicyTypeIngress)
				}, false},
				{"reordered", func(s *networkingv1.NetworkPolicySpec) {
					for i, j := 0, len(s.Ingress)-1; i < j; i, j = i+1, j-1 {
						s.Ingress[i], s.Ingress[j] = s.Ingress[j], s.Ingress[i]
					}
					for i, j := 0, len(s.Egress)-1; i < j; i, j = i+1, j-1 {
						s.Egress[i], s.Egress[j] = s.Egress[j], s.Egress[i]
					}
					for i, j := 0, len(s.PolicyTypes)-1; i < j; i, j = i+1, j-1 {
						s.PolicyTypes[i], s.PolicyTypes[j] = s.PolicyTypes[j], s.PolicyTypes[i]
					}
					for i := range s.Egress {
						p := s.Egress[i].Ports
						for a, b := 0, len(p)-1; a < b; a, b = a+1, b-1 {
							p[a], p[b] = p[b], p[a]
						}
					}
				}, true},
				{"defaulted TCP and empty collections", func(s *networkingv1.NetworkPolicySpec) {
					s.PodSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
					if len(s.Ingress) == 0 {
						s.Ingress = []networkingv1.NetworkPolicyIngressRule{}
					}
					for i := range s.Ingress {
						for j := range s.Ingress[i].Ports {
							s.Ingress[i].Ports[j].Protocol = nil
						}
						if len(s.Ingress[i].From) == 0 {
							s.Ingress[i].From = []networkingv1.NetworkPolicyPeer{}
						}
					}
					for i := range s.Egress {
						for j := range s.Egress[i].Ports {
							if *s.Egress[i].Ports[j].Protocol == corev1.ProtocolTCP {
								s.Egress[i].Ports[j].Protocol = nil
							}
						}
						if len(s.Egress[i].To) == 0 {
							s.Egress[i].To = []networkingv1.NetworkPolicyPeer{}
						}
					}
				}, true},
			}
			if name == utils.PodPlacementImageInspectionNetworkPolicyName {
				tests = append(tests, assertionCase{"all protocols", func(s *networkingv1.NetworkPolicySpec) { s.Egress[0].Ports = nil }, false})
			} else {
				for _, mutation := range []struct {
					name   string
					mutate func(*networkingv1.NetworkPolicyPeer)
				}{
					{"pod selector", func(p *networkingv1.NetworkPolicyPeer) { p.PodSelector = &metav1.LabelSelector{} }},
					{"namespace labels", func(p *networkingv1.NetworkPolicyPeer) { p.NamespaceSelector.MatchLabels["extra"] = "constraint" }},
					{"namespace expressions", func(p *networkingv1.NetworkPolicyPeer) {
						p.NamespaceSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{Key: "extra", Operator: metav1.LabelSelectorOpExists}}
					}},
					{"IPBlock", func(p *networkingv1.NetworkPolicyPeer) { p.IPBlock = &networkingv1.IPBlock{CIDR: "0.0.0.0/0"} }},
				} {
					tests = append(tests, assertionCase{"DNS " + mutation.name, func(s *networkingv1.NetworkPolicySpec) { mutation.mutate(&s.Egress[0].To[0]) }, false})
					if len(expected.Ingress) > 0 {
						tests = append(tests, assertionCase{"metrics " + mutation.name, func(s *networkingv1.NetworkPolicySpec) { mutation.mutate(&s.Ingress[0].From[0]) }, false})
					}
				}
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					actual := expected.DeepCopy()
					tt.mutate(actual)
					matched, err := MatchNetworkPolicySpec(name).Match(*actual)
					if err != nil || matched != tt.valid {
						t.Fatalf("match=%v error=%v, want %v", matched, err, tt.valid)
					}
				})
			}
		})
	}
}
