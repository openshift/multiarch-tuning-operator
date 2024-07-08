package framework

import (
	"context"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func VerifyPodsAreRunning(g Gomega, ctx context.Context, client runtimeclient.Client, ns *v1.Namespace, labelKey string, labelInValue string) {
	r, err := labels.NewRequirement(labelKey, selection.In, []string{labelInValue})
	labelSelector := labels.NewSelector().Add(*r)
	g.Expect(err).NotTo(HaveOccurred())
	pods := &v1.PodList{}
	err = client.List(ctx, pods, &runtimeclient.ListOptions{
		Namespace:     ns.Name,
		LabelSelector: labelSelector,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(pods.Items).NotTo(BeEmpty())
	g.Expect(pods.Items).Should(HaveEach(WithTransform(func(p v1.Pod) v1.PodPhase {
		return p.Status.Phase
	}, Equal(v1.PodRunning))))
}
