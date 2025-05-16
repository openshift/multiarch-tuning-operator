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

package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var enoexeceventlog = logf.Log.WithName("enoexecevent-resource")

func (r *ENoExecEvent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-multiarch-openshift-io-v1beta1-enoexecevent,mutating=false,failurePolicy=fail,sideEffects=None,groups=multiarch.openshift.io,resources=enoexecevents,verbs=create;update,versions=v1beta1,name=venoexecevent.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ENoExecEventValidator{}

type ENoExecEventValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ENoExecEventValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.validate(ctx)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ENoExecEventValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.validate(ctx)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ENoExecEventValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	enoexeceventlog.Info("validate delete", "name")
	return nil, nil
}

func (r *ENoExecEventValidator) validate(ctx context.Context) (warnings admission.Warnings, err error) {
	req, ok := admission.RequestFromContext(ctx)
	if ok != nil {
		enoexeceventlog.Error(nil, "unable to get admission request from context")
		return nil, fmt.Errorf("could not retrieve request info from context")
	}

	user := req.UserInfo.Username
	expectedSA := "system:serviceaccount:your-namespace:multiarch-tuning-operator-controller-manager"

	if user != expectedSA {
		enoexeceventlog.Info("creation denied: invalid user", "user", user)
		return nil, fmt.Errorf("only the operator service account is allowed to create this resource")
	}

	enoexeceventlog.Info("creation allowed", "user", user)
	return nil, nil
}
