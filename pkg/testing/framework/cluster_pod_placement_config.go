package framework

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/openshift/library-go/pkg/operator/v1helpers"

	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/common"
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
	"github.com/openshift/multiarch-tuning-operator/pkg/testing/builder"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

type ConditionTypeStatusTuple struct {
	ConditionType   string
	ConditionStatus corev1.ConditionStatus
}

func NewConditionTypeStatusTuple(conditionType string, conditionStatus corev1.ConditionStatus) ConditionTypeStatusTuple {
	return ConditionTypeStatusTuple{
		ConditionType:   conditionType,
		ConditionStatus: conditionStatus,
	}
}

func VerifyConditions(ctx context.Context, c client.Client, conditionTypeStatusTuples ...ConditionTypeStatusTuple) func(g gomega.Gomega) {
	return func(g gomega.Gomega) {
		ppc := &v1beta1.ClusterPodPlacementConfig{}
		err := c.Get(ctx, client.ObjectKey{
			Name: common.SingletonResourceObjectName,
		}, ppc)
		g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get ClusterPodPlacementConfig", err)
		for _, condStatusPairs := range conditionTypeStatusTuples {
			conditionType := condStatusPairs.ConditionType
			conditionStatus := condStatusPairs.ConditionStatus
			g.Expect(v1helpers.FindCondition(ppc.Status.Conditions, conditionType)).NotTo(gomega.BeNil(), "the condition "+conditionType+" should be set")
			g.Expect(v1helpers.FindCondition(ppc.Status.Conditions, conditionType).Status).To(gomega.BeEquivalentTo(conditionStatus), "the condition"+conditionType+" should be "+string(conditionStatus))
		}
	}
}

func getObjects() []client.Object {
	return []client.Object{
		builder.NewDeployment().WithName(utils.PodPlacementControllerName).WithNamespace(utils.Namespace()).Build(),
		builder.NewDeployment().WithName(utils.PodPlacementWebhookName).WithNamespace(utils.Namespace()).Build(),
		builder.NewService().WithName(utils.PodPlacementWebhookName).WithNamespace(utils.Namespace()).Build(),
		builder.NewService().WithName(utils.PodPlacementControllerName).WithNamespace(utils.Namespace()).Build(),
		builder.NewMutatingWebhookConfiguration().WithName(utils.PodMutatingWebhookConfigurationName).Build(),
		builder.NewClusterRole().WithName(utils.PodPlacementControllerName).Build(),
		builder.NewClusterRole().WithName(utils.PodPlacementWebhookName).Build(),
		builder.NewClusterRoleBinding().WithName(utils.PodPlacementControllerName).Build(),
		builder.NewClusterRoleBinding().WithName(utils.PodPlacementWebhookName).Build(),
		builder.NewRole().WithName(utils.PodPlacementControllerName).WithNamespace(utils.Namespace()).Build(),
		builder.NewRoleBinding().WithName(utils.PodPlacementControllerName).WithNamespace(utils.Namespace()).Build(),
		builder.NewServiceAccount().WithName(utils.PodPlacementControllerName).WithNamespace(utils.Namespace()).Build(),
		builder.NewServiceAccount().WithName(utils.PodPlacementWebhookName).WithNamespace(utils.Namespace()).Build(),
		builder.NewClusterPodPlacementConfig().WithName(common.SingletonResourceObjectName),
	}
}

func ValidateDeletion(cl client.Client, ctx context.Context) func(gomega.Gomega) {
	return func(g gomega.Gomega) {
		for _, obj := range getObjects() {
			newObj := obj.DeepCopyObject().(client.Object)
			err := cl.Get(ctx, client.ObjectKeyFromObject(obj), newObj)
			g.Expect(err).To(gomega.HaveOccurred(), "the object should be deleted", err)
			g.Expect(errors.IsNotFound(err)).To(gomega.BeTrue(), "the error should be \"Not found\"", err)
		}
	}
}

func ValidateCreation(cl client.Client, ctx context.Context) func(gomega.Gomega) {
	return func(g gomega.Gomega) {
		ginkgo.By("Verify all objects exist")
		for _, obj := range getObjects() {
			newObj := obj.DeepCopyObject().(client.Object)
			err := cl.Get(ctx, client.ObjectKeyFromObject(obj), newObj)
			g.Expect(err).NotTo(gomega.HaveOccurred(), "the object should be created", err)
			g.Expect(newObj).NotTo(gomega.BeNil(), "the object should not be nil")
			g.Expect(newObj.GetDeletionTimestamp().IsZero()).To(gomega.BeTrue(), "the object should not be marked for deletion")
		}
		ginkgo.By("Verify the ClusterPodPlacementConfig conditions")
		VerifyConditions(ctx, cl,
			NewConditionTypeStatusTuple(v1beta1.AvailableType, corev1.ConditionTrue),
			NewConditionTypeStatusTuple(v1beta1.ProgressingType, corev1.ConditionFalse),
			NewConditionTypeStatusTuple(v1beta1.DegradedType, corev1.ConditionFalse),
			NewConditionTypeStatusTuple(v1beta1.PodPlacementControllerNotRolledOutType, corev1.ConditionFalse),
			NewConditionTypeStatusTuple(v1beta1.PodPlacementWebhookNotRolledOutType, corev1.ConditionFalse),
			NewConditionTypeStatusTuple(v1beta1.MutatingWebhookConfigurationNotAvailable, corev1.ConditionFalse),
			NewConditionTypeStatusTuple(v1beta1.DeprovisioningType, corev1.ConditionFalse),
		)
	}
}
