/*
Copyright 2023.

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

package multiarch

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	multiarchv1alpha1 "multiarch-operator/apis/multiarch/v1alpha1"
)

const (
	replaceWebhooksValueTemplate = `{ "op": "replace", "path": "/webhooks/0/namespaceSelector", "value": %s }`
)

// PodPlacementConfigReconciler reconciles a PodPlacementConfig object
type PodPlacementConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
}

func generatePatchBytes(ops string) []byte {
	return []byte(fmt.Sprintf("[%s]", ops))
}

//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=podplacementconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=podplacementconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multiarch.openshift.io,resources=podplacementconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodPlacementConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PodPlacementConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx, "podplacementconfig", req.NamespacedName.Name, "namespace")

	// Lookup the PodPlacementConfig instance for this reconcile request
	podplacementconfig := &multiarchv1alpha1.PodPlacementConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: "podplacementconfig-sample", Namespace: ""}, podplacementconfig); err != nil {
		log.Error(err, "Unable to fetch PodPlacementConfig")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	podplacementwebhook, err := r.Clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "multiarch-operator-mutating-webhook-configuration", metav1.GetOptions{})
	if err != nil {
		log.Error(err, "Unable to fetch mutating webhook")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !reflect.DeepEqual(podplacementwebhook.Webhooks[0].NamespaceSelector, podplacementconfig.Spec.NamespaceSelector) {
		nsselectorbytes, err := json.Marshal(podplacementconfig.Spec.NamespaceSelector)
		if err != nil {
			log.Error(err, "Unable to marshal namespaceselector")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		op := fmt.Sprintf(replaceWebhooksValueTemplate, string(nsselectorbytes))
		_, err = r.Clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(ctx, "multiarch-operator-mutating-webhook-configuration", types.JSONPatchType, generatePatchBytes(op), metav1.PatchOptions{})
		if err != nil {
			if err != nil {
				log.Error(err, "Unable to update mutatingwebhookconfiguration")
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
	}
	err = r.Client.Update(ctx, podplacementconfig)
	if err != nil {
		log.Error(err, "Unable to update the podplacementconfig")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodPlacementConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiarchv1alpha1.PodPlacementConfig{}).
		Complete(r)
}
