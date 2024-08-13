package framework

import (
	"context"

	"github.com/onsi/gomega"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/common"
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConditionTypeStatusTuple struct {
	ConditionType   string
	ConditionStatus v1.ConditionStatus
}

func NewConditionTypeStatusTuple(conditionType string, conditionStatus v1.ConditionStatus) ConditionTypeStatusTuple {
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
