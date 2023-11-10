package podplacement

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	mmoimage "multiarch-operator/pkg/image"
	"multiarch-operator/pkg/image/fake"
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	. "github.com/onsi/gomega"
)

var ctx context.Context

func init() {
	ctx = context.TODO()
}

// PodFactory is a builder for v1.Pod objects to be used only in unit tests.
type PodFactory struct {
	pod v1.Pod
}

// newPod returns a new PodFactory to build v1.Pod objects. It is meant to be used only in unit tests.
func newPod() *PodFactory {
	return &PodFactory{
		pod: v1.Pod{},
	}
}

func (p *PodFactory) withImagePullSecrets(imagePullSecrets ...string) *PodFactory {
	p.pod.Spec.ImagePullSecrets = make([]v1.LocalObjectReference, len(imagePullSecrets))
	for i, secret := range imagePullSecrets {
		p.pod.Spec.ImagePullSecrets[i] = v1.LocalObjectReference{
			Name: secret,
		}
	}
	return p
}

func (p *PodFactory) withSchedulingGates(schedulingGates ...string) *PodFactory {
	p.pod.Spec.SchedulingGates = make([]v1.PodSchedulingGate, len(schedulingGates))
	for i, gate := range schedulingGates {
		p.pod.Spec.SchedulingGates[i] = v1.PodSchedulingGate{
			Name: gate,
		}
	}
	return p
}

func (p *PodFactory) withContainersImages(images ...string) *PodFactory {
	p.pod.Spec.Containers = make([]v1.Container, len(images))
	for i, image := range images {
		// compute hash of the image name
		sha := sha1.New()
		sha.Write([]byte(image))
		p.pod.Spec.Containers[i] = v1.Container{
			Image: image,
			Name:  hex.EncodeToString(sha.Sum(nil)), // hash of the image name (40 characters, 63 is max)
		}
	}
	return p
}

func (p *PodFactory) withInitContainersImages(images ...string) *PodFactory {
	p.pod.Spec.InitContainers = make([]v1.Container, len(images))
	for i, image := range images {
		p.pod.Spec.InitContainers[i] = v1.Container{
			Image: image,
		}
	}
	return p
}

// withNodeAffinity adds a node affinity to the pod. If initialAffinity is not nil, it is used as the initial value
// of the pod's affinity. Otherwise, the pod's affinity is initialized to an empty affinity if it is nil.
func (p *PodFactory) withAffinity(initialAffinity *v1.Affinity) *PodFactory {
	if p.pod.Spec.Affinity == nil {
		p.pod.Spec.Affinity = &v1.Affinity{}
	}
	if initialAffinity != nil {
		p.pod.Spec.Affinity = initialAffinity
	}
	return p
}

func (p *PodFactory) withNodeAffinity() *PodFactory {
	p.withAffinity(nil)
	if p.pod.Spec.Affinity.NodeAffinity == nil {
		p.pod.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{}
	}
	return p
}

func (p *PodFactory) withRequiredDuringSchedulingIgnoredDuringExecution() *PodFactory {
	p.withNodeAffinity()
	if p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{}
	}
	return p
}

func (p *PodFactory) withNodeSelectorTermsMatchExpressions(
	nodeSelectorTermsMatchExpressions ...[]v1.NodeSelectorRequirement) *PodFactory {
	p.withRequiredDuringSchedulingIgnoredDuringExecution()
	p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = make(
		[]v1.NodeSelectorTerm, len(nodeSelectorTermsMatchExpressions))
	for i, matchExpressions := range nodeSelectorTermsMatchExpressions {
		p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i] =
			v1.NodeSelectorTerm{
				MatchExpressions: matchExpressions,
			}
	}
	return p
}

func (p *PodFactory) withNodeSelectors(kv ...string) *PodFactory {
	if p.pod.Spec.NodeSelector == nil {
		p.pod.Spec.NodeSelector = make(map[string]string)
	}
	if len(kv)%2 != 0 {
		panic("the number of arguments must be even")
	}
	for i := 0; i < len(kv); i += 2 {
		p.pod.Spec.NodeSelector[kv[i]] = kv[i+1]
	}
	return p
}

func (p *PodFactory) withGenerateName(name string) *PodFactory {
	p.pod.GenerateName = name
	return p
}

func (p *PodFactory) withNamespace(namespace string) *PodFactory {
	p.pod.Namespace = namespace
	return p
}

func (p *PodFactory) build() v1.Pod {
	return p.pod
}

