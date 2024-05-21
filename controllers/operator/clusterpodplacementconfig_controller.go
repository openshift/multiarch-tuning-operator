/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"context"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/library-go/pkg/operator/events"

	multiarchv1alpha1 "github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1alpha1"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

// ClusterPodPlacementConfigReconciler reconciles a ClusterPodPlacementConfig object
type ClusterPodPlacementConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	ClientSet *kubernetes.Clientset
	Recorder  events.Recorder
}

const (
	podMutatingWebhookName              = "pod-placement-scheduling-gate.multiarch.openshift.io"
	podMutatingWebhookConfigurationName = "pod-placement-mutating-webhook-configuration"

	PodPlacementControllerName               = "pod-placement-controller"
	podPlacementControllerMetricsServiceName = "pod-placement-controller-metrics-service"
	PodPlacementWebhookName                  = "pod-placement-web-hook"
	podPlacementWebhookMetricsServiceName    = "pod-placement-web-hook-metrics-service"
	operandName                              = "pod-placement-controller"
	priorityClassName                        = "system-cluster-critical"
	serviceAccountName                       = "multiarch-tuning-operator-controller-manager"
)

//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=clusterpodplacementconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=clusterpodplacementconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=clusterpodplacementconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;update;patch;create;delete;list;watch
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations/status,verbs=get

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=events,verbs=create

// Reconcile reconciles the ClusterPodPlacementConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *ClusterPodPlacementConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.V(3).Info("Reconciling ClusterPodPlacementConfig...")
	// Lookup the ClusterPodPlacementConfig instance for this reconcile request
	clusterPodPlacementConfig := &multiarchv1alpha1.ClusterPodPlacementConfig{}
	var err error

	if err = r.Get(ctx, client.ObjectKey{
		Name: req.NamespacedName.Name,
	}, clusterPodPlacementConfig); client.IgnoreNotFound(err) != nil {
		log.Error(err, "Unable to fetch ClusterPodPlacementConfig")
		return ctrl.Result{}, err
	}

	log.V(3).Info("ClusterPodPlacementConfig fetched...", "name", clusterPodPlacementConfig.Name)
	if req.NamespacedName.Name == multiarchv1alpha1.SingletonResourceObjectName {
		if apierrors.IsNotFound(err) || !clusterPodPlacementConfig.DeletionTimestamp.IsZero() {
			// Only execute deletion iff the name of the object is 'cluster' and the object is being deleted or not found
			return ctrl.Result{}, r.handleDelete(ctx)
		}
		return ctrl.Result{}, r.reconcile(ctx, clusterPodPlacementConfig)
	}

	// If we hit here, the ClusterPodPlacementConfig has an invalid name.
	log.V(3).Info("ClusterPodPlacementConfig name is not cluster", "name", clusterPodPlacementConfig.Name)
	if clusterPodPlacementConfig.DeletionTimestamp.IsZero() {
		// Only execute deletion iff the name of the object is different from 'cluster' and the object is not yet deleted.
		log.V(3).Info("Deleting ClusterPodPlacementConfig", "name", clusterPodPlacementConfig.Name)
		err := r.Delete(ctx, clusterPodPlacementConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	log.Info("The ClusterPodPlacementConfig is already pending deletion, nothing to do.", "name", clusterPodPlacementConfig.Name)
	return ctrl.Result{}, nil
}

// handleDelete handles the deletion of the PodPlacement operand's resources.
func (r *ClusterPodPlacementConfigReconciler) handleDelete(ctx context.Context) error {
	// The ClusterPodPlacementConfig is being deleted, cleanup the resources
	log := ctrllog.FromContext(ctx)
	log.Info("Deleting the PodPlacement operand's resources")

	objsToDelete := []utils.ToDeleteRef{
		{
			NamespacedTypedClient: r.ClientSet.AppsV1().Deployments(utils.Namespace()),
			ObjName:               PodPlacementControllerName,
		},
		{
			NamespacedTypedClient: r.ClientSet.AppsV1().Deployments(utils.Namespace()),
			ObjName:               PodPlacementWebhookName,
		},
		{
			NamespacedTypedClient: r.ClientSet.CoreV1().Services(utils.Namespace()),
			ObjName:               PodPlacementWebhookName,
		},
		{
			NamespacedTypedClient: r.ClientSet.CoreV1().Services(utils.Namespace()),
			ObjName:               podPlacementControllerMetricsServiceName,
		},
		{
			NamespacedTypedClient: r.ClientSet.CoreV1().Services(utils.Namespace()),
			ObjName:               podPlacementWebhookMetricsServiceName,
		},
		{
			NamespacedTypedClient: r.ClientSet.AdmissionregistrationV1().MutatingWebhookConfigurations(),
			ObjName:               podMutatingWebhookConfigurationName,
		},
	}
	return utils.DeleteResources(ctx, objsToDelete)
}

// reconcile reconciles the ClusterPodPlacementConfig operand's resources.
func (r *ClusterPodPlacementConfigReconciler) reconcile(ctx context.Context, clusterPodPlacementConfig *multiarchv1alpha1.ClusterPodPlacementConfig) error {
	log := ctrllog.FromContext(ctx)
	objects := []client.Object{
		buildDeployment(clusterPodPlacementConfig, PodPlacementControllerName, 2,
			"--leader-elect",
			"--enable-ppc-controllers",
		),
		buildDeployment(clusterPodPlacementConfig, PodPlacementWebhookName, 3,
			"--enable-ppc-webhook",
		),
		buildService(PodPlacementControllerName, PodPlacementControllerName,
			443, intstr.FromInt32(9443)),
		buildService(PodPlacementWebhookName, PodPlacementWebhookName,
			443, intstr.FromInt32(9443)),
		buildService(
			podPlacementControllerMetricsServiceName, PodPlacementControllerName,
			8443, intstr.FromInt32(8443)),
		buildService(
			podPlacementWebhookMetricsServiceName, PodPlacementWebhookName,
			8443, intstr.FromInt32(8443)),
		buildMutatingWebhookConfiguration(clusterPodPlacementConfig),
	}

	errs := make([]error, 0)
	for _, o := range objects {
		if err := ctrl.SetControllerReference(clusterPodPlacementConfig, o, r.Scheme); err != nil {
			log.Error(err, "Unable to set controller reference", "name", o.GetName())
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errorutils.NewAggregate(errs)
	}

	if err := utils.ApplyResources(ctx, r.ClientSet, r.Recorder, objects); err != nil {
		log.Error(err, "Unable to apply resources")
		return err
	}

	/* TODO: Updates to the ClusterPodPlacementConfig's status will probably be considered in the future to address the
	 * ordered un-installation of the operator and operands.
	 */
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterPodPlacementConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiarchv1alpha1.ClusterPodPlacementConfig{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&admissionv1.MutatingWebhookConfiguration{}).
		Complete(r)
}
