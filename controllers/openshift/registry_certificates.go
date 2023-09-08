package openshift

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"multiarch-operator/pkg/system_config"
)

// RegistryCertificatesSyncer watches a configmap (openshift-image-registry/image-registry-certificates) and updates
// the registry certificates accordingly by using the SystemConfigSyncer.
// The configuration written by the SystemConfigSyncer due to the RegistryCertificatesSyncer is stored in-memory in the
// SystemConfigSyncer.registryCertTuples (type []system_config.registryCertTuple) and written to disk in the $conf_dir/docker/certs.d directory.
// In particular, an example of the configuration written by the SystemConfigSyncer due to the RegistryCertificatesSyncer in $conf_dir/docker/certs.d is:
// $conf_dir/docker/certs.d/registry.redhat.io/ca.crt
// $conf_dir/docker/certs.d/registry.redhat.io:5000/ca.crt
type RegistryCertificatesSyncer struct {
	// clientSet is the kubernetes clientset
	clientSet *kubernetes.Clientset
	// namespace is the namespace where the configmap that contains the registry certificates is stored
	namespace string
	// name is the name of the configmap that contains the registry certificates
	name string
}

func NewRegistryCertificatesSyncer(clientSet *kubernetes.Clientset, namespace, name string) *RegistryCertificatesSyncer {
	return &RegistryCertificatesSyncer{
		clientSet: clientSet,
		namespace: namespace,
		name:      name,
	}
}

func (s *RegistryCertificatesSyncer) Start(ctx context.Context) (err error) {
	klog.Warningf("starting the Openshift registry certificates syncer")
	ic := system_config.SystemConfigSyncerSingleton()
	clientSet := s.clientSet
	// Watch the ConfigMap that contains the registry certificates and Sync SystemConfig
	registryCertificatesInformer := v1.NewConfigMapInformer(clientSet, s.namespace, 0, cache.Indexers{})

	_, err = registryCertificatesInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    s.onAdd(ic),
			UpdateFunc: s.onUpdate(ic),
		},
	)

	if err != nil {
		klog.Errorf("error registering handler for the image-registry-certificates configmap: %w", err)
		return err
	}

	registryCertificatesInformer.Run(ctx.Done())

	return nil
}

func (s *RegistryCertificatesSyncer) onAddOrUpdate(ic system_config.IConfigSyncer, obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		// TODO[informers]: should we panic here?
		klog.Errorf("unexpected type %T, expected ConfigMap", obj)
		return
	}
	if cm.Name != s.name {
		// Ignore other configmaps
		return
	}
	klog.Infof("the %s/%s configmap has been updated", s.namespace, s.name)
	err := ic.StoreRegistryCerts(system_config.ParseRegistryCerts(cm.Data))
	if err != nil {
		klog.Warningf("error updating registry certs: %w", err)
		return
	}
}

func (s *RegistryCertificatesSyncer) onUpdate(ic system_config.IConfigSyncer) func(oldobj, newobj interface{}) {
	return func(oldobj, newobj interface{}) {
		s.onAddOrUpdate(ic, newobj)
	}
}

func (s *RegistryCertificatesSyncer) onAdd(ic system_config.IConfigSyncer) func(obj interface{}) {
	return func(obj interface{}) {
		s.onAddOrUpdate(ic, obj)
	}
}
