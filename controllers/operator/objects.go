package operator

import (
	"fmt"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

const (
	CREATE = "create"
	UPDATE = "update"
	PATCH  = "patch"
	LIST   = "list"
	WATCH  = "watch"
	GET    = "get"
	USE    = "use"
	DELETE = "delete"

	serviceAccountKind = "ServiceAccount"
	roleKind           = "Role"
	clusterRoleKind    = "ClusterRole"
)

func buildMutatingWebhookConfiguration(clusterPodPlacementConfig *v1beta1.ClusterPodPlacementConfig) *admissionv1.MutatingWebhookConfiguration {
	return &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.PodMutatingWebhookConfigurationName,
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: utils.PodPlacementWebhookName,
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      utils.PodPlacementWebhookName,
						Namespace: utils.Namespace(),
						Path:      utils.NewPtr("/add-pod-scheduling-gate"),
					},
				},
				NamespaceSelector: clusterPodPlacementConfig.Spec.NamespaceSelector,
				FailurePolicy:     utils.NewPtr(admissionv1.Ignore),
				SideEffects:       utils.NewPtr(admissionv1.SideEffectClassNone),
				Name:              utils.PodMutatingWebhookName,
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: []admissionv1.OperationType{
							admissionv1.Create,
						},
						Rule: admissionv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
			},
		},
	}
}

func buildService(name string, controllerName string, port int32, targetPort intstr.IntOrString) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: utils.Namespace(),
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: controllerName,
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       port,
					TargetPort: targetPort,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: controllerName,
			},
		},
	}
}

func buildWebhookDeployment(clusterPodPlacementConfig *v1beta1.ClusterPodPlacementConfig) *appsv1.Deployment {
	return buildDeployment(clusterPodPlacementConfig, utils.PodPlacementWebhookName, 3, utils.PodPlacementWebhookName, "",
		"--enable-ppc-webhook",
	)

}

func buildControllerDeployment(clusterPodPlacementConfig *v1beta1.ClusterPodPlacementConfig) *appsv1.Deployment {
	d := buildDeployment(clusterPodPlacementConfig, utils.PodPlacementControllerName, 2, utils.PodPlacementControllerName,
		utils.PodPlacementFinalizerName, "--leader-elect", "--enable-ppc-controllers",
	)
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: "docker-conf",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/docker/",
					Type: utils.NewPtr(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		corev1.Volume{
			Name: "containers-conf",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/containers/",
					Type: utils.NewPtr(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
	)
	d.Spec.Template.Spec.Containers[0].VolumeMounts = append(d.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "docker-conf",
			MountPath: "/etc/docker/",
			ReadOnly:  true,
		},
		corev1.VolumeMount{
			Name:      "containers-conf",
			MountPath: "/etc/containers/",
			ReadOnly:  true,
		},
	)
	return d
}

func buildDeployment(clusterPodPlacementConfig *v1beta1.ClusterPodPlacementConfig,
	name string, replicas int32, serviceAccount string, finalizer string, args ...string) *appsv1.Deployment {
	finalizers := make([]string, 0)
	if finalizer != "" {
		finalizers = append(finalizers, finalizer)
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: utils.Namespace(),
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: name,
			},
			Finalizers: finalizers,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.NewPtr(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					utils.OperandLabelKey:   operandName,
					utils.ControllerNameKey: name,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       utils.NewPtr(intstr.FromString("25%")),
					MaxUnavailable: utils.NewPtr(intstr.FromString("25%")),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						utils.OperandLabelKey:   operandName,
						utils.ControllerNameKey: name,
					},
					Annotations: map[string]string{
						// See https://github.com/openshift/enhancements/blob/c5b9aea25e/enhancements/workload-partitioning/management-workload-partitioning.md
						"target.workload.openshift.io/management": "{\"effect\": \"PreferredDuringScheduling\"}",
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      utils.ArchLabel,
												Operator: corev1.NodeSelectorOpIn,
												Values: []string{
													utils.ArchitectureAmd64,
													utils.ArchitectureArm64,
													utils.ArchitectureS390x,
													utils.ArchitecturePpc64le,
												},
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           utils.Image(),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "NAMESPACE",
									Value: utils.Namespace(),
								},
							},
							Args: append([]string{
								"--health-probe-bind-address=:8081",
								"--metrics-bind-address=127.0.0.1:8080",
								fmt.Sprintf("--initial-log-level=%d",
									clusterPodPlacementConfig.Spec.LogVerbosity.ToZapLevelInt()),
							}, args...),
							Command: []string{
								"/manager",
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt32(8081),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      1,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/readyz",
										Port:   intstr.FromInt32(8081),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: utils.NewPtr(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								Privileged:   utils.NewPtr(false),
								RunAsNonRoot: utils.NewPtr(true),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "webhook-server-cert",
									MountPath: "/var/run/manager/tls",
									ReadOnly:  true,
								},
								{
									Name:      "ca-projected-volume",
									MountPath: "/etc/ssl/certs",
									ReadOnly:  true,
								},
							},
						}, {
							Name:            "kube-rbac-proxy",
							Image:           "gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1@sha256:d4883d7c622683b3319b5e6b3a7edfbf2594c18060131a8bf64504805f875522",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"--secure-listen-address=0.0.0.0:8443",
								"--upstream=http://127.0.0.1:8080/",
								"--logtostderr=true",
								"--v=0",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
									Name:          "https",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: utils.NewPtr(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								Privileged:   utils.NewPtr(false),
								RunAsNonRoot: utils.NewPtr(true),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
							},
						},
					},
					PriorityClassName:  priorityClassName,
					ServiceAccountName: serviceAccount,
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       "kubernetes.io/hostname",
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									utils.OperandLabelKey:   operandName,
									utils.ControllerNameKey: name,
								},
							},
							MatchLabelKeys: []string{"pod-template-hash"},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "webhook-server-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  name,
									DefaultMode: utils.NewPtr(int32(420)),
								},
							},
						},
						{
							Name: "ca-projected-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									DefaultMode: utils.NewPtr(int32(420)),
									Sources: []corev1.VolumeProjection{
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "openshift-service-ca.crt",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "service-ca.crt",
														Path: "openshift-ca.crt",
													},
												},
												Optional: utils.NewPtr(true), // Account for the case where the ConfigMap does not exist (non openshift clusters)
											},
										},
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "kube-root-ca.crt",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "ca.crt",
														Path: "kube-root-ca.crt",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildClusterRole(name string, rules []rbacv1.PolicyRule) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: name,
			},
		},
		Rules: rules,
	}
}

