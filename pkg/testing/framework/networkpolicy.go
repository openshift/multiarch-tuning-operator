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

package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

// Independent MTO-0006 specifications, never derived from production builders.
var expectedNetworkPolicySpecs = map[string]string{
	utils.PodPlacementNetworkPolicyName: `{
		"podSelector":{"matchLabels":{"multiarch.openshift.io/operand":"pod-placement-controller"}},
		"policyTypes":["Ingress","Egress"],
		"ingress":[
			{"ports":[{"protocol":"TCP","port":8443}],"from":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"openshift-monitoring"}}}]},
			{"ports":[{"protocol":"TCP","port":9443}]}
		],
		"egress":[
			{"ports":[{"protocol":"TCP","port":5353},{"protocol":"UDP","port":5353}],"to":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"openshift-dns"}}}]},
			{"ports":[{"protocol":"TCP","port":6443}]}
		]}`,
	utils.PodPlacementImageInspectionNetworkPolicyName: `{
		"podSelector":{"matchLabels":{"multiarch.openshift.io/operand":"pod-placement-controller","controller":"pod-placement-controller"}},
		"policyTypes":["Egress"],
		"egress":[{"ports":[{"protocol":"TCP"}]}]}`,
	utils.EnoexecDaemonSet: `{
		"podSelector":{"matchLabels":{"app":"enoexec-event-daemon"}},
		"policyTypes":["Egress"],
		"egress":[
			{"ports":[{"protocol":"TCP","port":5353},{"protocol":"UDP","port":5353}],"to":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"openshift-dns"}}}]},
			{"ports":[{"protocol":"TCP","port":6443}]}
		]}`,
	utils.ManagerNetworkPolicyName: `{
		"podSelector":{"matchLabels":{"control-plane":"controller-manager"}},
		"policyTypes":["Ingress","Egress"],
		"ingress":[
			{"ports":[{"protocol":"TCP","port":8443}],"from":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"openshift-monitoring"}}}]},
			{"ports":[{"protocol":"TCP","port":9443}]}
		],
		"egress":[
			{"ports":[{"protocol":"TCP","port":5353},{"protocol":"UDP","port":5353}],"to":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"openshift-dns"}}}]},
			{"ports":[{"protocol":"TCP","port":6443}]}
		]}`,
}

// MatchNetworkPolicySpec compares the complete security contract, ignoring only
// collection order, nil/empty collections, and API-defaulted TCP protocols.
func MatchNetworkPolicySpec(name string) types.GomegaMatcher {
	var expected networkingv1.NetworkPolicySpec
	if err := json.Unmarshal([]byte(expectedNetworkPolicySpecs[name]), &expected); err != nil {
		panic(fmt.Sprintf("invalid expected NetworkPolicy %q: %v", name, err))
	}
	return gomega.WithTransform(canonicalNetworkPolicySpec, gomega.Equal(canonicalNetworkPolicySpec(expected)))
}

func canonicalNetworkPolicySpec(spec networkingv1.NetworkPolicySpec) string {
	spec = *spec.DeepCopy()
	defaultProtocols := func(ports []networkingv1.NetworkPolicyPort) {
		for i := range ports {
			if ports[i].Protocol == nil {
				ports[i].Protocol = utils.NewPtr(corev1.ProtocolTCP)
			}
		}
	}
	for i := range spec.Ingress {
		defaultProtocols(spec.Ingress[i].Ports)
	}
	for i := range spec.Egress {
		defaultProtocols(spec.Egress[i].Ports)
	}
	// API JSON omitempty equates nil/empty collections, preserving nil versus
	// empty selector pointers. Arrays are unordered; retain multiplicity and
	// rule/peer boundaries when sorting them.
	data, err := json.Marshal(spec)
	if err != nil {
		panic(err)
	}
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		panic(err)
	}
	var normalize func(interface{})
	normalize = func(v interface{}) {
		switch v := v.(type) {
		case map[string]interface{}:
			for _, child := range v {
				normalize(child)
			}
		case []interface{}:
			for _, child := range v {
				normalize(child)
			}
			sort.Slice(v, func(i, j int) bool {
				a, _ := json.Marshal(v[i])
				b, _ := json.Marshal(v[j])
				return string(a) < string(b)
			})
		}
	}
	normalize(value)
	data, err = json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func verifyNetworkPolicies(ctx context.Context, c client.Client, names ...string) func(gomega.Gomega) {
	return func(g gomega.Gomega) {
		for _, name := range names {
			ginkgo.By("Verify the NetworkPolicy " + name)
			np := &networkingv1.NetworkPolicy{}
			g.Expect(c.Get(ctx, client.ObjectKey{Name: name, Namespace: utils.Namespace()}, np)).To(gomega.Succeed())
			g.Expect(np.Spec).To(MatchNetworkPolicySpec(name), "NetworkPolicy %s", name)
			if name == utils.ManagerNetworkPolicyName {
				for key := range np.Annotations {
					g.Expect(key).NotTo(gomega.HavePrefix("include.release.openshift.io/"))
				}
			}
		}
	}
}

// VerifyOperandNetworkPolicies verifies both complete operand policy contracts.
func VerifyOperandNetworkPolicies(ctx context.Context, c client.Client) func(gomega.Gomega) {
	return verifyNetworkPolicies(ctx, c, utils.PodPlacementNetworkPolicyName, utils.PodPlacementImageInspectionNetworkPolicyName)
}

// VerifyENoExecDaemonNetworkPolicy verifies the complete daemon policy contract.
func VerifyENoExecDaemonNetworkPolicy(ctx context.Context, c client.Client) func(gomega.Gomega) {
	return verifyNetworkPolicies(ctx, c, utils.EnoexecDaemonSet)
}

// VerifyManagerNetworkPolicy verifies the complete runtime manager policy contract.
func VerifyManagerNetworkPolicy(ctx context.Context, c client.Client) func(gomega.Gomega) {
	return verifyNetworkPolicies(ctx, c, utils.ManagerNetworkPolicyName)
}
