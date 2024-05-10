package podplacement_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ocpappsv1 "github.com/openshift/api/apps/v1"
	ocpbuildv1 "github.com/openshift/api/build/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1alpha1"
	"github.com/openshift/multiarch-tuning-operator/controllers/operator"
	"github.com/openshift/multiarch-tuning-operator/pkg/e2e"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

var (
	client    runtimeclient.Client
	clientset *kubernetes.Clientset
	ctx       context.Context
	suiteLog  = ctrl.Log.WithName("setup")
)

func init() {
	e2e.CommonInit()
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multiarch Tuning Operator Suite (PodPlacementOperand E2E)", Label("e2e", "pod-placement-operand"))
}

var _ = BeforeSuite(func() {
	client, clientset, ctx, suiteLog = e2e.CommonBeforeSuite()
	err := ocpappsv1.Install(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = ocpbuildv1.Install(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = client.Create(ctx, &v1alpha1.ClusterPodPlacementConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(deploymentsAreRunning).Should(Succeed())

	updateGlobalPullSecret()
})

var _ = AfterSuite(func() {
	err := client.Delete(ctx, &v1alpha1.ClusterPodPlacementConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(deploymentsAreDeleted).Should(Succeed())
})

func deploymentsAreRunning(g Gomega) {
	d, err := clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementControllerName,
		metav1.GetOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.Status.AvailableReplicas).To(Equal(*d.Spec.Replicas),
		"at least one pod placement controller replicas is not available yet")
	d, err = clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementWebhookName,
		metav1.GetOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.Status.AvailableReplicas).To(Equal(*d.Spec.Replicas),
		"at least one pod placement webhook replicas is not available yet")
}

func deploymentsAreDeleted(g Gomega) {
	_, err := clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementControllerName,
		metav1.GetOptions{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
	_, err = clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementWebhookName,
		metav1.GetOptions{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

// updateGlobalPullSecret patches the global pull secret to onboard the
// read-only credentials of the quay.io org. for testing images stored
// in a repo for which credentials are expected to stay in the global pull secret.
// NOTE: TODO: do we need to change the location of the secrets even here for testing non-OCP distributions?
func updateGlobalPullSecret() {
	secret := corev1.Secret{}
	err := client.Get(ctx, runtimeclient.ObjectKeyFromObject(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		},
	}), &secret)
	Expect(err).NotTo(HaveOccurred(), "failed to get secret/pull-secret in namespace openshift-config", err)
	var dockerConfigJSON map[string]interface{}
	err = json.Unmarshal(secret.Data[".dockerconfigjson"], &dockerConfigJSON)
	Expect(err).NotTo(HaveOccurred(), "failed to unmarshal dockerconfigjson", err)
	auths := dockerConfigJSON["auths"].(map[string]interface{})
	// Add new auth for quay.io/multi-arch/tuning-test-global to global pull secret
	registry := "quay.io/multi-arch/tuning-test-global"
	auth := map[string]string{
		"auth": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
			"multi-arch+mto_testing_global_ps", "NELK81COHVFAZHY49MXK9XJ02U7A85V0HY3NS14O4K2AFRN3EY39SH64MFU3U90W"))),
	}
	auths[registry] = auth
	dockerConfigJSON["auths"] = auths
	newDockerConfigJSONBytes, err := json.Marshal(dockerConfigJSON)
	Expect(err).NotTo(HaveOccurred(), "failed to marshal dockerconfigjson", err)
	// Update secret
	secret.Data[".dockerconfigjson"] = []byte(newDockerConfigJSONBytes)
	err = client.Update(ctx, &secret)
	Expect(err).NotTo(HaveOccurred())
}