func TestPod_GetPodImagePullSecrets(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want []string
	}{
		{
			name: "pod with no imagePullSecrets",
			pod:  newPod().build(),
			want: []string{},
		},
		{
			name: "pod with imagePullSecrets",
			pod:  newPod().withImagePullSecrets("my-secret").build(),
			want: []string{"my-secret"},
		},
		{
			name: "pod with empty imagePullSecrets",
			pod:  newPod().withImagePullSecrets().build(),
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
			pod:  newPod().build(),
			want: false,
		},
		{
			name: "pod with empty scheduling gates",
			pod:  newPod().withSchedulingGates().build(),
			want: false,
		},
		{
			name: "pod with the multiarch-manager-operator scheduling gate",
			pod:  newPod().withSchedulingGates(schedulingGateName).build(),
			want: true,
		},
		{
			name: "pod with scheduling gates and NO multiarch-manager-operator scheduling gate",
			pod:  newPod().withSchedulingGates("some-other-scheduling-gate").build(),
			want: false,
		},
		{
			name: "pod with scheduling gates and the multiarch-manager-operator scheduling gate",
			pod: newPod().withSchedulingGates(
				"some-other-scheduling-gate-bar", schedulingGateName, "some-other-scheduling-gate-foo").build(),
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
			pod:  newPod().build(),
			want: nil,
		},
		{
			name: "pod with empty scheduling gates",
			pod:  newPod().withSchedulingGates().build(),
			want: []v1.PodSchedulingGate{},
		},
		{
			name: "pod with the multiarch-manager-operator scheduling gate",
			pod:  newPod().withSchedulingGates(schedulingGateName).build(),
			want: []v1.PodSchedulingGate{},
		},
		{
			name: "pod with scheduling gates and NO multiarch-manager-operator scheduling gate",
			pod:  newPod().withSchedulingGates("some-other-scheduling-gate").build(),
			want: []v1.PodSchedulingGate{
				{
					Name: "some-other-scheduling-gate",
				},
			},
		},
		{
			name: "pod with scheduling gates and the multiarch-manager-operator scheduling gate",
			pod: newPod().withSchedulingGates(
				"some-other-scheduling-gate-bar", schedulingGateName,
				"some-other-scheduling-gate-foo").build(),
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
			pod:  newPod().withContainersImages("bar/foo:latest", "bar/baz:latest", "bar/foo:latest").build(),
			want: sets.New[string]("//bar/foo:latest", "//bar/baz:latest"),
		},
		{
			name: "pod with multiple containers and init containers",
			pod: newPod().withInitContainersImages("foo/bar:latest").withContainersImages(
				"bar/foo:latest", "bar/baz:latest", "bar/foo:latest").build(),
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
			pod:                        newPod().withContainersImages(fake.MultiArchImage).build(),
			wantSupportedArchitectures: sets.New[string](fake.ArchitectureAmd64, fake.ArchitectureArm64),
		},
		{
			name:                       "pod with a single container and single-arch image",
			pod:                        newPod().withContainersImages(fake.SingleArchArm64Image).build(),
			wantSupportedArchitectures: sets.New[string](fake.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers and same image",
			pod:                        newPod().withContainersImages(fake.MultiArchImage, fake.MultiArchImage).build(),
			wantSupportedArchitectures: sets.New[string](fake.ArchitectureAmd64, fake.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, single-arch image and multi-arch image",
			pod:                        newPod().withContainersImages(fake.MultiArchImage, fake.SingleArchArm64Image).build(),
			wantSupportedArchitectures: sets.New[string](fake.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, two multi-arch images",
			pod:                        newPod().withContainersImages(fake.MultiArchImage, fake.MultiArchImage2).build(),
			wantSupportedArchitectures: sets.New[string](fake.ArchitectureAmd64, fake.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, one non-existing image",
			pod:                        newPod().withContainersImages(fake.MultiArchImage, "non-existing-image").build(),
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
				Key:      archLabel,
				Operator: v1.NodeSelectorOpIn,
				Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
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
			pod:  newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions().build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				},
			).build(),
		},
		{
			name: "pod with node selector terms and nil match expressions",
			pod: newPod().withContainersImages(fake.SingleArchAmd64Image).withNodeSelectorTermsMatchExpressions(
				nil).build(),
			want: newPod().withContainersImages(fake.SingleArchAmd64Image).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64},
					},
				},
			).build(),
		},
		{
			name: "pod with node selector terms and empty match expressions",
			pod: newPod().withContainersImages(fake.SingleArchArm64Image).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{}).build(),
			want: newPod().withContainersImages(fake.SingleArchArm64Image).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureArm64},
					},
				},
			).build(),
		},
		{
			name: "pod with node selector terms and match expressions",
			pod: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
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
				}).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				},
			).build(),
		},
		{
			name: "pod with node selector terms and match expressions and an architecture requirement",
			pod: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureS390x},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
				}).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureS390x},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					}, {
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
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
			pod:  newPod().withContainersImages(fake.MultiArchImage).withAffinity(nil).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				},
			).build(),
		},
		{
			name: "pod with node selector and no architecture requirement",
			pod:  newPod().withContainersImages(fake.MultiArchImage).withNodeSelectors("foo", "bar").build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectors(
				"foo", "bar").withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
		},
		{
			name: "pod with node selector and architecture requirement",
			pod: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectors("foo", "bar",
				archLabel, fake.ArchitectureArm64).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectors("foo", "bar",
				archLabel, fake.ArchitectureArm64).build(),
		},
		{
			name: "pod with no affinity",
			pod:  newPod().withContainersImages(fake.MultiArchImage).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
		},
		{
			name: "pod with no node affinity",
			pod:  newPod().withContainersImages(fake.MultiArchImage).withAffinity(nil).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
		},
		{
			name: "pod with no required during scheduling ignored during execution",
			pod:  newPod().withContainersImages(fake.MultiArchImage).withNodeAffinity().build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
		},
		{
			name: "pod with predefined node selector terms in the required during scheduling ignored during execution",
			pod: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions([]v1.NodeSelectorRequirement{
				{
					Key:      "foo",
					Operator: v1.NodeSelectorOpIn,
					Values:   []string{"bar"},
				},
			}).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
		},
		{
			name: "other affinity types should not be modified",
			pod: newPod().withContainersImages(fake.MultiArchImage).withAffinity(&v1.Affinity{
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
			}).build(),
			want: newPod().withContainersImages(fake.MultiArchImage).withAffinity(&v1.Affinity{
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
			}).withNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      archLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{fake.ArchitectureAmd64, fake.ArchitectureArm64},
					},
				}).build(),
		},
		{
			name: "should not modify the pod if unable to inspect the images",
			pod:  newPod().withContainersImages(fake.MultiArchImage, "non-readable-image").build(),
			want: newPod().withContainersImages(fake.MultiArchImage, "non-readable-image").build(),
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
