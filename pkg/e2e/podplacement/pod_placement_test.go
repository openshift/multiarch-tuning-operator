package podplacement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/multiarch-tuning-operator/pkg/e2e"
	. "github.com/openshift/multiarch-tuning-operator/pkg/testing/builder"
	"github.com/openshift/multiarch-tuning-operator/pkg/testing/framework"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

const helloOpenshiftPublicMultiarchImage = "quay.io/openshifttest/hello-openshift:1.2.0"

var _ = Describe("The Pod Placement Operand", func() {
	Context("When a deployment is deployed with a single container and a public image", func() {
		It("should set the node affinity", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			ps := NewPodSpec().
				WithContainersImages(helloOpenshiftPublicMultiarchImage).
				Build()
			d := NewDeployment().
				WithSelectorAndPodLabels("app", "test").
				WithPodSpec(ps).
				WithReplicas(utils.NewPtr(int32(1))).
				WithName("test-deployment").
				WithNamespace(ns.Name).
				Build()
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			archLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.ArchLabel, corev1.NodeSelectorOpIn, utils.ArchitectureAmd64,
					utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le).
				Build()
			expectedNSTs := NewNodeSelectorTerm().WithMatchExpressions(&archLabelNSR).Build()
			verifyPodNodeAffinity(ns, "app", "test", expectedNSTs)
		})
		It("should set the node affinity on privileged deployments", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			ns.Labels = map[string]string{
				"pod-security.kubernetes.io/audit":               "privileged",
				"pod-security.kubernetes.io/warn":                "privileged",
				"pod-security.kubernetes.io/enforce":             "privileged",
				"security.openshift.io/scc.podSecurityLabelSync": "false",
			}
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			// Since the CRB to enable using SCCs is cluster-scoped, build its name based on the NS to
			// ensure concurrency safety with other possible test cases
			ephemeralCRBName := framework.NormalizeNameString("priv-scc-" + ns.Name)
			s := NewSubject().WithKind(rbacv1.ServiceAccountKind).WithName("default").WithNamespace(ns.Name).Build()
			crb := NewClusterRoleBinding().
				WithRoleRef(rbacv1.GroupName, "ClusterRole", "system:openshift:scc:privileged").
				WithSubjects(s).
				WithName(ephemeralCRBName).
				Build()
			err = client.Create(ctx, &crb)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, &crb)
			sc := NewSecurityContext().WithPrivileged(utils.NewPtr(true)).
				WithRunAsGroup(utils.NewPtr(int64(0))).
				WithRunAsUSer(utils.NewPtr(int64(0))).
				Build()
			vm := NewVolumeMount().WithName("test-hostpath").WithMountPath("/mnt/hostpath").Build()
			c := NewContainer().WithImage(helloOpenshiftPublicMultiarchImage).
				WithSecurityContext(&sc).
				WithVolumeMounts(vm).
				Build()
			v := NewVolume().WithName("test-hostpath").
				WithVolumeSourceHostPath("/var/lib/kubelet/config.json", utils.NewPtr(corev1.HostPathFile)).
				Build()
			ps := NewPodSpec().WithContainers(c).
				WithServiceAccountName("default").
				WithVolumes(v).
				Build()
			d := NewDeployment().
				WithSelectorAndPodLabels("app", "test").
				WithPodSpec(ps).
				WithReplicas(utils.NewPtr(int32(1))).
				WithName("test-deployment").
				WithNamespace(ns.Name).
				Build()
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			// TODO: verify the pod has some scc label
			verifyPodAnnotations(ns, "app", "test", "openshift.io/scc", "privileged")
			archLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.ArchLabel, corev1.NodeSelectorOpIn, utils.ArchitectureAmd64,
					utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le).
				Build()
			expectedNSTs := NewNodeSelectorTerm().WithMatchExpressions(&archLabelNSR).Build()
			verifyPodNodeAffinity(ns, "app", "test", expectedNSTs)
		})
		It("should set the node affinity when users node affinity do not conflict", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			hostnameLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.HostnameLabel, corev1.NodeSelectorOpExists).
				Build()
			hostnameLabelNSTs := NewNodeSelectorTerm().WithMatchExpressions(&hostnameLabelNSR).Build()
			ps := NewPodSpec().
				WithContainersImages(helloOpenshiftPublicMultiarchImage).
				WithNodeSelectorTerms(hostnameLabelNSTs).
				Build()
			d := NewDeployment().
				WithSelectorAndPodLabels("app", "test").
				WithPodSpec(ps).
				WithReplicas(utils.NewPtr(int32(1))).
				WithName("test-deployment").
				WithNamespace(ns.Name).
				Build()
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			archLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.ArchLabel, corev1.NodeSelectorOpIn, utils.ArchitectureAmd64,
					utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le).
				Build()
			expectedNSTs := NewNodeSelectorTerm().WithMatchExpressions(&hostnameLabelNSR, &archLabelNSR).Build()
			verifyPodNodeAffinity(ns, "app", "test", expectedNSTs)
		})
		It("should not set the node affinity when users node affinity conflicts", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			archLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.ArchLabel, corev1.NodeSelectorOpIn, utils.ArchitectureArm64).
				Build()
			archLabelNSTs := NewNodeSelectorTerm().WithMatchExpressions(&archLabelNSR).Build()
			ps := NewPodSpec().
				WithContainersImages(helloOpenshiftPublicMultiarchImage).
				WithNodeSelectorTerms(archLabelNSTs).Build()
			d := NewDeployment().
				WithSelectorAndPodLabels("app", "test").
				WithPodSpec(ps).
				WithReplicas(utils.NewPtr(int32(1))).
				WithName("test-deployment").
				WithNamespace(ns.Name).
				Build()
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			verifyPodNodeAffinity(ns, "app", "test", archLabelNSTs)
		})
		It("should check each matchExpressions when users node affinity has multiple matchExpressions", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			archLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.ArchLabel, corev1.NodeSelectorOpIn, utils.ArchitectureArm64).
				Build()
			hostnameLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.HostnameLabel, corev1.NodeSelectorOpExists).
				Build()
			archLabelNSTs := NewNodeSelectorTerm().WithMatchExpressions(&archLabelNSR).Build()
			hostnameLabelNSTs := NewNodeSelectorTerm().WithMatchExpressions(&hostnameLabelNSR).Build()
			ps := NewPodSpec().
				WithContainersImages(helloOpenshiftPublicMultiarchImage).
				WithNodeSelectorTerms(archLabelNSTs, hostnameLabelNSTs).Build()
			d := NewDeployment().
				WithSelectorAndPodLabels("app", "test").
				WithPodSpec(ps).
				WithReplicas(utils.NewPtr(int32(1))).
				WithName("test-deployment").
				WithNamespace(ns.Name).
				Build()
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			expectedArchLabelNSR := NewNodeSelectorRequirement().
				WithKeyAndValues(utils.ArchLabel, corev1.NodeSelectorOpIn, utils.ArchitectureAmd64,
					utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le).
				Build()
			expectedHostnameNST := NewNodeSelectorTerm().WithMatchExpressions(&hostnameLabelNSR, &expectedArchLabelNSR).Build()
			verifyPodNodeAffinity(ns, "app", "test", expectedHostnameNST, archLabelNSTs)
		})
		It("should set the node affinity when nodeSelector exist", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			ps := NewPodSpec().
				WithContainersImages(helloOpenshiftPublicMultiarchImage).
				WithNodeSelectors(utils.ArchLabel, utils.ArchitectureAmd64).
				Build()
			d := NewDeployment().
				WithSelectorAndPodLabels("app", "test").
				WithPodSpec(ps).
				WithReplicas(utils.NewPtr(int32(1))).
				WithName("test-deployment").
				WithNamespace(ns.Name).
				Build()
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			verifyPodNodeAffinity(ns, "app", "test")
		})
	})
})

