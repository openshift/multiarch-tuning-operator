package openshift

import (
	"context"
	ocpv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"multiarch-operator/pkg/system_config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ICSPs report the set of registry sources that the cluster needs to reach via mirrors.
// Each registry source can have multiple mirrors.
// The ICSPSyncer watches ICSPs and updates the registry mirroring config accordingly by using
// the SystemConfigSyncer.
// The configuration written by the SystemConfigSyncer due to the ICSPSyncer is stored in-memory in the
// SystemConfigSyncer.registriesConfContent (type registriesConf) and written to disk in the /$conf_dir/container/registries.conf file.

// In particular, an example of the configuration written by the SystemConfigSyncer due to the ICSPSyncer in /$conf_dir/containers/registries.conf is:
// [[registries]]
//
//	location = "registry.redhat.io"
//	prefix = ""
//	mirror = ["myregistry.example.com"]
//
// [[registries]]
//
//	location = "docker.io"
//	prefix = ""
//	mirror = ["myregistry.example.com"]
//

type ICSPSyncer struct {
	mgr manager.Manager
}

func NewICSPSyncer(mgr manager.Manager) *ICSPSyncer {
	return &ICSPSyncer{
		mgr: mgr,
	}
}

func (s *ICSPSyncer) Start(ctx context.Context) (err error) {
	klog.Warningf("starting the Openshift ImageContentSourcePolicy [operator.openshift.io/v1alpha1] syncer")
	mgr := s.mgr
	ic := system_config.SystemConfigSyncerSingleton()
	// Watch ICSPs and Sync SystemConfig
	icspInformer, err := mgr.GetCache().GetInformerForKind(ctx, ocpv1alpha1.GroupVersion.WithKind("ImageContentSourcePolicy"))
	if err != nil {
		klog.Errorf("error getting informer for ImageContentSourcePolicy: %w", err)
		return err
	}
	_, err = icspInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd(ic),
		UpdateFunc: s.onUpdate(ic),
		DeleteFunc: s.onDelete(ic),
	})
	if err != nil {
		klog.Errorf("error registering handler for ImageContentSourcePolicy [operator.openshift.io/v1alpha1]: %w", err)
		return err
	}
	return nil
}

func (s *ICSPSyncer) onAdd(ic system_config.IConfigSyncer) func(obj interface{}) {
	return func(obj interface{}) {
		icsp, ok := obj.(*ocpv1alpha1.ImageContentSourcePolicy)
		if !ok {
			klog.Errorf("unexpected type %T, expected ImageContentSourcePolicy", obj)
			return
		}
		for _, source := range icsp.Spec.RepositoryDigestMirrors {
			err := ic.UpdateRegistryMirroringConfig(source.Source, source.Mirrors)
			if err != nil {
				klog.Warningf("error updating registry mirroring config %s's source %s : %w",
					icsp.Name, source.Source, err)
				continue
			}
		}
	}
}

func (s *ICSPSyncer) onDelete(ic system_config.IConfigSyncer) func(obj interface{}) {
	return func(obj interface{}) {
		icsp, ok := obj.(*ocpv1alpha1.ImageContentSourcePolicy)
		if !ok {
			klog.Errorf("unexpected type %T, expected ImageContentSourcePolicy", obj)
			return
		}
		for _, source := range icsp.Spec.RepositoryDigestMirrors {
			err := ic.DeleteRegistryMirroringConfig(source.Source)
			if err != nil {
				klog.Warningf("error removing registry mirroring config %s's source %s : %w",
					icsp.Name, source.Source, err)
				continue
			}
		}
	}
}

func (s *ICSPSyncer) onUpdate(ic system_config.IConfigSyncer) func(oldobj, newobj interface{}) {
	return func(oldobj, newobj interface{}) {
		s.onDelete(ic)(oldobj)
		s.onAdd(ic)(newobj)
	}
}
