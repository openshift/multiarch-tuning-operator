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
	"errors"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/library-go/pkg/operator/events"

	"go.uber.org/zap/zapcore"

	multiarchv1beta1 "github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
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
	operandName        = "pod-placement-controller"
	priorityClassName  = "system-cluster-critical"
	serviceAccountName = "multiarch-tuning-operator-controller-manager"
)

const (
	waitingForUngatingPodsError         = "waiting for pods with the scheduling gate to be ungated"
	waitingForWebhookSInterruptionError = "re-queueing to ensure the webhook objects deletion interrupt pods gating before checking the pods gating status"
	clusterPodPlacementConfigNotReady   = "cluster pod placement config is not ready yet. re-queueing"
)

//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=clusterpodplacementconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=clusterpodplacementconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=clusterpodplacementconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;update;patch;create;delete;list;watch
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations/status,verbs=get

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=events,verbs=create

// Reconcile reconciles the ClusterPodPlacementConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *ClusterPodPlacementConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.V(3).Info("+++++++++++++++++++ Reconciling ClusterPodPlacementConfig +++++++++++++++++++")
	// Lookup the ClusterPodPlacementConfig instance for this reconcile request
	clusterPodPlacementConfig := &multiarchv1beta1.ClusterPodPlacementConfig{}
	var err error

	if err = r.Get(ctx, client.ObjectKey{
		Name: req.NamespacedName.Name,
	}, clusterPodPlacementConfig); err != nil {
		log.Error(err, "Unable to fetch ClusterPodPlacementConfig")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(3).Info("ClusterPodPlacementConfig fetched...", "name", clusterPodPlacementConfig.Name)
	err = r.dependentsStatusToClusterPodPlacementConfig(ctx, clusterPodPlacementConfig)
	if err != nil {
		log.Error(err, "Unable to retrieve the status of the PodPlacementConfig dependencies")
		return ctrl.Result{}, err
	}
	switch {
	case !clusterPodPlacementConfig.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(clusterPodPlacementConfig, utils.PodPlacementFinalizerName):
		log.V(4).Info("the ClusterPodPlacementConfig object is being deleted, and the finalizer has already been removed successfully.")
		return ctrl.Result{}, nil
	case !clusterPodPlacementConfig.DeletionTimestamp.IsZero():
		// Only execute deletion if the object is being deleted and the finalizer is present
		return ctrl.Result{}, r.handleDelete(ctx, clusterPodPlacementConfig)
	}
	return ctrl.Result{}, r.reconcile(ctx, clusterPodPlacementConfig)
}

// dependentsStatusToClusterPodPlacementConfig gathers the status of the dependents of the ClusterPodPlacementConfig object.
// The status is propagated to the ClusterPodPlacementConfig object.
func (r *ClusterPodPlacementConfigReconciler) dependentsStatusToClusterPodPlacementConfig(ctx context.Context, config *multiarchv1beta1.ClusterPodPlacementConfig) error {
	log := ctrllog.FromContext(ctx).WithValues("ClusterPodPlacementConfig", config.Name,
		"function", "updateStatus")
	podPlacementController, err := r.ClientSet.AppsV1().Deployments(utils.Namespace()).Get(ctx, utils.PodPlacementControllerName, metav1.GetOptions{})
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "Unable to get the PodPlacement controller deployment")
		return err
	}
	podPlacementWebhook, err := r.ClientSet.AppsV1().Deployments(utils.Namespace()).Get(ctx, utils.PodPlacementWebhookName, metav1.GetOptions{})
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "Unable to get the PodPlacement webhook deployment")
		return err
	}
	_, err = r.ClientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, utils.PodMutatingWebhookConfigurationName, metav1.GetOptions{})
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "Unable to get the mutating webhook configuration")
		return err
	}
	config.Status.Build(
		isDeploymentAvailable(podPlacementController), isDeploymentAvailable(podPlacementWebhook),
		isDeploymentUpToDate(podPlacementController), isDeploymentUpToDate(podPlacementWebhook),
		// err == nil means the MutatingWebhookConfiguration is available
		err == nil, !config.DeletionTimestamp.IsZero())
	return nil
}

