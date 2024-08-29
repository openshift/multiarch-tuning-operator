package podplacement

import (
	"context"
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	. "github.com/onsi/gomega"

	mmoimage "github.com/openshift/multiarch-tuning-operator/pkg/image"
	"github.com/openshift/multiarch-tuning-operator/pkg/testing/image/fake"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"

	. "github.com/openshift/multiarch-tuning-operator/pkg/testing/builder"
)

var ctx context.Context

func init() {
	ctx = context.TODO()
}

func TestPod_GetPodImagePullSecrets(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want []string
	}{
		{
			name: "pod with no imagePullSecrets",
			pod:  NewPod().Build(),
			want: []string{},
		},
		{
			name: "pod with imagePullSecrets",
			pod:  NewPod().WithImagePullSecrets("my-secret").Build(),
			want: []string{"my-secret"},
		},
		{
			name: "pod with empty imagePullSecrets",
			pod:  NewPod().WithImagePullSecrets().Build(),
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			g.Expect(pod.GetPodImagePullSecrets()).To(Equal(tt.want))
		})
	}
}

func TestPod_HasSchedulingGate(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want bool
	}{
		{
			name: "pod with no scheduling gates",
			pod:  NewPod().Build(),
			want: false,
		},
		{
			name: "pod with empty scheduling gates",
			pod:  NewPod().WithSchedulingGates().Build(),
			want: false,
		},
		{
			name: "pod with the multiarch-tuning-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates(utils.SchedulingGateName).Build(),
			want: true,
		},
		{
			name: "pod with scheduling gates and NO multiarch-tuning-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates("some-other-scheduling-gate").Build(),
			want: false,
		},
		{
			name: "pod with scheduling gates and the multiarch-tuning-operator scheduling gate",
			pod: NewPod().WithSchedulingGates(
				"some-other-scheduling-gate-bar", utils.SchedulingGateName, "some-other-scheduling-gate-foo").Build(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			g.Expect(pod.HasSchedulingGate()).To(Equal(tt.want))
		})
	}
}

func TestPod_RemoveSchedulingGate(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want []v1.PodSchedulingGate
	}{
		{
			name: "pod with no scheduling gates",
			pod:  NewPod().Build(),
			want: nil,
		},
		{
			name: "pod with empty scheduling gates",
			pod:  NewPod().WithSchedulingGates().Build(),
			want: []v1.PodSchedulingGate{},
		},
		{
			name: "pod with the multiarch-tuning-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates(utils.SchedulingGateName).Build(),
			want: []v1.PodSchedulingGate{},
		},
		{
			name: "pod with scheduling gates and NO multiarch-tuning-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates("some-other-scheduling-gate").Build(),
			want: []v1.PodSchedulingGate{
				{
					Name: "some-other-scheduling-gate",
				},
			},
		},
		{
			name: "pod with scheduling gates and the multiarch-tuning-operator scheduling gate",
			pod: NewPod().WithSchedulingGates(
				"some-other-scheduling-gate-bar", utils.SchedulingGateName,
				"some-other-scheduling-gate-foo").Build(),
			want: []v1.PodSchedulingGate{
				{
					Name: "some-other-scheduling-gate-bar",
				},
				{
					Name: "some-other-scheduling-gate-foo",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			pod.RemoveSchedulingGate()
			g := NewGomegaWithT(t)
			g.Expect(pod.Spec.SchedulingGates).To(Equal(tt.want))
		})
	}
}

func TestPod_imagesNamesSet(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want sets.Set[string]
	}{
		{
			name: "pod with a single container",
			pod: v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image: "bar/foo:latest",
						},
					},
				},
			},
			want: sets.New[string]("//bar/foo:latest"),
		},
		{
			name: "pod with multiple containers, some with the same image",
			pod:  NewPod().WithContainersImages("bar/foo:latest", "bar/baz:latest", "bar/foo:latest").Build(),
			want: sets.New[string]("//bar/foo:latest", "//bar/baz:latest"),
		},
		{
			name: "pod with multiple containers and init containers",
			pod: NewPod().WithInitContainersImages("foo/bar:latest").WithContainersImages(
				"bar/foo:latest", "bar/baz:latest", "bar/foo:latest").Build(),
			want: sets.New[string]("//bar/foo:latest", "//bar/baz:latest", "//foo/bar:latest"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			g.Expect(pod.imagesNamesSet()).To(Equal(tt.want))
		})
	}
}