func verifyPodNodeAffinity(ns *corev1.Namespace, labelKey string, labelInValue string, nodeSelectorTerms ...corev1.NodeSelectorTerm) {
	r, err := labels.NewRequirement(labelKey, "in", []string{labelInValue})
	labelSelector := labels.NewSelector().Add(*r)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func(g Gomega) {
		pods := &corev1.PodList{}
		err := client.List(ctx, pods, &runtimeclient.ListOptions{
			Namespace:     ns.Name,
			LabelSelector: labelSelector,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pods.Items).NotTo(BeEmpty())
		if len(nodeSelectorTerms) == 0 {
			g.Expect(pods.Items).To(HaveEach(WithTransform(func(p corev1.Pod) *corev1.Affinity {
				return p.Spec.Affinity
			}, BeNil())))
		} else {
			g.Expect(pods.Items).To(HaveEach(framework.HaveEquivalentNodeAffinity(
				&corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: nodeSelectorTerms,
					},
				})))
		}
	}, e2e.WaitShort).Should(Succeed())
}

func verifyPodAnnotations(ns *corev1.Namespace, labelKey string, labelInValue string, kv ...string) {
	if len(kv)%2 != 0 {
		panic("the number of arguments must be even")
	}
	r, err := labels.NewRequirement(labelKey, "in", []string{labelInValue})
	labelSelector := labels.NewSelector().Add(*r)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func(g Gomega) {
		pods := &corev1.PodList{}
		err := client.List(ctx, pods, &runtimeclient.ListOptions{
			Namespace:     ns.Name,
			LabelSelector: labelSelector,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pods.Items).NotTo(BeEmpty())
		for i := 0; i < len(pods.Items); i++ {
			pod := pods.Items[i]
			g.Expect(pod.Annotations).NotTo(BeEmpty())
			for j := 0; j < len(kv); j += 2 {
				g.Expect(pod.Annotations).To(HaveKeyWithValue(kv[j], kv[j+1]))
			}
		}
	}, e2e.WaitShort).Should(Succeed())
}
