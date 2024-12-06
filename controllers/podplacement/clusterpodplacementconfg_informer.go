package podplacement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	multiarchv1beta1 "github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
	"github.com/openshift/multiarch-tuning-operator/pkg/informers"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// CPPCSyncer syncs ClusterPodPlacementConfig resources using an informer.
type CPPCSyncer struct {
	clientSet *kubernetes.Clientset
	mgr       manager.Manager
	log       logr.Logger
}

// NewCPPCSyncer creates a new CPPCSyncer.
func NewCPPCSyncer(mgr manager.Manager, client *kubernetes.Clientset) *CPPCSyncer {
	return &CPPCSyncer{
		clientSet: client,
		mgr:       mgr,
	}
}

// Start initializes the CPPC informer and starts syncing.
func (s *CPPCSyncer) Start(ctx context.Context) error {
	s.log = log.FromContext(ctx, "handler", "CPPCSyncer")
	s.log.Info("Starting CPPC Syncer")
	mgr := s.mgr

	ic := informers.CacheSingleton()
	// Get informer for ClusterPodPlacementConfig
	CPPCInformer, err := mgr.GetCache().GetInformerForKind(ctx, multiarchv1beta1.GroupVersion.WithKind("ClusterPodPlacementConfig"))
	if err != nil {
		s.log.Error(err, "Error getting informer for ClusterPodPlacementConfig")
		return err
	}

	// Register event handlers
	_, err = CPPCInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd(ic),
		UpdateFunc: s.onUpdate(ic),
		DeleteFunc: s.onDelete(ic),
	})
	if err != nil {
		s.log.Error(err, "Error registering handler for ClusterPodPlacementConfig")
		return err
	}

	return nil
}

// onAdd handles the addition of a ClusterPodPlacementConfig.
func (s *CPPCSyncer) onAdd(ic informers.ICache) func(obj interface{}) {
	return func(obj interface{}) {
		CPPC, ok := obj.(*multiarchv1beta1.ClusterPodPlacementConfig)
		if !ok {
			s.log.Error(errors.New("unexpected type, expected ClusterPodPlacementConfig"), "unexpected type",
				"type", fmt.Sprintf("%T", obj))
			return
		}

		jsonCPPC, err := json.Marshal(CPPC)
		if err != nil {
			s.log.Error(err, "Error marshalling ClusterPodPlacementConfig")
		}
		s.log.Info("%%%%%%%%%%% onAdd")

		jsonWebhook := json.RawMessage{}
		// Get the MutatingWebhookConfiguration by name
		mutatingWebhookConfig, err := s.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), utils.PodMutatingWebhookConfigurationName, metav1.GetOptions{})
		if client.IgnoreNotFound(err) != nil {
			s.log.Error(err, "Unable to get the mutating webhook configuration")
		} else {
			jsonWebhook, err = json.Marshal(mutatingWebhookConfig)
			if err != nil {
				s.log.Error(err, "Error marshalling ClusterPodPlacementConfig Webhook")
			}
		}

		err = ic.StoreClusterPodPlacementConfig(jsonCPPC, jsonWebhook)
		if err != nil {
			s.log.Error(err, "%%%%%%%%%%% Error updating ClusterPodPlacementConfig",
				"name", CPPC.Name)
		} else {
			s.log.Info("%%%%%%% Added ClusterPodPlacementConfig", "name", CPPC.Name, "namespace", CPPC.Namespace)
		}
	}
}

// onDelete handles the deletion of a ClusterPodPlacementConfig.
func (s *CPPCSyncer) onDelete(ic informers.ICache) func(obj interface{}) {
	return func(obj interface{}) {

		CPPC, ok := obj.(*multiarchv1beta1.ClusterPodPlacementConfig)
		if !ok {
			s.log.Error(errors.New("%%%%%%% unexpected type, expected ClusterPodPlacementConfig"), "unexpected type",
				"type", fmt.Sprintf("%T", obj))
			return
		}

		s.log.Info("%%%%%%%%%%% onDelete")

		err := ic.DeleteClusterPodPlacementConfig()
		if err != nil {
			s.log.Error(err, "%%%%%%%%%%% Error deleting ClusterPodPlacementConfig",
				"name", CPPC.Name)
		} else {
			s.log.Info("%%%%%%% Deleted ClusterPodPlacementConfig", "name", CPPC.Name, "namespace", CPPC.Namespace)
		}
	}
}

// onUpdate handles updates to a ClusterPodPlacementConfig.
func (s *CPPCSyncer) onUpdate(ic informers.ICache) func(oldObj, newObj interface{}) {
	return func(oldobj, newobj interface{}) {
		s.log.Info("%%%%%%%% onUpdate")
		oldConfig, ok := oldobj.(*multiarchv1beta1.ClusterPodPlacementConfig)

		if !ok {
			s.log.Error(errors.New("unexpected type, expected ClusterPodPlacementConfig"), "unexpected type",
				"type", fmt.Sprintf("%T", oldobj))
			return
		}

		newConfig, ok := newobj.(*multiarchv1beta1.ClusterPodPlacementConfig)
		if !ok {
			s.log.Error(errors.New("unexpected type, expected ClusterPodPlacementConfig"), "unexpected type",
				"type", fmt.Sprintf("%T", newobj))
			return
		}

		if oldConfig.ResourceVersion == newConfig.ResourceVersion {
			s.log.Info("%%%%%%%%% they are the same no update to be made")
			return
		}
		s.onAdd(ic)(newobj)
	}
}
