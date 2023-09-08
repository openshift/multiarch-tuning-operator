package common

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	clientv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"multiarch-operator/pkg/image"
	"multiarch-operator/pkg/utils"
)

type GlobalPullSecretSyncer struct {
	clientSet *kubernetes.Clientset
	namespace string
	name      string
}

func NewGlobalPullSecretSyncer(clientSet *kubernetes.Clientset, namespace, name string) *GlobalPullSecretSyncer {
	return &GlobalPullSecretSyncer{
		clientSet: clientSet,
		namespace: namespace,
		name:      name,
	}
}

func (s *GlobalPullSecretSyncer) Start(ctx context.Context) (err error) {
	klog.Warningf("starting the Openshift global pull-secret syncer")
	clientSet := s.clientSet
	// Watch the Secret that contains the global pull secret and Sync the inspector
	globalPullSecretInformer := clientv1.NewSecretInformer(clientSet, s.namespace, 0, cache.Indexers{})

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
		klog.Errorf("unexpected type %T, expected ConfigMap", obj)
		return
	}
	if secret.Name != s.name {
		// Ignore other configmaps
		return
	}
	klog.Infof("The global pull secret %s/%s was updated", s.namespace, s.name)
	if pullSecret, err := utils.ExtractAuthFromSecret(secret); err == nil {
		image.FacadeSingleton().StoreGlobalPullSecret(pullSecret)
	} else {
		klog.Warningf("Error extracting the auth from the secret %s/%s: %v", s.name, s.namespace, err)
	}
}

func (s *GlobalPullSecretSyncer) onUpdate() func(oldobj, newobj interface{}) {
	return func(oldobj, newobj interface{}) {
		s.onAddOrUpdate(newobj)
	}
}
