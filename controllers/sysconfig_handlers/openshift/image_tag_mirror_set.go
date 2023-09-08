package openshift

import (
	"context"
	v1 "github.com/openshift/api/config/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"multiarch-operator/pkg/system_config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ITMSs report the set of registry sources that the cluster needs to reach via mirrors.
// Each registry source can have multiple mirrors.
// The ICSPSyncer watches ICSPs and updates the registry mirroring config accordingly by using
// the SystemConfigSyncer.
// The configuration written by the SystemConfigSyncer due to the ICSPSyncer is stored in-memory in the
// SystemConfigSyncer.registriesConfContent (type registriesConf) and written to disk in the /$conf_dir/container/registries.conf file.

// In particular, an example of the configuration written by the SystemConfigSyncer due to the ITMSSyncer in /$conf_dir/containers/registries.conf is:
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

type ITMSSyncer struct {
	mgr manager.Manager
}

func NewITMSSyncer(mgr manager.Manager) *ITMSSyncer {
	return &ITMSSyncer{
		mgr: mgr,
	}
}

func (s *ITMSSyncer) Start(ctx context.Context) (err error) {
	klog.Warningf("starting the Openshift ImageTagMirrorSet [config.openshift.io/v1] syncer")
	mgr := s.mgr
	ic := system_config.SystemConfigSyncerSingleton()
	icspInformer, err := mgr.GetCache().GetInformerForKind(ctx, v1.GroupVersion.WithKind("ImageTagMirrorSet"))
	if err != nil {
		klog.Errorf("error getting informer for ImageTagMirrorSet: %w", err)
		return err
	}
	_, err = icspInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd(ic),
		UpdateFunc: s.onUpdate(ic),
		DeleteFunc: s.onDelete(ic),
	})
	if err != nil {
		klog.Errorf("error registering handler for ImageTagMirrorSet [config.openshift.io/v1]: %w", err)
		return err
	}
	return nil
}

func (s *ITMSSyncer) onAdd(ic system_config.IConfigSyncer) func(obj interface{}) {
	return func(obj interface{}) {
		icsp, ok := obj.(*v1.ImageTagMirrorSet)
		if !ok {
			klog.Errorf("unexpected type %T, expected ImageTagMirrorSet ", obj)
			return
		}
		for _, source := range icsp.Spec.ImageTagMirrors {
			err := ic.UpdateRegistryMirroringConfig(source.Source, mirrorsToStrings(source.Mirrors), system_config.PullTypeTagOnly)
			if err != nil {
				klog.Warningf("error updating registry mirroring config %s's source %s : %w",
					icsp.Name, source.Source, err)
				continue
			}
		}
	}
}

func (s *ITMSSyncer) onDelete(ic system_config.IConfigSyncer) func(obj interface{}) {
	return func(obj interface{}) {
		icsp, ok := obj.(*v1.ImageTagMirrorSet)
		if !ok {
			klog.Errorf("unexpected type %T, expected ImageTagMirrorSet", obj)
			return
		}
		for _, source := range icsp.Spec.ImageTagMirrors {
			err := ic.DeleteRegistryMirroringConfig(source.Source)
			if err != nil {
				klog.Warningf("error removing registry mirroring config %s's source %s : %w",
					icsp.Name, source.Source, err)
				continue
			}
		}
	}
}

func (s *ITMSSyncer) onUpdate(ic system_config.IConfigSyncer) func(oldobj, newobj interface{}) {
	return func(oldobj, newobj interface{}) {
		s.onDelete(ic)(oldobj)
		s.onAdd(ic)(newobj)
	}
}
