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
	"encoding/json"
	"fmt"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"golang.org/x/sys/unix"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecretData struct for storing cred structure
type AuthList struct {
	Auths RegAuthList `json:"auths"`
}
type RegAuthData struct {
	Auth string `json:"auth"`
}
type RegAuthList map[string]RegAuthData

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
	architectureRequirement, err := prepareRequirement(ctx, r.Clientset, pod)
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
	// Build a set of all the images used by the pod
	imageNamesSet := sets.New[string]()
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		imageNamesSet.Insert(fmt.Sprintf("//%s", container.Image))
	}
	klog.V(3).Infof("Images list for pod %s/%s: %+v", pod.Namespace, pod.Name, imageNamesSet)
	// https://github.com/containers/skopeo/blob/v1.11.1/cmd/skopeo/inspect.go#L72
	// Iterate over the images, get their architectures and intersect (as in set intersection) them each other
	var supportedArchitecturesSet sets.Set[string]
	for imageName := range imageNamesSet {
		currentImageSupportedArchitectures, err := inspectImage(ctx, clientset, pod, imageName)
		if err != nil {
			// The image cannot be inspected, we skip from adding the nodeAffinity
			klog.Warningf("Error inspecting the image %s: %v", imageName, err)
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

// inspectImage inspects the image and returns the supported architectures. Any error when inspecting the image is returned so that
// the caller can decide what to do.
func inspectImage(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod, imageName string) (supportedArchitectures sets.Set[string], err error) {
	klog.V(5).Infof("Checking %s/%s's image %s", pod.Namespace, pod.Name, imageName)
	// Check if the image is a manifest list
	ref, err := docker.ParseReference(imageName)
	if err != nil {
		klog.Warningf("Error parsing the image reference for the %s/%s's image %s: %v",
			pod.Namespace, pod.Name, imageName, err)
		return nil, err
	}
	// TODO: handle private registries, credentials, TLS verification, etc.
	// Get pull secrets and create a tmp auth file
	secretAuths, err := PullSecretAuthList(ctx, clientset, pod)
	if err != nil {
		klog.Warningf("Error consolidating pull secrets for pod %s ns: %s", pod.Name, pod.Namespace)
		return nil, err
	}
	if len(secretAuths) == 0 {
		klog.Warningf("No pull secrets available for %s ns: %s", pod.Name, pod.Namespace)
		return nil, err
	}
	secretList := AuthList{Auths: secretAuths}
	//Marshal and write auth json file
	authFile, err := WriteAuthFile(&secretList, pod.Name+"-"+pod.Namespace)
	if err != nil {
		klog.Warningf("Couldn't write auth file for %s ns: %s %v", pod.Name, pod.Namespace, err)
		return nil, err
	} else {
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				klog.Warningf("Failed to close auth file %s %v", f.Name(), err)
			}
		}(authFile)
	}

	src, err := ref.NewImageSource(ctx, &types.SystemContext{
		AuthFilePath: authFile.Name(),
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
		supportedArchitectures = sets.New[string]()
		for _, m := range index.Manifests {
			supportedArchitectures = sets.Insert(supportedArchitectures, m.Platform.Architecture)
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
		return sets.New(config.Architecture), nil
	}
}

// Function consolidates image pull secrets
// - ImagePullSecrets in pod
// - global pull secret in openshift-config
// TODO? - default pull secret in pod namespace if no ImagePullSecrets field?
func PullSecretAuthList(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) (RegAuthList, error) {
	secretAuths := make(RegAuthList)
	secretList := getPodImagePullSecrets(pod)
	for _, pullsecret := range secretList {
		secret, err := clientset.CoreV1().Secrets(pod.Namespace).Get(ctx, pullsecret, metav1.GetOptions{})
		if err != nil {
			klog.Warningf("Error getting secret: %s namespace: %s", pullsecret, pod.Namespace)
			return nil, err
		}
		var tmpAuths RegAuthList
		if secret.Type == "kubernetes.io/dockercfg" {
			err := json.Unmarshal(secret.Data[".dockercfg"], &tmpAuths)
			if err != nil {
				klog.Warningf("Error unmarshaling secret data for: %v", pullsecret)
				return nil, err
			}
		} else if secret.Type == "kubernetes.io/dockerconfigjson" {
			var objmap map[string]json.RawMessage
			err := json.Unmarshal(secret.Data[".dockerconfigjson"], &objmap)
			if err != nil {
				klog.Warningf("Error unmarshaling secret data for: %v", pullsecret)
				return nil, err
			}
			err = json.Unmarshal(objmap["auths"], &tmpAuths)
			if err != nil {
				klog.Warningf("Error unmarshaling secret data for: %v", pullsecret)
				return nil, err
			}
		} else {
			klog.Warningf("Error getting secret data for: %v", pullsecret)
			return nil, err
		}
		// NOTE: Keys are overwritten with the latest in this case.
		// TODO: decide how to handle dup keys
		for k, v := range tmpAuths {
			secretAuths[k] = v
		}
	}
	//merge global pull secret
	secret, err := clientset.CoreV1().Secrets("openshift-config").Get(ctx, "pull-secret", metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Error getting global pull secret")
		return nil, err
	}
	var objmap map[string]json.RawMessage
	var tmpAuths RegAuthList
	err = json.Unmarshal(secret.Data[".dockerconfigjson"], &objmap)
	if err != nil {
		klog.Warningf("Error unmarshaling secret data for the global pull secret")
		return nil, err
	}
	err = json.Unmarshal(objmap["auths"], &tmpAuths)
	if err != nil {
		klog.Warningf("Error unmarshaling secret data for the global pull secret")
		return nil, err
	}
	// NOTE: Keys are overwritten with the latest in this case.
	// TODO: decide how to handle dup keys
	for k, v := range tmpAuths {
		secretAuths[k] = v
	}
	return secretAuths, nil
}

// Write auth json file to pass to c/Image API
func WriteAuthFile(authList *AuthList, fileName string) (*os.File, error) {
	authJson, err := json.Marshal(*authList)
	if err != nil {
		klog.Warningf("Error marshalling pull secrets")
		return nil, err
	}
	fd, err := writeMemFile(fileName, authJson)
	if err != nil {
		return nil, err
	}
	// filepath to our newly created in-memory file descriptor
	fp := fmt.Sprintf("/proc/self/fd/%d", fd)
	tmpFile := os.NewFile(uintptr(fd), fp)
	return tmpFile, err
}

// writeMemFile creates an in memory file based on memfd_create
// returns a file descriptor. Once all references to the file are
// dropped it is automatically released. It is up to the caller
// to close the returned descriptor.
func writeMemFile(name string, b []byte) (int, error) {
	fd, err := unix.MemfdCreate(name, 0)
	if err != nil {
		return 0, fmt.Errorf("MemfdCreate: %v", err)
	}
	err = unix.Ftruncate(fd, int64(len(b)))
	if err != nil {
		return 0, fmt.Errorf("Ftruncate: %v", err)
	}
	data, err := unix.Mmap(fd, 0, len(b), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return 0, fmt.Errorf("Mmap: %v", err)
	}
	copy(data, b)
	err = unix.Munmap(data)
	if err != nil {
		return 0, fmt.Errorf("Munmap: %v", err)
	}
	return fd, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
