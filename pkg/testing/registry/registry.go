package registry

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	ocpconfigv1 "github.com/openshift/api/config/v1"

	. "github.com/openshift/multiarch-tuning-operator/pkg/testing/builder"
	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

type RegistryConfig struct {
	Namespace           *corev1.Namespace
	Name                string
	RegistryHost        string
	CertConfigmapName   string
	KeyPath             string
	CaPath              string
	RegistryProxyUrl    string
	RegistryProxyUser   string
	RegistryProxyPasswd string
	Port                int32
}

func NewRegistry(ns *corev1.Namespace, name, certConfigmapName, registryProxyUrl, registryProxyUser, registryProxyPasswd string) (*RegistryConfig, error) {
	registryHost := fmt.Sprintf("%s.%s.svc.cluster.local", name, ns.Name)
	serverTLS, err := buildRegistryTLSConfig(registryHost)
	if err != nil {
		return nil, err
	}
	return &RegistryConfig{
		Namespace:           ns,
		Name:                name,
		RegistryHost:        registryHost,
		CertConfigmapName:   certConfigmapName,
		KeyPath:             serverTLS.privateKeyPath,
		CaPath:              serverTLS.certificatePath,
		RegistryProxyUrl:    registryProxyUrl,
		RegistryProxyUser:   registryProxyUser,
		RegistryProxyPasswd: registryProxyPasswd,
		Port:                5001,
	}, nil
}

func Deploy(ctx context.Context, client runtimeclient.Client, r *RegistryConfig) error {
	var (
		registryLabel = map[string]string{"app": r.Name}
		httpProxy     = ""
		httpsProxy    = ""
		noProxy       = ""
	)
	// Create service
	log.Printf("create service for %s registry", r.Name)
	service := NewService().
		WithName(r.Name).
		WithNamespace(r.Namespace.Name).
		WithPorts(
			NewServicePort().
				WithTCPProtocol().
				WithPort(r.Port).
				WithTargetPort(r.Port).
				Build()).
		WithSelector(registryLabel).
		Build()
	err := client.Create(ctx, service)
	if err != nil {
		return err
	}

	// Create secret
	log.Printf("create secret for %s registry", r.Name)
	tlsKeyData, err := os.ReadFile(r.KeyPath)
	if err != nil {
		return err
	}
	tlsCrtData, err := os.ReadFile(r.CaPath)
	if err != nil {
		return err
	}
	secretData := map[string][]byte{"tls.crt": tlsCrtData, "tls.key": tlsKeyData}
	secret := NewSecret().
		WithData(secretData).
		WithOpaqueType().
		WithName(r.Name).
		WithNameSpace(r.Namespace.Name).
		Build()
	err = client.Create(ctx, secret)
	if err != nil {
		return err
	}

	// Create configmap for ca bundles for cluster-wide proxy
	log.Printf("create configmap for ca bundles if cluster-wide proxy exist")
	config := &v1.ConfigMap{}
	err = client.Get(ctx, runtimeclient.ObjectKeyFromObject(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-trusted-ca",
			Namespace: r.Namespace.Name,
		},
	}), config)
	if err != nil {
		if runtimeclient.IgnoreNotFound(err) == nil {
			configmap := NewConfigMap().
				WithLabels(map[string]string{"config.openshift.io/inject-trusted-cabundle": "true"}).
				WithName("registry-trusted-ca").
				WithNamespace(r.Namespace.Name).
				Build()
			err = client.Create(ctx, configmap)
			if err != nil {
				return err
			}
			config = configmap
		} else {
			return err
		}
	}
	proxy, err := getClusterProxy(ctx, client)
	if err != nil {
		return err
	}
	if proxy != nil {
		httpProxy = proxy.Spec.HTTPProxy
		httpsProxy = proxy.Spec.HTTPSProxy
		noProxy = proxy.Spec.NoProxy
	}

	// Create deployment
	log.Printf("create deployment for %s registry", r.Name)
	registryD := NewDeployment().
		WithSelectorAndPodLabels(registryLabel).
		WithPodSpec(
			NewPodSpec().
				WithContainers(
					NewContainer().
						WithImage("quay.io/openshifttest/registry:1.2.0").
						WithVolumeMounts(NewVolumeMount().WithName("registry-storage").WithMountPath("/var/lib/registry").Build(),
							NewVolumeMount().WithName("registry-secret").WithMountPath("/etc/secrets").Build(),
							NewVolumeMount().WithName("trusted-ca").WithMountPath("/etc/pki/ca-trust/extracted/pem").WithReadOnly().Build()).
						WithEnv(NewContainerEnv().WithName("REGISTRY_HTTP_ADDR").WithValue(":5001").Build(),
							NewContainerEnv().WithName("REGISTRY_HTTP_TLS_CERTIFICATE").WithValue("/etc/secrets/tls.crt").Build(),
							NewContainerEnv().WithName("REGISTRY_HTTP_TLS_KEY").WithValue("/etc/secrets/tls.key").Build(),
							NewContainerEnv().WithName("REGISTRY_STORAGE_DELETE_ENABLED").WithValue("true").Build(),
							NewContainerEnv().WithName("REGISTRY_PROXY_REMOTEURL").WithValue(r.RegistryProxyUrl).Build(),
							NewContainerEnv().WithName("REGISTRY_PROXY_USERNAME").WithValue(r.RegistryProxyUser).Build(),
							NewContainerEnv().WithName("REGISTRY_PROXY_PASSWORD").WithValue(r.RegistryProxyPasswd).Build(),
							NewContainerEnv().WithName("HTTP_PROXY").WithValue(httpProxy).Build(),
							NewContainerEnv().WithName("HTTPS_PROXY").WithValue(httpsProxy).Build(),
							NewContainerEnv().WithName("NO_PROXY").WithValue(noProxy).Build(),
						).
						WithPortsContainerPort(r.Port).
						Build()).
				WithVolumes(NewVolume().WithName("registry-storage").WithVolumeEmptyDir(&corev1.EmptyDirVolumeSource{}).Build(),
					NewVolume().WithName("registry-secret").WithVolumeProjectedDefaultMode(utils.NewPtr(int32(420))).
						WithVolumeProjectedSourcesSecretLocalObjectReference(secret.Name).Build(),
					NewVolume().WithVolumeSourceConfigmap(config.Name, v1.KeyToPath{Key: "ca-bundle.crt", Path: "tls-ca-bundle.pem"}).
						WithName("trusted-ca").Build()).
				Build()).
		WithReplicas(utils.NewPtr(int32(1))).
		WithName(r.Name).
		WithNamespace(r.Namespace.Name).
		Build()
	err = client.Create(ctx, registryD)
	if err != nil {
		return err
	}
	return nil
}

