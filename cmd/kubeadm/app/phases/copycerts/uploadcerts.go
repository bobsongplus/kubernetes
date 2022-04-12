package copycerts

import (
	"crypto/x509"
	"fmt"
	certutil "k8s.io/client-go/util/cert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/keyutil"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

// UploadEtcdClientCerts load the certificates from the disk and upload to a Secret.
func UploadEtcdClientCerts(client clientset.Interface, cfg *kubeadmapi.InitConfiguration) error {
	certsDir := cfg.CertificatesDir

	rootCrt, _, err := pkiutil.TryLoadCertAndKeyFromDisk(certsDir, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("[upload-etcd-certs] unable to load etcd-ca in path %s", certsDir)
	}
	rootCrtEncoded := pkiutil.EncodeCertPEM(rootCrt)

	clientCrt, clientKey, err := pkiutil.TryLoadCertAndKeyFromDisk(certsDir, kubeadmconstants.EtcdHealthcheckClientCertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("[upload-etcd-certs] unable to load etcd-certs in path %s", certsDir)
	}
	crtEncoded := pkiutil.EncodeCertPEM(clientCrt)
	keyEncoded, err := keyutil.MarshalPrivateKeyToPEM(clientKey)

	EtcdSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceSystem,
			Name:      kubeadmconstants.EtcdCertsSecret,
		},
		// notice: you should not change etcd-ca, etcd-cert, etcd-key name.
		Data: map[string][]byte{
			"etcd-ca":   rootCrtEncoded,
			"etcd-cert": crtEncoded,
			"etcd-key":  keyEncoded,
		},
	}
	if err = apiclient.CreateOrUpdateSecret(client, EtcdSecret); err != nil {
		return fmt.Errorf("[upload-etcd-certs] unable to create etcd certificate secret %s: %v", EtcdSecret.Name, err)
	}
	fmt.Printf("[upload-etcd-certs] create etcd certificate secret %s in the %q Namespace\n", EtcdSecret.Name, metav1.NamespaceSystem)
	return nil
}

// UploadCalicoAPIServerCerts upload calico-apiserver certs to a secret.
func UploadCalicoAPIServerCerts(client clientset.Interface, cfg *kubeadmapi.InitConfiguration) error {
	certsDir := cfg.CertificatesDir

	cacrt, cakey, err := pkiutil.TryLoadCertAndKeyFromDisk(certsDir, kubeadmconstants.CACertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("[upload-calico-apiserver-certs] unable to load kubernetes ca in path %s", certsDir)
	}
	certconfig := pkiutil.CertConfig{
		Config: certutil.Config{
			CommonName:   "calico-apiserver",
			Organization: []string{"calico-apiserver"},
			AltNames: certutil.AltNames{
				//san must be svcName.nsName.svc
				//error trying to reach service: x509: certificate is valid for calico-apiserver.kube-system, not calico-apiserver.kube-system.svc
				DNSNames: []string{fmt.Sprintf("%s.%s.svc", "calico-apiserver", metav1.NamespaceSystem)},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		},
		PublicKeyAlgorithm: x509.RSA,
	}

	apiservercert, apiserverkey, err := pkiutil.NewCertAndKey(cacrt, cakey, &certconfig)
	if err != nil {
		fmt.Errorf("error when create calico apiserver.crt and apiserver.key: %v", err)
	}
	apiserverCertBytes := pkiutil.EncodeCertPEM(apiservercert)
	apiserverKeyBytes, err := keyutil.MarshalPrivateKeyToPEM(apiserverkey)
	if err != nil {
		fmt.Errorf("error when marshal calico apiserver private key %s", "apiserver.key")
	}

	CalicoAPIServerSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceSystem,
			//should be same with cmd/kubeadm/app/phases/network/calico/k8s/manifest.go#L789
			Name: kubeadmconstants.CalicoAPIServerSecret,
		},
		// notice: you should not change apiserver.crt, apiserver.key name.
		Data: map[string][]byte{
			"apiserver.crt": apiserverCertBytes,
			"apiserver.key": apiserverKeyBytes,
		},
	}
	if err = apiclient.CreateOrUpdateSecret(client, CalicoAPIServerSecret); err != nil {
		return fmt.Errorf("[upload-calico-apiserver-certs] unable to create calico-apiserver secret %s: %v", CalicoAPIServerSecret.Name, err)
	}
	fmt.Printf("[upload-calico-apiserver-certs] create calico-apiserver secret %s in the %q Namespace\n", CalicoAPIServerSecret.Name, metav1.NamespaceSystem)
	return nil
}
