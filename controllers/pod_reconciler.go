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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"multiarch-operator/pkg/image"
	"multiarch-operator/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
// Reconcile has to watch the pod object if it has the scheduling gate with name schedulingGateName,
// inspect the images in the pod spec, update the nodeAffinity accordingly and remove the scheduling gate.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx, "namespace", req.Namespace, "name", req.Name, "action", "Reconcile")

	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		log.V(3).Error(err, "Unable to fetch pod")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// verify whether the pod is in the proper phase to add a schedulingGate

	// verify whether the pod has the scheduling gate
	if !hasSchedulingGate(pod) {
		log.V(4).Info("Pod does not have the scheduling gate. Ignoring...")
		// if not, return
		return ctrl.Result{}, nil
	}

	log.V(4).Info("Processing pod", "namespace")
	// The scheduling gate is found.
	var err error

	// Prepare the requirement for the node affinity.
	architectureRequirement, err := prepareRequirement(ctx, r.Clientset, pod)
	if err != nil {
		log.Error(err, "Unable to get the architecture requirements for the pod. "+
			"The nodeAffinity for this pod will not be set.")
		// we still need to remove the scheduling gate. Therefore, we do not return here.
	} else {
		// Update the node affinity
		setPodNodeAffinityRequirement(ctx, pod, architectureRequirement)
	}

	// Remove the scheduling gate
	log.V(4).Info("Removing the scheduling gate from pod.")
	removeSchedulingGate(pod)

	err = r.Client.Update(ctx, pod)
	if err != nil {
		log.Error(err, "Unable to update the pod", "namespace")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func prepareRequirement(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) (corev1.NodeSelectorRequirement, error) {
	values, err := inspectImages(ctx, clientset, pod)
	// if an error occurs, we return an empty NodeSelectorRequirement and the error.
	if err != nil {
		return corev1.NodeSelectorRequirement{}, err
	}
	return corev1.NodeSelectorRequirement{
		Key:      "kubernetes.io/arch",
		Operator: corev1.NodeSelectorOpIn,
		Values:   values,
	}, nil
}

// setPodNodeAffinityRequirement sets the node affinity for the pod to the given requirement based on the rules in
// the sig-scheduling's KEP-3838: https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/3838-pod-mutable-scheduling-directives.
func setPodNodeAffinityRequirement(ctx context.Context, pod *corev1.Pod,
	requirement corev1.NodeSelectorRequirement) {
	log := ctrllog.FromContext(ctx, "namespace", pod.Namespace, "name", pod.Name, "action", "setPodNodeAffinityRequirement")
	// We are ignoring the podSpec.nodeSelector field,
	// TODO: validate this is ok when a pod has both nodeSelector and (our) nodeAffinity
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}

	// the .requiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms are ORed
	if len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
		// We create a new array of NodeSelectorTerm of length one so that we can always iterate it in the next.
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = make([]corev1.NodeSelectorTerm, 1)
	}
	nodeSelectorTerms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	// The expressions within the nodeSelectorTerms are ANDed.
	// Therefore, we iterate over the nodeSelectorTerms and add an expression to each of the terms to verify the
	// kubernetes.io/arch label has compatible values.
	// Note that the NodeSelectorTerms will always be long at least 1, because we (re-)created it with size 1 above if it was nil (or having 0 length).
	var skipMatchExpressionPatch bool
	for i := range nodeSelectorTerms {
		skipMatchExpressionPatch = false
		if nodeSelectorTerms[i].MatchExpressions == nil {
			nodeSelectorTerms[i].MatchExpressions = make([]corev1.NodeSelectorRequirement, 0, 1)
		}
		// Check if the nodeSelectorTerm already has a matchExpression for the kubernetes.io/arch label.
		// if yes, we ignore to add it.
		for _, expression := range nodeSelectorTerms[i].MatchExpressions {
			if expression.Key == requirement.Key {
				log.V(4).Info("The current nodeSelectorTerm already has a matchExpression for the kubernetes.io/arch label. Ignoring...")
				skipMatchExpressionPatch = true
				break
			}
		}
		// if skipMatchExpressionPatch is true, we skip to add the matchExpression so that conflictual matchExpressions provided by the user are not overwritten.
		if !skipMatchExpressionPatch {
			nodeSelectorTerms[i].MatchExpressions = append(nodeSelectorTerms[i].MatchExpressions, requirement)
		}
	}
}

