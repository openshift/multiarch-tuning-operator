package framework

import (
	"context"
	"log"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	ocpconfigv1 "github.com/openshift/api/config/v1"
)

func VerifyCOAreUpdating(g Gomega, ctx context.Context, client runtimeclient.Client, operatorName string) {
	var err error
	co := ocpconfigv1.ClusterOperator{}
	err = client.Get(ctx, runtimeclient.ObjectKeyFromObject(&ocpconfigv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: operatorName,
		},
	}), &co)
	g.Expect(err).NotTo(HaveOccurred())
	progressingStatus := ocpconfigv1.ConditionFalse
	for _, condition := range co.Status.Conditions {
		if condition.Type == "Progressing" {
			progressingStatus = condition.Status
			break
		}
	}
	g.Expect(progressingStatus).To(Equal(ocpconfigv1.ConditionTrue))
}

func VerifyCOAreUpdated(g Gomega, ctx context.Context, client runtimeclient.Client, operatorName string) {
	var err error
	co := ocpconfigv1.ClusterOperator{}
	err = client.Get(ctx, runtimeclient.ObjectKeyFromObject(&ocpconfigv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: operatorName,
		},
	}), &co)
	g.Expect(err).NotTo(HaveOccurred())
	progressingStatus := ocpconfigv1.ConditionTrue
	for _, condition := range co.Status.Conditions {
		if condition.Type == "Progressing" {
			progressingStatus = condition.Status
			break
		}
	}
	g.Expect(progressingStatus).To(Equal(ocpconfigv1.ConditionFalse))
}

func WaitForCOComplete(ctx context.Context, client runtimeclient.Client, operatorName string) {
	log.Printf("Verifying ClusterOperator %s start updating", operatorName)
	Eventually(func(g Gomega) {
		VerifyCOAreUpdating(g, ctx, client, operatorName)
	}, 3*time.Minute, 1*time.Minute).Should(Succeed())
	log.Printf("Verifying ClusterOperator %s finish updating", operatorName)
	Eventually(func(g Gomega) {
		VerifyCOAreUpdated(g, ctx, client, operatorName)
	}, 5*time.Minute, 1*time.Minute).Should(Succeed())
}
