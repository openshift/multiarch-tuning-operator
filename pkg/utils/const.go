package utils

const (
	ControllerNameKey = "controller"
	OperandLabelKey   = "multiarch.openshift.io/operand"
	OperatorName      = "multiarch-tuning-operator"
)

const (
	ArchitectureAmd64   = "amd64"
	ArchitectureArm64   = "arm64"
	ArchitecturePpc64le = "ppc64le"
	ArchitectureS390x   = "s390x"
)

const (
	ArchLabel                       = "kubernetes.io/arch"
	NodeAffinityLabel               = "multiarch.openshift.io/node-affinity"
	NodeAffinityLabelValueSet       = "set"
	NodeAffinityLabelValueNotSet    = "not-set"
	HostnameLabel                   = "kubernetes.io/hostname"
	SchedulingGateLabel             = "multiarch.openshift.io/scheduling-gate"
	SchedulingGateLabelValueGated   = "gated"
	SchedulingGateLabelValueRemoved = "removed"
	PodPlacementFinalizerName       = "finalizers.multiarch.openshift.io/pod-placement"
)

const (
	// SchedulingGateName is the name of the Scheduling Gate
	SchedulingGateName            = "multiarch.openshift.io/scheduling-gate"
	MasterNodeSelectorLabel       = "node-role.kubernetes.io/master"
	ControlPlaneNodeSelectorLabel = "node-role.kubernetes.io/control-plane"
)