func TestPod_intersectImagesArchitecture(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		// pullSecretDataList is a list of pull secrets in the form of a slice of bytes. It is not used in the unit
		// tests. It is used in the integration tests.
		pullSecretDataList         [][]byte
		wantSupportedArchitectures sets.Set[string]
		wantErr                    bool
	}{
		{
			name:                       "pod with a single container and multi-arch image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureAmd64, utils.ArchitectureArm64),
		},
		{
			name:                       "pod with a single container and single-arch image",
			pod:                        NewPod().WithContainersImages(fake.SingleArchArm64Image).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers and same image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, fake.MultiArchImage).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureAmd64, utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, single-arch image and multi-arch image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, fake.SingleArchArm64Image).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, two multi-arch images",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, fake.MultiArchImage2).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureAmd64, utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, one non-existing image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, "non-existing-image").Build(),
			wantErr:                    true,
			wantSupportedArchitectures: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			gotSupportedArchitectures, err := pod.intersectImagesArchitecture(tt.pullSecretDataList)
			g := NewGomegaWithT(t)
			g.Expect(err).Should(WithTransform(func(err error) bool { return err != nil }, Equal(tt.wantErr)),
				"error expectation failed")
			g.Expect(gotSupportedArchitectures).Should(WithTransform(func(arches []string) sets.Set[string] {
				if arches == nil {
					return nil
				}
				return sets.New[string](arches...)
			}, Equal(tt.wantSupportedArchitectures)),
				"the set in gotSupportedArchitectures is not equal to the expected one")
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

func TestPod_getArchitecturePredicate(t *testing.T) {
	tests := []struct {
		name               string
		pod                v1.Pod
		pullSecretDataList [][]byte
		// Be aware that the values in the want.Values slice must be sorted alphabetically
		want    v1.NodeSelectorRequirement
		wantErr bool
	}{
		{
			name: "pod with several containers using multi-arch images",
			pod: v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image: fake.MultiArchImage,
						},
					},
					InitContainers: []v1.Container{
						{
							Image: fake.MultiArchImage2,
						},
					},
				},
			},
			want: v1.NodeSelectorRequirement{
				Key:      utils.ArchLabel,
				Operator: v1.NodeSelectorOpIn,
				Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
			},
		},
		{
			name: "pod with non-existing image",
			pod: v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image: fake.MultiArchImage,
						},
						{
							Image: "non-existing-image",
						},
					},
				},
			},
			wantErr: true,
			want:    v1.NodeSelectorRequirement{},
		},
		{
			name: "pod with conflicting architectures",
			pod:  NewPod().WithContainersImages(fake.SingleArchAmd64Image, fake.SingleArchArm64Image).Build(),
			want: v1.NodeSelectorRequirement{
				Key:      utils.NoSupportedArchLabel,
				Operator: v1.NodeSelectorOpExists,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			got, err := pod.getArchitecturePredicate(tt.pullSecretDataList)
			g := NewGomegaWithT(t)
			g.Expect(err).Should(WithTransform(func(err error) bool { return err != nil }, Equal(tt.wantErr)),
				"error expectation failed")
			// sort the architectures to make the comparison easier
			sort.Strings(got.Values)
			g.Expect(got).To(Equal(tt.want))
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

func TestPod_setArchNodeAffinity(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want v1.Pod
	}{
		{
			name: "pod with empty node selector terms",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions().Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and nil match expressions",
			pod: NewPod().WithContainersImages(fake.SingleArchAmd64Image).WithNodeSelectorTermsMatchExpressions(
				nil).Build(),
			want: NewPod().WithContainersImages(fake.SingleArchAmd64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and empty match expressions",
			pod: NewPod().WithContainersImages(fake.SingleArchArm64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{}).Build(),
			want: NewPod().WithContainersImages(fake.SingleArchArm64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and match expressions",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
				}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and match expressions and an architecture requirement",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureS390x},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
				}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureS390x},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					}, {
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			pred, err := pod.getArchitecturePredicate(nil)
			g.Expect(err).ShouldNot(HaveOccurred())
			pod.setArchNodeAffinity(pred)
			g.Expect(pod.Spec.Affinity).Should(Equal(tt.want.Spec.Affinity))
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

func TestPod_SetNodeAffinityArchRequirement(t *testing.T) {
	tests := []struct {
		name               string
		pullSecretDataList [][]byte
		pod                v1.Pod
		want               v1.Pod
	}{
		{
			name: "pod with no node selector terms",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(nil).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector and no architecture requirement",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors("foo", "bar").Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors(
				"foo", "bar").WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with node selector and architecture requirement",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors("foo", "bar",
				utils.ArchLabel, utils.ArchitectureArm64).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors("foo", "bar",
				utils.ArchLabel, utils.ArchitectureArm64).Build(),
		},
		{
			name: "pod with no affinity",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with no node affinity",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with no required during scheduling ignored during execution",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithNodeAffinity().Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with predefined node selector terms in the required during scheduling ignored during execution",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions([]v1.NodeSelectorRequirement{
				{
					Key:      "foo",
					Operator: v1.NodeSelectorOpIn,
					Values:   []string{"bar"},
				},
			}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "other affinity types should not be modified",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(&v1.Affinity{
				PodAffinity: &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
						{
							TopologyKey: "foo",
						},
					},
				},
				NodeAffinity: &v1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
						{
							Weight: 1,
							Preference: v1.NodeSelectorTerm{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "foo",
										Operator: v1.NodeSelectorOpIn,
										Values:   []string{"bar"},
									},
								},
							},
						},
					},
				},
			}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(&v1.Affinity{
				PodAffinity: &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
						{
							TopologyKey: "foo",
						},
					},
				},
				NodeAffinity: &v1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
						{
							Weight: 1,
							Preference: v1.NodeSelectorTerm{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "foo",
										Operator: v1.NodeSelectorOpIn,
										Values:   []string{"bar"},
									},
								},
							},
						},
					},
				},
			}).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "should not modify the pod if unable to inspect the images",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage, "non-readable-image").Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage, "non-readable-image").Build(),
		},
		{
			name: "should prevent the pod from being scheduled when no common architecture is found",
			pod:  NewPod().WithContainersImages(fake.SingleArchAmd64Image, fake.SingleArchArm64Image).Build(),
			want: NewPod().WithContainersImages(fake.SingleArchAmd64Image, fake.SingleArchArm64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.NoSupportedArchLabel,
						Operator: v1.NodeSelectorOpExists,
					},
				}).Build(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			pod.SetNodeAffinityArchRequirement(tt.pullSecretDataList)
			g := NewGomegaWithT(t)
			g.Expect(pod.Spec.Affinity).Should(Equal(tt.want.Spec.Affinity))
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

// TestEnsureLabel checks the ensureLabel method to verify that it correctly sets labels.
func TestEnsureLabel(t *testing.T) {
	tests := []struct {
		name           string
		initialLabels  []string
		label          string
		value          string
		expectedLabels map[string]string
	}{
		{
			name:           "Empty Labels",
			initialLabels:  nil,
			label:          "testLabel",
			value:          "testValue",
			expectedLabels: map[string]string{"testLabel": "testValue"},
		},
		{
			name:           "Non-empty Labels",
			initialLabels:  []string{"existingLabel", "existingValue"},
			label:          "testLabel",
			value:          "testValue",
			expectedLabels: map[string]string{"existingLabel": "existingValue", "testLabel": "testValue"},
		},
		{
			name:           "Overwrite Existing Label",
			initialLabels:  []string{"testLabel", "oldValue"},
			label:          "testLabel",
			value:          "newValue",
			expectedLabels: map[string]string{"testLabel": "newValue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: NewPod().WithLabels(tt.initialLabels...).Build(),
			}

			pod.ensureLabel(tt.label, tt.value)

			if len(pod.Labels) != len(tt.expectedLabels) {
				t.Errorf("expected %d labels, got %d", len(tt.expectedLabels), len(pod.Labels))
			}

			for k, v := range tt.expectedLabels {
				if pod.Labels[k] != v {
					t.Errorf("expected label %s to have value %s, got %s", k, v, pod.Labels[k])
				}
			}
		})
	}
}