func AddCertificateToConfigmap(ctx context.Context, client runtimeclient.Client, r *RegistryConfig) error {
	caData, err := os.ReadFile(r.CaPath)
	if err != nil {
		return err
	}
	c := v1.ConfigMap{}
	err = client.Get(ctx, runtimeclient.ObjectKeyFromObject(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.CertConfigmapName,
			Namespace: "openshift-config",
		},
	}), &c)
	if err != nil {
		if runtimeclient.IgnoreNotFound(err) == nil {
			configMapData := map[string]string{fmt.Sprintf("%s..%d", r.RegistryHost, r.Port): string(caData)}
			configmap := NewConfigMap().
				WithData(configMapData).
				WithName(r.CertConfigmapName).
				WithNamespace("openshift-config").
				Build()
			err = client.Create(ctx, configmap)
			if err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	if c.Data == nil {
		c.Data = map[string]string{}
	}
	c.Data[fmt.Sprintf("%s..%d", r.RegistryHost, r.Port)] = string(caData)
	err = client.Update(ctx, &c)
	if err != nil {
		return err
	}
	return nil
}

func RemoveCertificateFromConfigmap(ctx context.Context, client runtimeclient.Client, r *RegistryConfig) error {
	c := v1.ConfigMap{}
	err := client.Get(ctx, runtimeclient.ObjectKeyFromObject(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.CertConfigmapName,
			Namespace: "openshift-config",
		},
	}), &c)
	if err != nil {
		if runtimeclient.IgnoreNotFound(err) == nil {
			return nil
		} else {
			return err
		}
	}
	delete(c.Data, fmt.Sprintf("%s..%d", r.RegistryHost, r.Port))
	err = client.Update(ctx, &c)
	if err != nil {
		return err
	}
	return nil
}

func getClusterProxy(ctx context.Context, client runtimeclient.Client) (*ocpconfigv1.Proxy, error) {
	proxy := ocpconfigv1.Proxy{}
	err := client.Get(ctx, runtimeclient.ObjectKeyFromObject(&ocpconfigv1.Proxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}), &proxy)
	if err != nil {
		if runtimeclient.IgnoreNotFound(err) == nil {
			log.Printf("No cluster-wide proxy found, return")
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &proxy, nil
}

type registryTLSConfig struct {
	certificatePath string
	privateKeyPath  string
	certificate     *tls.Certificate
	caCertBytes     []byte
}

// createRegistryCertificate creates a certificate and returns: (path to the key used to sign the CA, path to the CA, error)
func buildRegistryTLSConfig(dns string) (*registryTLSConfig, error) {
	var (
		caFileName  = "registry.crt"
		keyFileName = "registry.key"
		priv        interface{}
		pub         crypto.PublicKey
		err         error
	)
	// Generate private key
	priv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to create rsa private key: %v", err)
	}
	rsaKey := priv.(*rsa.PrivateKey)
	pub = rsaKey.Public()
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 6)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to create serial number: %v", err)
	}
	cert := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"MultiArch-QE"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{},
		DNSNames:              []string{dns},
		IsCA:                  true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}
	if _, err = os.Stat(os.TempDir()); os.IsNotExist(err) {
		//nolint:errcheck
		//#nosec:G301 (CWE-276): Expect directory permissions to be 0750 or less (Confidence: HIGH, Severity: MEDIUM)
		os.Mkdir(os.TempDir(), 1777)
	}
	tmpDir := path.Join(os.TempDir(), fmt.Sprintf("mto-test-%s", uuid.New().String()))
	err = os.MkdirAll(tmpDir, 0750)
	if err != nil {
		return nil, err
	}
	certPath := path.Join(tmpDir, caFileName)
	// create a folder for the certs
	//#nosec:G304 (CWE-22): Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM)
	certOut, err := os.Create(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create pem: %v", err)
	}
	caCertBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	// write caCertBytes to disk
	_, err = certOut.Write(caCertBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to %s: %v", certPath, err)
	}
	if err := certOut.Close(); err != nil {
		return nil, fmt.Errorf("error closing %s: %v", certPath, err)
	}
	keyPath := path.Join(tmpDir, keyFileName)
	//#nosec G304 (CWE-22): Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM)
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s for writing: %v", keyPath, err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return nil, fmt.Errorf("failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		return nil, fmt.Errorf("error closing %s: %v", keyPath, err)
	}
	tlsCert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	tlsTestCfg := registryTLSConfig{
		certificatePath: certPath,
		privateKeyPath:  keyPath,
		certificate:     &tlsCert,
		caCertBytes:     caCertBytes,
	}

	return &tlsTestCfg, nil
}

func RemoveCertificateFiles(keyPath string) error {
	dir := path.Dir(keyPath)
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	return nil
}