// handleDelete handles the deletion of the PodPlacement operand's resources.
func (r *ClusterPodPlacementConfigReconciler) handleDelete(ctx context.Context,
	clusterPodPlacementConfig *multiarchv1beta1.ClusterPodPlacementConfig) error {
	// The ClusterPodPlacementConfig is being deleted, cleanup the resources
	log := ctrllog.FromContext(ctx).WithValues("operation", "handleDelete")
	// The error by the updateStatus function, if any, is ignored, as the deletion should always proceed.
	// We execute the update here because this function returns multiple times before the whole deletion process is completed.
	// Executing it here ensures that the conditions are updated throughout the deletion process.
	_ = r.updateStatus(ctx, clusterPodPlacementConfig)
	objsToDelete := []utils.ToDeleteRef{
		{
			NamespacedTypedClient: r.ClientSet.AdmissionregistrationV1().MutatingWebhookConfigurations(),
			ObjName:               utils.PodMutatingWebhookConfigurationName,
		},
		{
			NamespacedTypedClient: r.ClientSet.CoreV1().Services(utils.Namespace()),
			ObjName:               utils.PodPlacementWebhookName,
		},
		{
			NamespacedTypedClient: r.ClientSet.CoreV1().Services(utils.Namespace()),
			ObjName:               utils.PodPlacementControllerMetricsServiceName,
		},
		{
			NamespacedTypedClient: r.ClientSet.CoreV1().Services(utils.Namespace()),
			ObjName:               utils.PodPlacementWebhookMetricsServiceName,
		},
		{
			NamespacedTypedClient: r.ClientSet.AppsV1().Deployments(utils.Namespace()),
			ObjName:               utils.PodPlacementWebhookName,
		},
		{
			NamespacedTypedClient: r.ClientSet.AppsV1().Deployments(utils.Namespace()),
			ObjName:               utils.PodPlacementControllerName,
		},
	}
	log.Info("Deleting the pod placement operand's resources")
	// NOTE: err aggregates non-nil errors, excluding NotFound errors
	if err := utils.DeleteResources(ctx, objsToDelete); err != nil {
		log.Error(err, "Unable to delete resources")
		return err
	}
	_, err := r.ClientSet.CoreV1().Services(utils.Namespace()).Get(ctx, utils.PodPlacementWebhookName, metav1.GetOptions{})
	// We look for the webhook service to ensure that the webhook has stopped communicating with the API server.
	// If the error is nil, the service was found. If the error is not nil and the error is not NotFound, some other error occurred.
	// In both the cases we return an error, to requeue the request and ensure no race conditions between the verification of the
	// pods gating status and the webhook stopping to communicate with the API server.
	if err == nil || client.IgnoreNotFound(err) != nil {
		return errors.New(waitingForWebhookSInterruptionError)
	}

	log.Info("Looking for pods with the scheduling gate")
	// get pending pods as we cannot query for the scheduling gate
	pods, err := r.ClientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		log.Error(err, "Unable to list pods")
		return err
	}
	if len(pods.Items) != 0 {
		// Check if any pods really have our scheduling gate
		found := false
		for _, pod := range pods.Items {
			for _, sg := range pod.Spec.SchedulingGates {
				log.V(4).Info("Pod has scheduling gate", "pod", pod.Name, "gate", sg.Name)
				if sg.Name == utils.SchedulingGateName {
					log.Info("Found pod with the pod placement scheduling gate", "pod", pod.Name)
					found = true
				}
			}
		}
		if found {
			return errors.New(waitingForUngatingPodsError)
		}
	}

	// The pods have been ungated and no other errors occurred, so we can remove the finalizer
	log.Info("Pods have been ungated")
	log = log.WithValues("finalizer", utils.PodPlacementFinalizerName)
	dlog := log.WithValues("deployment", utils.PodPlacementControllerName)
	dlog.Info("Removing the finalizer from the deployment")
	dlog.V(4).Info("Fetching the deployment", "Deployment", utils.PodPlacementControllerName)
	ppcDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      utils.PodPlacementControllerName,
		Namespace: utils.Namespace(),
	}, ppcDeployment)
	if client.IgnoreNotFound(err) != nil {
		dlog.Error(err, "Unable to fetch the deployment")
		return err
	}
	controllerutil.RemoveFinalizer(ppcDeployment, utils.PodPlacementFinalizerName)
	dlog.V(4).Info("Updating the deployment")
	if err = r.Update(ctx, ppcDeployment); err != nil {
		dlog.Error(err, "Unable to remove the finalizer")
		return err
	}
	// we can remove the finalizer in the ClusterPodPlacementConfig object now
	log.Info("Removing the finalizer from the ClusterPodPlacementConfig")
	controllerutil.RemoveFinalizer(clusterPodPlacementConfig, utils.PodPlacementFinalizerName)
	if err = r.Update(ctx, clusterPodPlacementConfig); err != nil {
		log.Error(err, "Unable to remove finalizers.",
			clusterPodPlacementConfig.Kind, clusterPodPlacementConfig.Name)
		return err
	}
	return err
}

