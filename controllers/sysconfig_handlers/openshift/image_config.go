package openshift

import (
	"context"
	ocpv1 "github.com/openshift/api/config/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"multiarch-operator/pkg/system_config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ImageRegistryConfigSyncer watches the image.config.openshift.io/cluster object and updates the registry configuration accordingly by using
// the SystemConfigSyncer.
// The configuration written by the SystemConfigSyncer due to the ImageRegistryConfigSyncer is stored in-memory in the
// SystemConfigSyncer.registryConfContent (type registryConf) and written to disk in the $conf_dir/containers/registries.conf,
// and $conf_dir/containers/polices.json files.
// In particular, an example of the configuration written by the SystemConfigSyncer due to the ImageRegistryConfigSyncer in $conf_dir/containers/registries.conf is:
// [[registries]]
//
//	location = "registry.redhat.io"
//	allowed = true
//
// An example of the configuration written by the SystemConfigSyncer due to the ImageRegistryConfigSyncer in $conf_dir/containers/policies.json is:
// {
//   "default": [
//     {
//       "type": "insecureAcceptAnything"
//     }
//   ],
//   "transports": {
//     "atomic": {
//       "docker.io": [
//         {
//           "type": "reject"
//         }
//       ]
//     },
//     "docker": {
//       "docker.io": [
//         {
//           "type": "reject"
//         }
//       ]
//     },
//     "docker-daemon": {
//       "": [
//         {
//           "type": "insecureAcceptAnything"
//         }
//       ]
//     }
//   }
// }
//

type ImageRegistryConfigSyncer struct {
	mgr manager.Manager
}

func NewImageRegistryConfigSyncer(mgr manager.Manager) *ImageRegistryConfigSyncer {
	return &ImageRegistryConfigSyncer{
		mgr: mgr,
	}
}

func (s *ImageRegistryConfigSyncer) Start(ctx context.Context) (err error) {
	klog.Warningf("starting the Openshift Image Registry Config syncer")
	mgr := s.mgr
	ic := system_config.SystemConfigSyncerSingleton()
	imageInformer, err := mgr.GetCache().GetInformerForKind(ctx, ocpv1.GroupVersion.WithKind("Image"))
	if err != nil {
		klog.Errorf("error getting informer for Image [config.openshift.io/v1]: %w", err)
		return err
	}
	_, err = imageInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd(ic),
		UpdateFunc: s.onUpdate(ic),
	})
	if err != nil {
		klog.Errorf("error registering handler for Image [config.openshift.io/v1]: %w", err)
		return err
	}

	return nil
}

func (s *ImageRegistryConfigSyncer) onAddOrUpdate(ic system_config.IConfigSyncer, obj interface{}) {
	image, ok := obj.(*ocpv1.Image)
	if !ok {
		klog.Errorf("unexpected type %T, expected Image", obj)
		return
	}
	if image.Name != "cluster" {
		klog.Warningf("ignoring image.config.openshift.io/%s object", image.Name)
		return
	}
	klog.Warningln("the image.config.openshift.io/cluster object has been updated.")
	err := ic.StoreImageRegistryConf(image.Spec.RegistrySources.AllowedRegistries,
		image.Spec.RegistrySources.BlockedRegistries, image.Spec.RegistrySources.InsecureRegistries)
	if err != nil {
		klog.Warningf("error updating registry conf: %w", err)
		return
	}
}

func (s *ImageRegistryConfigSyncer) onAdd(ic system_config.IConfigSyncer) func(interface{}) {
	return func(obj interface{}) {
		s.onAddOrUpdate(ic, obj)
	}
}

func (s *ImageRegistryConfigSyncer) onUpdate(ic system_config.IConfigSyncer) func(interface{}, interface{}) {
	return func(oldobj, newobj interface{}) {
		s.onAddOrUpdate(ic, newobj)
	}
}
