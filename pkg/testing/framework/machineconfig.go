package framework

import (
	"context"
	"log"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	ocpmachineconfigurationv1 "github.com/openshift/api/machineconfiguration/v1"
)

func VerifyMCPsAreUpdating(g Gomega, ctx context.Context, client runtimeclient.Client) {
	var err error
	mcps := ocpmachineconfigurationv1.MachineConfigPoolList{}
	err = client.List(ctx, &mcps)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mcps.Items).NotTo(BeEmpty())
	g.Expect(mcps.Items).Should(HaveEach(WithTransform(func(mcp ocpmachineconfigurationv1.MachineConfigPool) corev1.ConditionStatus {
		status := corev1.ConditionFalse
		for _, condition := range mcp.Status.Conditions {
			if condition.Type == "Updating" {
				status = condition.Status
				break
			}
		}
		return status
	}, Equal(corev1.ConditionTrue))))
}

func VerifyMCPAreUpdated(ctx context.Context, client runtimeclient.Client, poolName string) {
	var err error
	mcp := ocpmachineconfigurationv1.MachineConfigPool{}
	err = client.Get(ctx, runtimeclient.ObjectKeyFromObject(&ocpmachineconfigurationv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: poolName,
		},
	}), &mcp)
	Expect(err).NotTo(HaveOccurred())
	machineCount := mcp.Status.MachineCount
	timeToWait := time.Duration(10*machineCount) * time.Minute
	log.Printf("Waiting %s for MCP %s to be completed.", timeToWait, poolName)
	Eventually(func(g Gomega) {
		mcp := ocpmachineconfigurationv1.MachineConfigPool{}
		err = client.Get(ctx, runtimeclient.ObjectKeyFromObject(&ocpmachineconfigurationv1.MachineConfigPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: poolName,
			},
		}), &mcp)
		g.Expect(err).NotTo(HaveOccurred())
		status := corev1.ConditionFalse
		for _, condition := range mcp.Status.Conditions {
			if condition.Type == "Updated" {
				status = condition.Status
				break
			}
		}
		g.Expect(status).Should(Equal(corev1.ConditionTrue))
	}, timeToWait, 1*time.Minute).Should(Succeed())
}

func WaitForMCPComplete(ctx context.Context, client runtimeclient.Client) {
	log.Printf("Verifying machineconfig start updating")
	Eventually(func(g Gomega) {
		VerifyMCPsAreUpdating(g, ctx, client)
	}, 3*time.Minute, 1*time.Minute).Should(Succeed())
	log.Printf("Verifying machineconfig finish updating")
	VerifyMCPAreUpdated(ctx, client, "worker")
	VerifyMCPAreUpdated(ctx, client, "master")
}
