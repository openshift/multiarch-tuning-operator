package system_config

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"os"
	"sync"
)

var (
	singletonSystemConfigInstance IConfigSyncer
	once                          sync.Once
)

type SystemConfigSyncer struct {
	registriesConfContent registriesConf
	policyConfContent     policyConf
	registryCertTuples    []registryCertTuple

	ch chan bool
	mu sync.Mutex
}

// SystemConfigSyncerSingleton returns the singleton instance of the SystemConfigSyncer
func SystemConfigSyncerSingleton() IConfigSyncer {
	once.Do(func() {
		singletonSystemConfigInstance = newSystemConfigSyncer()
	})
	return singletonSystemConfigInstance
}

func (s *SystemConfigSyncer) StoreImageRegistryConf(allowedRegistries []string, blockedRegistries []string, insecureRegistries []string) error {
	if len(allowedRegistries) > 0 && len(blockedRegistries) > 0 {
		return fmt.Errorf("only one of allowedRegistries and blockedRegistries can be set. Ignoring this event")
	}
	s.mu.Lock()
	defer s.unlockAndSync()
	// Ensure the previous state is reset
	for _, rc := range s.registriesConfContent.Registries {
		rc.Allowed = nil
		rc.Blocked = nil
		rc.Insecure = nil
	}
	s.policyConfContent.resetTransports()
	// At the time of writing, we don't see the need to generate multiple bool pointers. Keeping it the same, but at
	// the registryConf level.
	trueValue := true
	for _, registry := range allowedRegistries {
		rc := s.registriesConfContent.getRegistryConfOrCreate(registry)
		rc.Allowed = &trueValue
		rc.Blocked = nil
	}
	for _, registry := range blockedRegistries {
		rc := s.registriesConfContent.getRegistryConfOrCreate(registry)
		rc.Allowed = nil
		rc.Blocked = &trueValue
		s.policyConfContent.setRejectForRegistry(registry)
	}
	for _, registry := range insecureRegistries {
		rc := s.registriesConfContent.getRegistryConfOrCreate(registry)
		rc.Insecure = &trueValue
	}
	s.registriesConfContent.cleanupAllRegistryConfIfEmpty()
	return nil
}

func (s *SystemConfigSyncer) unlockAndSync() {
	s.mu.Unlock()
	s.ch <- true
}

func (s *SystemConfigSyncer) StoreRegistryCerts(registryCertTuples []registryCertTuple) error {
	s.mu.Lock()
	defer s.unlockAndSync()
	s.registryCertTuples = registryCertTuples
	return nil
}

func (s *SystemConfigSyncer) UpdateRegistryMirroringConfig(registry string, mirrors []string) error {
	s.mu.Lock()
	defer s.unlockAndSync()
	rc := s.registriesConfContent.getRegistryConfOrCreate(registry)
	rc.Mirrors = mirrorsFor(mirrors)
	return nil
}

func (s *SystemConfigSyncer) DeleteRegistryMirroringConfig(registry string) error {
	s.mu.Lock()
	defer s.unlockAndSync()
	if rc, ok := s.registriesConfContent.getRegistryConf(registry); ok {
		rc.Mirrors = nil
		s.registriesConfContent.cleanupRegistryConfIfEmpty(registry)
		return nil
	}
	return fmt.Errorf("registry %s not found", registry)
}

func (s *SystemConfigSyncer) CleanupRegistryMirroringConfig() error {
	s.mu.Lock()
	defer s.unlockAndSync()
	for _, registry := range s.registriesConfContent.Registries {
		registry.Mirrors = nil
		s.registriesConfContent.cleanupRegistryConfIfEmpty(registry.Location)
	}
	return nil
}

func (s *SystemConfigSyncer) sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// marshall registries.conf and write to file
	if err := s.registriesConfContent.writeToFile(); err != nil {
		klog.Errorf("error writing registries.conf: %v", err)
		return err
	}
	// marshall policy.json and write to file
	if err := s.policyConfContent.writeToFile(); err != nil {
		klog.Errorf("error writing policy.json: %v", err)
		return err
	}
	// delete the certs.d content
	if err := os.RemoveAll(DockerCertsDir); err != nil {
		klog.Errorf("error deleting certs.d directory: %v", err)
		return err
	}
	// write registry certs to file
	for _, tuple := range s.registryCertTuples {
		if err := tuple.writeToFile(); err != nil {
			klog.Errorf("error writing registry cert: %v", err)
			return err
		}
	}
	return nil
}

func (s *SystemConfigSyncer) getch() chan bool {
	return s.ch
}

// Namespaced RBAC rules and cluster scoped RBAC rules cannot be combined through the controller-gen RBAC generator.
// See https://github.com/kubernetes-sigs/controller-tools/pull/839 and https://github.com/kubernetes-sigs/controller-tools/pull/839
// This rbac rule is added manually.
//#kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch,resourceNames=image-registry-certificates,namespace="openshift-image-registry"

//+kubebuilder:rbac:groups=config.openshift.io,resources=images,verbs=get;list;watch

//+kubebuilder:rbac:groups=operator.openshift.io,resources=imagecontentsourcepolicies,verbs=get;list;watch

// newSystemConfigSyncer creates a new SystemConfigSyncer object
func newSystemConfigSyncer() IConfigSyncer {
	ic := &SystemConfigSyncer{
		registriesConfContent: defaultRegistriesConf(),
		policyConfContent:     defaultPolicyConf(),
		registryCertTuples:    []registryCertTuple{},
		ch:                    make(chan bool),
	}
	return ic
}

type ConfigSyncerRunnable struct{}

func (r *ConfigSyncerRunnable) Start(ctx context.Context) error {
	s := SystemConfigSyncerSingleton()
	for {
		select {
		case <-s.getch():
			if err := s.sync(); err != nil {
				klog.Errorf("error syncing system config: %v", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// ParseRegistryCerts parses the registry certs from a map of registry url to cert
// This map, in ocp, is stored in the data field of the configmap "image-registry-certifiates" in the
// openshift-image-registry namespace.
func ParseRegistryCerts(dataMap map[string]string) []registryCertTuple {
	var registryCertTuples []registryCertTuple
	for k, v := range dataMap {
		registryCertTuples = append(registryCertTuples, registryCertTuple{
			registry: k,
			cert:     v,
		})
	}
	return registryCertTuples
}
