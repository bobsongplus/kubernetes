package copycerts

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/keyutil"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

// UploadEtcdMetricClientCerts load the certificates from the disk and upload to a Secret.
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