func getPodImagePullSecrets(pod *corev1.Pod) []string {
	if pod.Spec.ImagePullSecrets == nil {
		// If the imagePullSecrets array is nil, return emptylist
		return []string{}
	}
	secretRefs := []string{}
	for _, secret := range pod.Spec.ImagePullSecrets {
		secretRefs = append(secretRefs, secret.Name)
	}
	return secretRefs
}

func hasSchedulingGate(pod *corev1.Pod) bool {
	if pod.Spec.SchedulingGates == nil {
		// If the schedulingGates array is nil, we return false
		return false
	}
	for _, condition := range pod.Spec.SchedulingGates {
		if condition.Name == schedulingGateName {
			return true
		}
	}
	// the scheduling gate is not found.
	return false
}

func removeSchedulingGate(pod *corev1.Pod) {
	if len(pod.Spec.SchedulingGates) == 0 {
		// If the schedulingGates array is nil, we return
		return
	}
	filtered := make([]corev1.PodSchedulingGate, 0, len(pod.Spec.SchedulingGates))
	for _, schedulingGate := range pod.Spec.SchedulingGates {
		if schedulingGate.Name != schedulingGateName {
			filtered = append(filtered, schedulingGate)
		}
	}
	pod.Spec.SchedulingGates = filtered
}

// inspectImages returns the list of supported architectures for the images used by the pod.
// if an error occurs, it returns the error and a nil slice of strings.
func inspectImages(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) (supportedArchitectures []string, err error) {
	log := ctrllog.FromContext(ctx, "namespace", pod.Namespace, "name", pod.Name, "action", "inspectImages")
	// Build a set of all the images used by the pod
	imageNamesSet := sets.New[string]()
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		imageNamesSet.Insert(fmt.Sprintf("//%s", container.Image))
	}
	log.V(3).Info("Images list for pod", "imageNamesSet", fmt.Sprintf("%+v", imageNamesSet))

	// https://github.com/containers/skopeo/blob/v1.11.1/cmd/skopeo/inspect.go#L72
	// Iterate over the images, get their architectures and intersect (as in set intersection) them each other
	var supportedArchitecturesSet sets.Set[string]
	for imageName := range imageNamesSet {
		secretAuths, err := pullSecretAuthList(ctx, clientset, pod)
		if err != nil {
			log.Error(err, "Error consolidating pull secrets for pod")
			return nil, err
		}
		log.V(5).Info("Checking image", "imageName", imageName)
		currentImageSupportedArchitectures, err := image.FacadeSingleton().GetCompatibleArchitecturesSet(ctx, imageName, secretAuths)
		if err != nil {
			// The image cannot be inspected, we skip from adding the nodeAffinity
			log.V(3).Error(err, "Error inspecting the image", "imageName", imageName)
			return nil, err
		}
		if supportedArchitecturesSet == nil {
			supportedArchitecturesSet = currentImageSupportedArchitectures
		} else {
			supportedArchitecturesSet = supportedArchitecturesSet.Intersection(currentImageSupportedArchitectures)
		}
	}
	return sets.List(supportedArchitecturesSet), nil
}

// pullSecretAuthList returns the list of secrets data for the given pod given its imagePullSecrets field
func pullSecretAuthList(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) ([][]byte, error) {
	log := ctrllog.FromContext(ctx, "namespace", pod.Namespace, "name", pod.Name, "action", "pullSecretAuthList")
	secretAuths := make([][]byte, 0)
	secretList := getPodImagePullSecrets(pod)
	for _, pullsecret := range secretList {
		secret, err := clientset.CoreV1().Secrets(pod.Namespace).Get(ctx, pullsecret, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "Error getting secret", "secret", pullsecret)
			continue
		}
		if secretData, err := utils.ExtractAuthFromSecret(secret); err != nil {
			log.Error(err, "Error extracting auth from secret", "secret", pullsecret)
			continue
		} else {
			secretAuths = append(secretAuths, secretData)
		}
	}
	return secretAuths, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
