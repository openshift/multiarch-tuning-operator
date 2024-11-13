package podplacement

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/common/plugins"
	v1alpha1 "github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1alpha1"
	v1beta1 "github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ClusterPodPlacementConfig Conversion Tests", func() {
	var (
		ctx context.Context
	)
	BeforeEach(func() {
		ctx = context.TODO()
		err := v1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
		err = v1beta1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		err := k8sClient.Delete(ctx, &v1beta1.ClusterPodPlacementConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cppc",
				Namespace: "default",
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})
	Context("When a v1beta1 pod placement config is created", func() {
		It("should create a v1beta1 CR omitting the plugins key and successfully convert it to v1alpha1", func() {
			By("Creating a v1beta1 ClusterPodPlacementConfig")
			err := k8sClient.Create(ctx, &v1beta1.ClusterPodPlacementConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cppc",
					Namespace: "default",
				},
				Spec: v1beta1.ClusterPodPlacementConfigSpec{
					LogVerbosity: "Normal",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				v1alpha1Obj := &v1alpha1.ClusterPodPlacementConfig{}
				err := k8sClient.Get(ctx, runtimeclient.ObjectKey{Name: "test-cppc", Namespace: "default"}, v1alpha1Obj)
				g.Expect(err).NotTo(HaveOccurred())
				// Verify the LogVerbosity field
				g.Expect(v1alpha1Obj.Spec.LogVerbosity).NotTo(Equal("Normal"))
			}, time.Second*10, time.Millisecond*250).Should(Succeed())
		})
	})
	Context("When a v1beta1 ClusterPodPlacementConfig with NodeAffinityScoringPlugin is created", func() {
		It("should convert to v1beta1 and get rid of the NodeAffinityScoringPlugin configuration", func() {
			// Step 1: Create a v1alpha1 ClusterPodPlacementConfig object
			err := k8sClient.Create(ctx, &v1beta1.ClusterPodPlacementConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cppc",
					Namespace: "default",
				},
				Spec: v1beta1.ClusterPodPlacementConfigSpec{
					LogVerbosity: "Normal",
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"env": "test"},
					},
					Plugins: plugins.Plugins{
						NodeAffinityScoring: &plugins.NodeAffinityScoring{
							BasePlugin: plugins.BasePlugin{
								Enabled: true,
							},
							Platforms: []plugins.NodeAffinityScoringPlatformTerm{
								{Architecture: "ppc64le", Weight: 50},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Step 2: Validate the conversion to v1beta1
			Eventually(func(g Gomega) {
				v1alpha1Obj := &v1alpha1.ClusterPodPlacementConfig{}
				err := k8sClient.Get(ctx, runtimeclient.ObjectKey{Name: "test-cppc", Namespace: "default"}, v1alpha1Obj)
				g.Expect(err).NotTo(HaveOccurred())

				// Verify the LogVerbosity field
				g.Expect(v1alpha1Obj.Spec.LogVerbosity).NotTo(Equal("Normal"))

				// Verify the NamespaceSelector
				g.Expect(v1alpha1Obj.Spec.NamespaceSelector.MatchLabels).To(Equal(map[string]string{"env": "test"}))

				// Verify the Plugins field
				g.Expect(v1alpha1Obj.Spec.Plugins.NodeAffinityScoring).NotTo(BeNil())
				g.Expect(v1alpha1Obj.Spec.Plugins.NodeAffinityScoring.Enabled).To(BeTrue())
				g.Expect(v1alpha1Obj.Spec.Plugins.NodeAffinityScoring.Platforms).To(ConsistOf(
					plugins.NodeAffinityScoringPlatformTerm{Architecture: "ppc64le", Weight: 50},
				))
			}, time.Second*10, time.Millisecond*250).Should(Succeed())
		})
	})
})

var _ = ginkgo.Describe("NodeAffinityScoring Validation", func() {
	tests := []struct {
		name       string
		platforms  []plugins.NodeAffinityScoringPlatformTerm
		shouldFail bool
	}{
		{
			name: "Valid Platforms",
			platforms: []plugins.NodeAffinityScoringPlatformTerm{
				{Architecture: "ppc64le", Weight: 50},
				{Architecture: "amd64", Weight: 30},
			},
			shouldFail: false,
		},
		{
			name: "Invalid Architecture",
			platforms: []plugins.NodeAffinityScoringPlatformTerm{
				{Architecture: "invalid_arch", Weight: 20},
			},
			shouldFail: true,
		},
		{
			name: "Invalid Weight",
			platforms: []plugins.NodeAffinityScoringPlatformTerm{
				{Architecture: "amd64", Weight: -10},
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		testCase := tt
		ginkgo.It(testCase.name, func() {
			scoring := &plugins.NodeAffinityScoring{
				BasePlugin: plugins.BasePlugin{Enabled: true},
				Platforms:  testCase.platforms,
			}

			err := validateNodeAffinityScoring(scoring)
			if testCase.shouldFail {
				gomega.Expect(err).To(gomega.HaveOccurred(), "Expected validation to fail but it passed")
			} else {
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Expected validation to pass but it failed")
			}
		})
	}
})

func validateNodeAffinityScoring(scoring *plugins.NodeAffinityScoring) error {
	for _, platform := range scoring.Platforms {
		// Check if Architecture is non-empty.
		if len(platform.Architecture) == 0 {
			return fmt.Errorf("Architecture cannot be empty")
		}
		// Check if Weight is within range.
		if platform.Weight < 0 || platform.Weight > 100 {
			return fmt.Errorf("Weight must be between 0 and 100")
		}
		// Validate architecture value (simulate Enum validation).
		validArchitectures := map[string]bool{
			"arm64":   true,
			"amd64":   true,
			"ppc64le": true,
			"s390x":   true,
		}
		if !validArchitectures[platform.Architecture] {
			return fmt.Errorf("Invalid architecture: %s", platform.Architecture)
		}
	}
	return nil
}