func buildClusterRoleBinding(name string, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: name,
			},
		},
		RoleRef:  roleRef,
		Subjects: subjects,
	}
}

func buildRoleBinding(name string, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: utils.Namespace(),
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: name,
			},
		},
		RoleRef:  roleRef,
		Subjects: subjects,
	}
}

func buildServiceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: utils.Namespace(),
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: name,
			},
		},
	}
}

func buildClusterRoleWebhook() *rbacv1.ClusterRole {
	return buildClusterRole(utils.PodPlacementWebhookName, []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{CREATE, PATCH},
		},
		{
			APIGroups: []string{v1beta1.GroupVersion.Group},
			Resources: []string{v1beta1.ClusterPodPlacementConfigResource},
			Verbs:     []string{LIST, WATCH, GET},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{LIST, WATCH, GET},
		},
	})
}

func buildClusterRoleController() *rbacv1.ClusterRole {
	return buildClusterRole(utils.PodPlacementControllerName, []rbacv1.PolicyRule{
		{
			APIGroups: []string{"security.openshift.io"},
			Resources: []string{"securitycontextconstraints"},
			Verbs:     []string{USE},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{LIST, WATCH, GET, UPDATE},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{CREATE, PATCH},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods/status"},
			Verbs:     []string{UPDATE},
		},
		{
			APIGroups: []string{v1beta1.GroupVersion.Group},
			Resources: []string{v1beta1.ClusterPodPlacementConfigResource},
			Verbs:     []string{LIST, WATCH, GET},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "secrets"},
			Verbs:     []string{LIST, WATCH, GET},
		},
		{
			APIGroups: []string{"operator.openshift.io"},
			Resources: []string{"imagecontentsourcepolicies"},
			Verbs:     []string{LIST, WATCH, GET},
		},
		{
			APIGroups: []string{"config.openshift.io"},
			Resources: []string{"imagetagmirrorsets", "imagedigestmirrorsets", "images"},
			Verbs:     []string{LIST, WATCH, GET},
		},
	})
}

func buildRoleController() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.PodPlacementControllerName,
			Namespace: utils.Namespace(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{LIST, WATCH, GET, UPDATE, PATCH, CREATE, DELETE},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{LIST, WATCH, GET, UPDATE, PATCH, CREATE, DELETE},
			},
		},
	}
}
