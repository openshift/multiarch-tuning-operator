/*
Copyright 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift/multiarch-tuning-operator/pkg/testing/framework"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

func TestBuildNetworkPolicies(t *testing.T) {
	for name, build := range map[string]func() *networkingv1.NetworkPolicy{
		utils.PodPlacementNetworkPolicyName:                buildNetworkPolicyPodPlacement,
		utils.PodPlacementImageInspectionNetworkPolicyName: buildNetworkPolicyPodPlacementImageInspection,
		utils.EnoexecDaemonSet:                             buildNetworkPolicyENoExecDaemon,
		utils.ManagerNetworkPolicyName:                     buildNetworkPolicyManager,
	} {
		t.Run(name, func(t *testing.T) {
			np := build()
			g := gomega.NewWithT(t)
			g.Expect(np.Name).To(gomega.Equal(name))
			g.Expect(np.Namespace).To(gomega.Equal(utils.Namespace()))
			g.Expect(np.Spec).To(framework.MatchNetworkPolicySpec(name))
			invalid := np.DeepCopy()
			invalid.Spec.Ingress = append(invalid.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{})
			g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
			invalid = np.DeepCopy()
			invalid.Spec.PodSelector.MatchLabels["extra"] = "constraint"
			g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
			invalid = np.DeepCopy()
			invalid.Spec.Egress[0].Ports[0].EndPort = utils.NewPtr(int32(65535))
			g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
			invalid = np.DeepCopy()
			invalid.Spec.Ingress = append(invalid.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
				Ports: []networkingv1.NetworkPolicyPort{{Port: utils.NewPtr(intstr.FromInt32(1234))}},
			})
			g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
			if name == utils.PodPlacementImageInspectionNetworkPolicyName {
				invalid = np.DeepCopy()
				invalid.Spec.Egress[0].Ports = nil
				g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
			}
			for i, rule := range np.Spec.Ingress {
				if len(rule.From) == 0 {
					continue
				}
				invalid = np.DeepCopy()
				invalid.Spec.Ingress[i].From[0].PodSelector = &metav1.LabelSelector{}
				g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
				invalid = np.DeepCopy()
				invalid.Spec.Ingress[i].From[0].NamespaceSelector.MatchLabels["extra"] = "constraint"
				g.Expect(invalid.Spec).NotTo(framework.MatchNetworkPolicySpec(name))
			}
		})
	}
}

func TestDefaultKustomizationOmitsNetwork(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "config", "default", "kustomization.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "../network") {
		t.Fatal("default kustomization must omit runtime NetworkPolicies")
	}
}

func TestNetworkPolicyRBACIsNamespaceScoped(t *testing.T) {
	clusterRoleData, err := os.ReadFile(filepath.Join("..", "..", "..", "config", "rbac", "role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(clusterRoleData), "networkpolicies") {
		t.Fatal("ClusterRole must not grant networkpolicies")
	}
	npRoleData, err := os.ReadFile(filepath.Join("..", "..", "..", "config", "rbac", "networkpolicy_role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(npRoleData), "kind: Role") || !strings.Contains(string(npRoleData), "- networkpolicies") {
		t.Fatal("NetworkPolicy permissions must be in a namespaced Role")
	}
}
