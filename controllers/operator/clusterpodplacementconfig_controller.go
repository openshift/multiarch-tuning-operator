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

const waitingForUngatingPodsError = "waiting for pods with the scheduling gate to be ungated"

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
	log.V(3).Info("Reconciling ClusterPodPlacementConfig...")
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

// handleDelete handles the deletion of the PodPlacement operand's resources.
func (r *ClusterPodPlacementConfigReconciler) handleDelete(ctx context.Context,
	clusterPodPlacementConfig *multiarchv1beta1.ClusterPodPlacementConfig) error {
	// The ClusterPodPlacementConfig is being deleted, cleanup the resources
	log := ctrllog.FromContext(ctx).WithValues("operation", "handleDelete")
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
	log.Info("Deleting the PodPlacement operand's resources")
	err := utils.DeleteResources(ctx, objsToDelete)
	// NOTE: err aggregates non-nil errors, excluding NotFound errors
	if err != nil {
		log.Error(err, "Unable to delete resources")
		return err
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
	dlog := log.WithValues("deployment", PodPlacementControllerName)
	dlog.Info("Removing the finalizer from the deployment")
	dlog.V(4).Info("Fetching the deployment", "Deployment", PodPlacementControllerName)
	ppcDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      PodPlacementControllerName,
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
		buildDeployment(clusterPodPlacementConfig, PodPlacementControllerName, 2,
			utils.PodPlacementFinalizerName, "--leader-elect",
			"--enable-ppc-controllers",
		),
		buildDeployment(clusterPodPlacementConfig, PodPlacementWebhookName, 3, "",
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

	if !controllerutil.ContainsFinalizer(clusterPodPlacementConfig, utils.PodPlacementFinalizerName) {
		// Add the finalizer to the object
		log.Info("Adding finalizer to the ClusterPodPlacementConfig")
		controllerutil.AddFinalizer(clusterPodPlacementConfig, utils.PodPlacementFinalizerName)
		if err := r.Update(ctx, clusterPodPlacementConfig); err != nil {
			log.Error(err, "Unable to update finalizers in the ClusterPodPlacementConfig")
			return err
		}
	}
	return nil
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
