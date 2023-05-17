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
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// There is not Set implementation in Golang. The void struct is thought to emulate it via maps.
type void struct{}

var voidVal void

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch

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
	_ = log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		klog.V(3).Infof("unable to fetch Pod %s/%s: %v", req.Namespace, req.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// verify whether the pod is in the proper phase to add a schedulingGate

	// verify whether the pod has the scheduling gate
	if !hasSchedulingGate(pod) {
		klog.V(4).Infof("pod %s/%s does not have the scheduling gate. Ignoring...", pod.Namespace, pod.Name)
		// if not, return
		return ctrl.Result{}, nil
	}

	klog.V(4).Infof("Processing pod %s/%s", pod.Namespace, pod.Name)
	// The scheduling gate is found.
	var err error

	// Prepare the requirement for the node affinity.
	architectureRequirement, err := prepareRequirement(ctx, pod)
	if err != nil {
		klog.Errorf("unable to get the architecture requirements for pod %s/%s: %v. "+
			"The nodeAffinity for this pod will not be set.", pod.Namespace, pod.Name, err)
		// we still need to remove the scheduling gate. Therefore, we do not return here.
	} else {
		// Update the node affinity
		setPodNodeAffinityRequirement(ctx, pod, architectureRequirement)
	}

	// Remove the scheduling gate
	klog.V(4).Infof("Removing the scheduling gate from pod %s/%s", pod.Namespace, pod.Name)
	removeSchedulingGate(pod)

	err = r.Client.Update(ctx, pod)
	if err != nil {
		klog.Errorf("unable to update the pod %s/%s: %v", pod.Namespace, pod.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func prepareRequirement(ctx context.Context, pod *corev1.Pod) (corev1.NodeSelectorRequirement, error) {
	values, err := inspectImages(ctx, pod)
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
				klog.V(4).Infof("the current nodeSelectorTerm already has a matchExpression for the kubernetes.io/arch label. Ignoring...")
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
func inspectImages(ctx context.Context, pod *corev1.Pod) (supportedArchitectures []string, err error) {
	// Build a set of all the images used by the pod
	imageNamesSet := map[string]void{}
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		imageNamesSet[fmt.Sprintf("//%s", container.Image)] = voidVal
	}
	klog.V(3).Infof("Images list for pod %s/%s: %+v", pod.Namespace, pod.Name, imageNamesSet)
	// https://github.com/containers/skopeo/blob/v1.11.1/cmd/skopeo/inspect.go#L72
	// Iterate over the images, get their architectures and intersect (as in set intersection) them each other
	supportedArchitecturesPerImage := make([][]string, 0, len(imageNamesSet))
	for imageName := range imageNamesSet {
		currentImageSupportedArchitectures, err := inspectImage(ctx, pod, imageName)
		if err != nil {
			// The image cannot be inspected, we skip from adding the nodeAffinity
			klog.Warningf("Error inspecting the image %s: %v", imageName, err)
			return nil, err
		}
		supportedArchitecturesPerImage = append(supportedArchitecturesPerImage, currentImageSupportedArchitectures)
	}

	// Intersect the supported architectures of each image once the inspection of all the images is done
	supportedArchitectures = intersect(supportedArchitecturesPerImage...)
	return
}

// inspectImage inspects the image and returns the supported architectures. Any error when inspecting the image is returned so that
// the caller can decide what to do.
func inspectImage(ctx context.Context, pod *corev1.Pod, imageName string) (supportedArchitectures []string, err error) {
	klog.V(5).Infof("Checking %s/%s's image %s", pod.Namespace, pod.Name, imageName)
	// Check if the image is a manifest list
	ref, err := docker.ParseReference(imageName)
	if err != nil {
		klog.Warningf("Error parsing the image reference for the %s/%s's image %s: %v",
			pod.Namespace, pod.Name, imageName, err)
		return nil, err
	}

	// TODO: handle private registries, credentials, TLS verification, etc.
	// For OCP, we also need to handle the access to the internal registry via the proper RBAC rule.
	src, err := ref.NewImageSource(ctx, &types.SystemContext{
		OCIInsecureSkipTLSVerify:    true,
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	})
	if err != nil {
		klog.Warningf("Error creating the image source: %v", err)
		return nil, err
	}
	defer func(src types.ImageSource) {
		err := src.Close()
		if err != nil {
			klog.Warningf("Error closing the image source for the %s/%s's image %s: %v",
				pod.Namespace, pod.Name, imageName, err)
		}
	}(src)

	rawManifest, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		klog.Infof("Error getting the image manifest: %v", err)
		return nil, err
	}
	if manifest.MIMETypeIsMultiImage(manifest.GuessMIMEType(rawManifest)) {
		klog.V(5).Infof("%s/%s's image %s is a manifest list... getting the list of supported architectures",
			pod.Namespace, pod.Name, imageName)
		// The image is a manifest list
		index, err := manifest.OCI1IndexFromManifest(rawManifest)
		if err != nil {
			klog.Warningf("Error parsing the OCI index from the raw manifest of the %s/%s's image %s: %v",
				pod.Namespace, pod.Name, imageName, err)
		}
		supportedArchitectures = make([]string, 0, len(index.Manifests))
		for _, m := range index.Manifests {
			supportedArchitectures = append(supportedArchitectures, m.Platform.Architecture)
		}
		return supportedArchitectures, nil

	} else {
		klog.V(5).Infof("%s/%s's image %s is not a manifest list... getting the supported architecture",
			pod.Namespace, pod.Name, imageName)
		sys := &types.SystemContext{}
		parsedImage, err := image.FromUnparsedImage(ctx, sys, image.UnparsedInstance(src, nil))
		if err != nil {
			klog.Warningf("Error parsing the manifest of the %s/%s's image %s: %v",
				pod.Namespace, pod.Name, imageName, err)
			return nil, err
		}
		config, err := parsedImage.OCIConfig(ctx)
		if err != nil {
			// Ignore errors due to invalid images at this stage
			klog.Warningf("Error parsing the OCI config of the %s/%s's image %s: %v",
				pod.Namespace, pod.Name, imageName, err)
			return nil, err
		}
		return []string{config.Architecture}, nil
	}
}

func intersect[T comparable](sets ...[]T) []T {
	setMap := make(map[T]int) // Map to track element occurrences

	// Count occurrences of elements in all sets
	for _, set := range sets {
		visited := make(map[T]void) // Track visited elements in the current set

		for _, element := range set {
			// Skip if the element has already been visited in the current set
			if _, ok := visited[element]; ok {
				continue
			}
			setMap[element]++
		}
	}

	intersection := make([]T, 0, len(setMap))
	setCount := len(sets)

	// Check for elements that occurred in all sets, considering duplicates
	for element, occurrences := range setMap {
		if occurrences == setCount {
			intersection = append(intersection, element)
		}
	}

	return intersection
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
