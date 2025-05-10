package enoexecevent

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/multiarch-tuning-operator/pkg/testing/framework"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"

	"github.com/openshift/multiarch-tuning-operator/pkg/testing/builder"
)

var _ = Describe("Controllers/ENoExecEvent/ENoExecEventReconciler", Serial, Ordered, func() {
	When("The ENoExecEvent", func() {
		Context("is handling the lifecycle of the operand", func() {
			It("should create a ENoExecEvent CR", func() {
				By("Creating the ENoExecEvent")
				enee := builder.NewENoExecEvent().WithName("test").WithNamespace(utils.Namespace()).Build()
				err := k8sClient.Create(ctx, enee)
				Expect(err).NotTo(HaveOccurred(), "failed to create ENoExecEvent", err)
				By("Deleting the ENoExecEvent")
				err = k8sClient.Delete(ctx, enee)
				Expect(err).NotTo(HaveOccurred(), "failed to delete ENoExecEvent", err)
				Eventually(framework.ValidateDeletion(k8sClient, ctx)).Should(Succeed(), "the ENoExecEvent should be deleted")
			})
		})
	})
})
