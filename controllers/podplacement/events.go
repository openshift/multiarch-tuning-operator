package podplacement

import "github.com/openshift/multiarch-tuning-operator/pkg/utils"

const (
	ArchitecturePredicatesConflict                = "ArchAwarePredicatesConflict"
	ImageArchitectureInspectionError              = "ArchAwareInspectionError"
	ArchitectureAwareNodeAffinitySet              = "ArchAwarePredicateSet"
	ArchitectureAwareSchedulingGateAdded          = "ArchAwareSchedGateAdded"
	ArchitectureAwareSchedulingGateRemovalFailure = "ArchAwareSchedGateRemovalFailed"
	ArchitectureAwareSchedulingGateRemovalSuccess = "ArchAwareSchedGateRemovalSuccess"

	SchedulingGateAddedMsg              = "Successfully gated with the " + utils.SchedulingGateName + " scheduling gate"
	SchedulingGateRemovalSuccessMsg     = "Successfully removed the " + utils.SchedulingGateName + " scheduling gate"
	SchedulingGateRemovalFailureMsg     = "Failed to remove the scheduling gate \"" + utils.SchedulingGateName + "\""
	ArchitecturePredicatesConflictMsg   = "All the scheduling predicates already include architecture-specific constraints"
	ArchitecturePredicateSetupMsg       = "Set the nodeAffinity for the architecture to "
	ImageArchitectureInspectionErrorMsg = "Failed to retrieve the supported architectures: "
)
