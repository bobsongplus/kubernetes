package copycerts

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"path"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

// CreateEtcdClientCerts generates the certificates needed by network plugins.
func CreateEtcdClientCerts(client clientset.Interface, cfg *kubeadmapi.InitConfiguration) error {
	fmt.Printf("[etcd-client-certs] Downloading the certificates in Secret %q in the %q Namespace\n", kubeadmconstants.KubeadmCertsSecret, metav1.NamespaceSystem)

	secret, err := getSecret(client)
	if err != nil {
		return errors.Wrap(err, "error downloading the secret")
	}

	decodedKey, err := hex.DecodeString(secret.Annotations[certificateKey])
	if err != nil {
		return errors.Wrap(err, "error decoding certificate key")
	}

	secretData, err := getDataFromSecret(secret, decodedKey)
	if err != nil {
		return errors.Wrap(err, "error decoding secret data with provided key")
	}

	for certOrKeyName, certOrKeyPath := range certsToPath(cfg) {
		certOrKeyData, found := secretData[certOrKeyNameToSecretName(certOrKeyName)]
		if !found {
			return errors.New("couldn't find required certificate or key in Secret")
		}
		if len(certOrKeyData) == 0 {
			klog.V(1).Infof("[download-certs] Not saving %q to disk, since it is empty in the %q Secret\n", certOrKeyName, kubeadmconstants.KubeadmCertsSecret)
			continue
		}
		if err := writeCertOrKey(certOrKeyPath, certOrKeyData); err != nil {
			return err
		}
	}
	etcdCaCert, err := cert.ParseCertsPEM(secretData[certOrKeyNameToSecretName(kubeadmconstants.EtcdCACertName)])
	if err != nil {
		return fmt.Errorf("[etcd-client-certs] failed to Parse etcd ca cert [%v]", err)
	}
	etcdCaKey, err := keyutil.ParsePrivateKeyPEM(secretData[certOrKeyNameToSecretName(kubeadmconstants.EtcdCAKeyName)])
	if err != nil {
		return fmt.Errorf("[etcd-client-certs] failed to Parse etcd ca key [%v]", err)
	}
	// sign etcd client certificate with etcd ca
	certCfg := &pkiutil.CertConfig{
		Config: cert.Config{
			CommonName:   kubeadmconstants.EtcdClientCertCommonName,
			Organization: []string{kubeadmconstants.NodesGroup},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}
	etcdClientCert, etcdClientKey, err := pkiutil.NewCertAndKey(etcdCaCert[0], etcdCaKey.(*rsa.PrivateKey), certCfg)
	if err != nil {
		return fmt.Errorf("[etcd-client-certs] failed to Create etcd client certificate & key   [%v]", err)
	}
	if err := pkiutil.WriteCertAndKey(cfg.CertificatesDir, kubeadmconstants.EtcdClientCertAndKeyBaseName, etcdClientCert, etcdClientKey); err != nil {
		return fmt.Errorf("[etcd-client-certs] failed while saving %s certificate and key: %v", kubeadmconstants.EtcdClientCertAndKeyBaseName, err)
	}
	return nil
}

func certsToPath(cfg *kubeadmapi.InitConfiguration) map[string]string {
	certsDir := cfg.CertificatesDir
	certs := map[string]string{}
	if cfg.Etcd.External == nil {
		certs[kubeadmconstants.EtcdCACertName] = path.Join(certsDir, kubeadmconstants.EtcdCACertName)
		certs[kubeadmconstants.EtcdCAKeyName] = path.Join(certsDir, kubeadmconstants.EtcdCAKeyName)
	}
	return certs
}

// UploadEtcdMetricClientCerts load the certificates from the disk and upload to a Secret.
func UploadEtcdClientCerts(client clientset.Interface, cfg *kubeadmapi.InitConfiguration) error {
	certsDir := cfg.CertificatesDir

	rootCrt, _, err := pkiutil.TryLoadCertAndKeyFromDisk(certsDir, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("[upload-etcd-certs] unable to load etcd-ca in path %s", certsDir)
	}
	rootCrtEncoded := pkiutil.EncodeCertPEM(rootCrt)

	clientCrt, clientKey, err := pkiutil.TryLoadCertAndKeyFromDisk(certsDir, kubeadmconstants.EtcdMetricClientCertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("[upload-etcd-certs] unable to load etcd-metric-crt in path %s", certsDir)
	}
	crtEncoded := pkiutil.EncodeCertPEM(clientCrt)
	keyEncoded, err := keyutil.MarshalPrivateKeyToPEM(clientKey)

	EtcdSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceSystem,
			Name:      kubeadmconstants.EtcdCertsSecret,
		},
		Data: map[string][]byte{
			"ca.crt":     rootCrtEncoded,
			"client.crt": crtEncoded,
			"client.key": keyEncoded,
		},
	}
	if err = apiclient.CreateOrUpdateSecret(client, EtcdSecret); err != nil {
		return fmt.Errorf("[upload-etcd-certs] unable to create etcd certificate secret %s: %v", EtcdSecret.Name, err)
	}
	fmt.Printf("[upload-etcd-certs] create etcd certificate secret %s in the %q Namespace\n", EtcdSecret.Name, metav1.NamespaceSystem)
	return nil
}
