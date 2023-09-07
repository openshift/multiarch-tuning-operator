package openshift

import (
	"context"
	"encoding/json"
	"errors"
	corev1 "k8s.io/api/core/v1"
	clientv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"multiarch-operator/pkg/image"
)

type GlobalPullSecretSyncer struct {
	clientSet *kubernetes.Clientset
	observers []IGlobalPullSecretObserver
}

type IGlobalPullSecretObserver interface {
	// Update notifies the observer that the global pull secret has been updated
	Update(pullSecret []byte)
}

func NewGlobalPullSecretSyncer(clientSet *kubernetes.Clientset) *GlobalPullSecretSyncer {
	return &GlobalPullSecretSyncer{
		clientSet: clientSet,
	}
}

func (s *GlobalPullSecretSyncer) Start(ctx context.Context) (err error) {
	klog.Warningf("starting the Openshift global pull-secret syncer")
	clientSet := s.clientSet
	// Watch the Secret that contains the global pull secret and Sync the inspector
	globalPullSecretInformer := clientv1.NewSecretInformer(clientSet, "openshift-config", 0, cache.Indexers{})

	_, err = globalPullSecretInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    s.onAddOrUpdate,
			UpdateFunc: s.onUpdate(),
		},
	)
	if err != nil {
		klog.Errorf("error registering handler for the global pull-secret configmap: %w", err)
		return err
	}

	globalPullSecretInformer.Run(ctx.Done())

	return nil
}

func (s *GlobalPullSecretSyncer) onAddOrUpdate(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		// TODO[informers]: should we panic here?
		klog.Errorf("unexpected type %T, expected ConfigMap", obj)
		return
	}
	if secret.Name != "pull-secret" {
		// Ignore other configmaps
		return
	}
	klog.Warningln("global pull secret update")
	if pullSecret, err := ExtractAuthFromSecret(secret); err == nil {
		image.FacadeSingleton().StoreGlobalPullSecret(pullSecret)
	} else {
		klog.Warningf("Error extracting the auth from the secret: %v", err)
	}
}

func (s *GlobalPullSecretSyncer) onUpdate() func(oldobj, newobj interface{}) {
	return func(oldobj, newobj interface{}) {
		s.onAddOrUpdate(newobj)
	}
}

func ExtractAuthFromSecret(secret *corev1.Secret) ([]byte, error) {
	switch secret.Type {
	case "kubernetes.io/dockercfg":
		return secret.Data[".dockercfg"], nil
	case "kubernetes.io/dockerconfigjson":
		var objmap map[string]json.RawMessage
		if err := json.Unmarshal(secret.Data[".dockerconfigjson"], &objmap); err != nil {
			klog.Warningf("Error unmarshaling secret data for: %s/%s", secret.Namespace, secret.Name)
			return nil, err
		}
		return objmap["auths"], nil
	}
	return nil, errors.New("unknown secret type")
}
