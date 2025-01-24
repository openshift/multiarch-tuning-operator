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
	"math/big"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"

	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"

	_ "github.com/openshift/multiarch-tuning-operator/pkg/testing/image/fake/registry/auth/htpasswd"
)

// Adapted from https://github.com/distribution/distribution/blob/main/registry/registry_test.go#L56

type registryTLSConfig struct {
	cipherSuites    []string
	certificatePath string
	privateKeyPath  string
	certificate     *tls.Certificate
	caCertBytes     []byte
}

var (
	perRegistryCertDirPath string
	mu                     sync.Mutex
)

const (
	// TODO: Port 5001 could be busy, we might set this to 0 to get an ephemeral port, but the Registry object
	// does not expose the server object, hence we can't get the port from there.
	url = "localhost:5001"
)

// RunRegistry starts a registry server in a goroutine and returns the url and the caCertBytes
// It is the only one function that should be called from outside this package to instantiate a registry server
func RunRegistry(ctx context.Context, registryChan chan error) (string, []byte, error) {
	serverTLS, err := buildRegistryTLSConfig()
	if err != nil {
		return "", nil, err
	}
	log.Info("serverTLS: ", "tlsConf", serverTLS)

	registryServer, err := setupRegistry(ctx, serverTLS, url)
	if err != nil {
		return "", nil, err
	}

	go func() {
		registryChan <- registryServer.ListenAndServe()
	}()
	return url, serverTLS.caCertBytes, nil
}

// setupRegistry creates a registry server with the given configuration
func setupRegistry(ctx context.Context, tlsCfg *registryTLSConfig, addr string) (*registry.Registry, error) {
	config := &configuration.Configuration{}
	config.HTTP.Addr = addr
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	if tlsCfg != nil {
		config.HTTP.TLS.CipherSuites = tlsCfg.cipherSuites
		config.HTTP.TLS.Certificate = tlsCfg.certificatePath
		config.HTTP.TLS.Key = tlsCfg.privateKeyPath
	}
	config.Storage = configuration.Storage{"inmemory": configuration.Parameters{}}
	config.Log.AccessLog.Disabled = true
	config.Log.Level = "warn"
	config.Auth = configuration.Auth{
		"htpasswd_authorization": configuration.Parameters{
			"realm":              "https://my-registry.io",
			"credentials":        getMockCredentials(),
			"allowedUsersByRepo": getMockAllowedUsersByRepos(),
		},
	}
	return registry.NewRegistry(ctx, config)
}

// buildRegistryTLSConfig creates a TLS configuration for the registry server. The output certificate
// is self-signed and valid for 6 hours. It should be added in the image-registry-certificates configmap
// to be trusted by the test cluster.
func buildRegistryTLSConfig() (*registryTLSConfig, error) {
	var priv interface{}
	var pub crypto.PublicKey
	var err error
	cipherSuites := []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"}
	name := "registry_test_server"

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
			Organization: []string{"registry_test"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
		IsCA:                  true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}
	if _, err := os.Stat(os.TempDir()); os.IsNotExist(err) {
		//nolint:errcheck
		//#nosec:G301 (CWE-276): Expect directory permissions to be 0750 or less (Confidence: HIGH, Severity: MEDIUM)
		os.Mkdir(os.TempDir(), 1777)
	}

	// create a folder for the certs
	perRegistryCertDirPath, _ = os.MkdirTemp("", "registry_test")
	err = os.Mkdir(path.Join(perRegistryCertDirPath, url), 0750)
	if err != nil {
		return nil, err
	}
	certPath := path.Join(perRegistryCertDirPath, url, "ca.crt")
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

	keyPath := path.Join(os.TempDir(), name+".key")

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
		cipherSuites:    cipherSuites,
		certificatePath: certPath,
		privateKeyPath:  keyPath,
		certificate:     &tlsCert,
		caCertBytes:     caCertBytes,
	}

	return &tlsTestCfg, nil
}

func SetCertPath(path string) {
	mu.Lock()
	defer mu.Unlock()
	perRegistryCertDirPath = path
}

func GetCertPath() string {
	return perRegistryCertDirPath
}
