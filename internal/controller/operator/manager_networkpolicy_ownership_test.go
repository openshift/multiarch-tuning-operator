package operator

import (
	"context"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

func TestManagerNetworkPolicyOwnership(t *testing.T) {
	owner := metav1.OwnerReference{APIVersion: "apps/v1", Kind: "Deployment", Name: utils.OperatorName + "-controller-manager", UID: "current", Controller: utils.NewPtr(true), BlockOwnerDeletion: utils.NewPtr(true)}
	for _, tt := range []struct {
		name     string
		missing  bool
		change   func(*metav1.OwnerReference)
		conflict bool
	}{
		{name: "missing", missing: true},
		{name: "unowned", change: func(o *metav1.OwnerReference) { o.Controller = nil }},
		{name: "current"},
		{name: "stale UID", change: func(o *metav1.OwnerReference) { o.UID = "stale" }},
		{name: "unrelated Deployment", change: func(o *metav1.OwnerReference) { o.Name = "other" }, conflict: true},
		{name: "unrelated kind", change: func(o *metav1.OwnerReference) { o.Kind = "StatefulSet" }, conflict: true},
		{name: "unrelated API version", change: func(o *metav1.OwnerReference) { o.APIVersion = "apps/v1beta1" }, conflict: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := appsv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := networkingv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: owner.Name, Namespace: utils.Namespace(), UID: owner.UID}}
			policy := buildNetworkPolicyManager()
			ref := owner
			if tt.change != nil {
				tt.change(&ref)
			}
			policy.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "ConfigMap", Name: "keep", UID: "keep"}, ref}
			policy.Spec.Egress = nil
			policy.Labels = map[string]string{"keep": "on-conflict"}
			c := &managerPolicyTestClient{deployment: deployment}
			if !tt.missing {
				c.policy = policy.DeepCopy()
			}
			before := &networkingv1.NetworkPolicy{}
			key := client.ObjectKeyFromObject(policy)
			if !tt.missing {
				if err := c.Get(context.Background(), key, before); err != nil {
					t.Fatal(err)
				}
			}
			r := &ManagerNetworkPolicyReconciler{Client: c, Scheme: scheme}
			_, err := r.Reconcile(context.Background(), ctrl.Request{})
			if tt.conflict {
				if c.writes != 0 {
					t.Fatalf("conflict caused %d writes", c.writes)
				}
				if err == nil || !strings.Contains(err.Error(), "ownership conflict") {
					t.Fatalf("expected ownership conflict, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			after := &networkingv1.NetworkPolicy{}
			if err := c.Get(context.Background(), key, after); err != nil {
				t.Fatal(err)
			}
			if tt.conflict {
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("conflicting policy changed: before=%+v after=%+v", before, after)
				}
				return
			}
			if !reflect.DeepEqual(after.Spec, buildNetworkPolicyManager().Spec) {
				t.Fatal("policy spec was not reconciled")
			}
			actualOwner := metav1.GetControllerOf(after)
			if !reflect.DeepEqual(actualOwner, &owner) {
				t.Fatalf("controller owner=%+v, want %+v", actualOwner, owner)
			}
			if !tt.missing && !reflect.DeepEqual(after.OwnerReferences[0], before.OwnerReferences[0]) {
				t.Fatal("non-controller owner changed")
			}
		})
	}
}

// Only the operations used by CreateOrUpdate are implemented. Deep copies keep
// mutations in the reconciler separate from the stored API object.
type managerPolicyTestClient struct {
	client.Client
	deployment *appsv1.Deployment
	policy     *networkingv1.NetworkPolicy
	writes     int
}

func (c *managerPolicyTestClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch obj := obj.(type) {
	case *appsv1.Deployment:
		*obj = *c.deployment.DeepCopy()
	case *networkingv1.NetworkPolicy:
		if c.policy == nil {
			return apierrors.NewNotFound(networkingv1.Resource("networkpolicies"), utils.ManagerNetworkPolicyName)
		}
		*obj = *c.policy.DeepCopy()
	default:
		panic("unexpected test client Get")
	}
	return nil
}

func (c *managerPolicyTestClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	c.policy = obj.(*networkingv1.NetworkPolicy).DeepCopy()
	c.writes++
	return nil
}

func (c *managerPolicyTestClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	c.policy = obj.(*networkingv1.NetworkPolicy).DeepCopy()
	c.writes++
	return nil
}

func TestManagerPolicyPredicate(t *testing.T) {
	p := managerPolicyPredicate()
	for _, tt := range []struct {
		name, namespace string
		accept          bool
	}{
		{utils.ManagerNetworkPolicyName, utils.Namespace(), true},
		{"unrelated", utils.Namespace(), false},
		{utils.ManagerNetworkPolicyName, "unrelated", false},
	} {
		np := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tt.name, Namespace: tt.namespace}}
		for kind, accepted := range map[string]bool{
			"create":  p.Create(event.CreateEvent{Object: np}),
			"update":  p.Update(event.UpdateEvent{ObjectOld: np.DeepCopy(), ObjectNew: np}),
			"delete":  p.Delete(event.DeleteEvent{Object: np}),
			"generic": p.Generic(event.GenericEvent{Object: np}),
		} {
			if accepted != tt.accept {
				t.Errorf("%s event %s/%s accepted=%v, want %v", kind, tt.namespace, tt.name, accepted, tt.accept)
			}
		}
	}
}