// reconcile reconciles the ClusterPodPlacementConfig operand's resources.
func (r *ClusterPodPlacementConfigReconciler) reconcile(ctx context.Context, clusterPodPlacementConfig *multiarchv1beta1.ClusterPodPlacementConfig) error {
	log := ctrllog.FromContext(ctx)
	if int8(utils.AtomicLevel.Level()) != int8(clusterPodPlacementConfig.Spec.LogVerbosity.ToZapLevelInt()) {
		log.Info("Setting log level", "level", -clusterPodPlacementConfig.Spec.LogVerbosity.ToZapLevelInt())
		utils.AtomicLevel.SetLevel(zapcore.Level(-clusterPodPlacementConfig.Spec.LogVerbosity.ToZapLevelInt()))
	}

	objects := []client.Object{
		// The finalizer will not affect the reconciliation of ReplicaSets and Pods
		// when updates to the ClusterPodPlacementConfig are made.
		buildDeployment(clusterPodPlacementConfig, utils.PodPlacementControllerName, 2,
			utils.PodPlacementFinalizerName, "--leader-elect",
			"--enable-ppc-controllers",
		),
		buildDeployment(clusterPodPlacementConfig, utils.PodPlacementWebhookName, 3, "",
			"--enable-ppc-webhook",
		),
		buildService(utils.PodPlacementControllerName, utils.PodPlacementControllerName,
			443, intstr.FromInt32(9443)),
		buildService(utils.PodPlacementWebhookName, utils.PodPlacementWebhookName,
			443, intstr.FromInt32(9443)),
		buildService(
			utils.PodPlacementControllerMetricsServiceName, utils.PodPlacementControllerName,
			8443, intstr.FromInt32(8443)),
		buildService(
			utils.PodPlacementWebhookMetricsServiceName, utils.PodPlacementWebhookName,
			8443, intstr.FromInt32(8443)),
	}
	// We ensure the MutatingWebHookConfiguration is created and present only if the operand is ready to serve the admission request and add/remove the scheduling gate.
	shouldEnsureMWC := clusterPodPlacementConfig.Status.CanDeployMutatingWebhook()
	shouldDeleteMWC := !shouldEnsureMWC && !clusterPodPlacementConfig.Status.IsMutatingWebhookConfigurationNotAvailable()
	if shouldEnsureMWC {
		objects = append(objects, buildMutatingWebhookConfiguration(clusterPodPlacementConfig))
	}
	if shouldDeleteMWC {
		log.Info("Deleting the mutating webhook configuration as the operand is not ready to serve the admission request or remove the scheduling gate")
		_ = r.ClientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, utils.PodMutatingWebhookConfigurationName, metav1.DeleteOptions{})
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

	if !controllerutil.ContainsFinalizer(clusterPodPlacementConfig, utils.PodPlacementFinalizerName) {
		// Add the finalizer to the object
		log.Info("Adding finalizer to the ClusterPodPlacementConfig")
		controllerutil.AddFinalizer(clusterPodPlacementConfig, utils.PodPlacementFinalizerName)
		if err := r.Update(ctx, clusterPodPlacementConfig); err != nil {
			log.Error(err, "Unable to update finalizers in the ClusterPodPlacementConfig")
			return err
		}
	}
	return r.updateStatus(ctx, clusterPodPlacementConfig)
}

// updateStatus updates the status of the ClusterPodPlacementConfig object.
// It returns an error if the object is progressing or the status update fails. Otherwise, it returns nil.
// When it returns an error, the caller should requeue the request, unless the Reconciler is handling the deletion of the object.
func (r *ClusterPodPlacementConfigReconciler) updateStatus(ctx context.Context, config *multiarchv1beta1.ClusterPodPlacementConfig) error {
	log := ctrllog.FromContext(ctx).WithValues("ClusterPodPlacementConfig", config.Name,
		"function", "updateStatus")
	log.V(4).Info("----------------- ClusterPodPlacementConfig status report ------------------",
		"ready", config.Status.IsReady(),
		"progressing", config.Status.IsProgressing(),
		"degraded", config.Status.IsDegraded(),
		"deprovisioning", config.Status.IsDeprovisioning(),
		"podPlacementControllerNotReady", config.Status.IsPodPlacementControllerNotReady(),
		"podPlacementWebhookNotReady", config.Status.IsPodPlacementWebhookNotReady(),
		"mutatingWebhookConfigurationNotAvailable", config.Status.IsMutatingWebhookConfigurationNotAvailable())

	progressing := config.Status.IsProgressing()
	if err := r.Status().Update(ctx, config); err != nil {
		log.Error(err, "Unable to update conditions in the ClusterPodPlacementConfig")
		return err
	}
	var err error = nil
	if progressing {
		err = errors.New(clusterPodPlacementConfigNotReady)
	}
	return err
}

func isDeploymentAvailable(deployment *appsv1.Deployment) bool {
	if deployment == nil {
		return false
	}
	return deployment.Status.AvailableReplicas > 0
}

func isDeploymentUpToDate(deployment *appsv1.Deployment) bool {
	if deployment == nil {
		return false
	}
	expectedReplicas := int32(-1)
	if deployment.Spec.Replicas != nil {
		expectedReplicas = *deployment.Spec.Replicas
	}
	return deployment.Status.UpdatedReplicas == expectedReplicas &&
		deployment.Status.Replicas == expectedReplicas &&
		deployment.Status.AvailableReplicas == expectedReplicas &&
		deployment.Status.ReadyReplicas == expectedReplicas &&
		deployment.Status.UnavailableReplicas == 0 &&
		deployment.Status.ObservedGeneration == deployment.Generation
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterPodPlacementConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiarchv1beta1.ClusterPodPlacementConfig{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&admissionv1.MutatingWebhookConfiguration{}).
		Complete(r)
}