// TestEnsureArchitectureLabels checks the ensureArchitectureLabels method to ensure it sets the correct labels based on NodeSelectorRequirement.
func TestEnsureArchitectureLabels(t *testing.T) {
	tests := []struct {
		name           string
		requirement    v1.NodeSelectorRequirement
		expectedLabels map[string]string
	}{
		{
			name: "No Values",
			requirement: v1.NodeSelectorRequirement{
				Values: nil,
			},
			expectedLabels: map[string]string{},
		},
		{
			name: "Zero Values",
			requirement: v1.NodeSelectorRequirement{
				Values: []string{},
			},
			expectedLabels: map[string]string{
				utils.NoSupportedArchLabel: "",
			},
		},
		{
			name: "Single Value",
			requirement: v1.NodeSelectorRequirement{
				Values: []string{utils.ArchitectureAmd64},
			},
			expectedLabels: map[string]string{
				utils.SingleArchLabel:                         "",
				utils.ArchLabelValue(utils.ArchitectureAmd64): "",
			},
		},
		{
			name: "Multiple Values",
			requirement: v1.NodeSelectorRequirement{
				Values: []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
			},
			expectedLabels: map[string]string{
				utils.MultiArchLabel:                          "",
				utils.ArchLabelValue(utils.ArchitectureAmd64): "",
				utils.ArchLabelValue(utils.ArchitectureArm64): "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: NewPod().Build(),
			}

			pod.ensureArchitectureLabels(tt.requirement)

			if len(pod.Labels) != len(tt.expectedLabels) {
				t.Errorf("expected %d labels, got %d", len(tt.expectedLabels), len(pod.Labels))
			}

			for k, v := range tt.expectedLabels {
				if pod.Labels[k] != v {
					t.Errorf("expected label %s to have value %s, got %s", k, v, pod.Labels[k])
				}
			}
		})
	}
}
