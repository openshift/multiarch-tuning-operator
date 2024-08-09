package podplacement

const (
	ArchitecturePredicatesConflict                = "ArchAwarePredicatesConflict"
	ImageArchitectureInspectionError              = "ArchAwareInspectionError"
	ArchitectureAwareNodeAffinitySet              = "ArchAwarePredicateSet"
	ArchitectureAwareSchedulingGateAdded          = "ArchAwareSchedGateAdded"
	ArchitectureAwareSchedulingGateRemovalFailure = "ArchAwareSchedGateRemovalFailed"
	ArchitectureAwareSchedulingGateRemovalSuccess = "ArchAwareSchedGateRemovalSuccess"

	SchedulingGateAddedMsg              = "Successfully gated with the " + schedulingGateName + " scheduling gate"
	SchedulingGateRemovalSuccessMsg     = "Successfully removed the " + schedulingGateName + " scheduling gate"
	SchedulingGateRemovalFailureMsg     = "Failed to remove the scheduling gate \"" + schedulingGateName + "\""
	ArchitecturePredicatesConflictMsg   = "All the scheduling predicates already include architecture-specific constraints"
	ArchitecturePredicateSetupMsg       = "Set the nodeAffinity for the architecture to "
	ImageArchitectureInspectionErrorMsg = "Failed to retrieve the supported architectures: "
)
