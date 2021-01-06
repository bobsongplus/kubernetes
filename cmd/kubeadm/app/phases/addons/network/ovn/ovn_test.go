package ovn

import (
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

func TestCreateOvnPKi(t *testing.T) {
	caconfig := pkiutil.CertConfig{
		Config: certutil.Config{
			CommonName: "OVS switchca CA Certificate",
			Organization: []string{"Open vSwitch"},
		},
		PublicKeyAlgorithm: x509.RSA,
	}
	cacert, cakey, err := pkiutil.NewCertificateAuthority(&caconfig)
	if err != nil {
		fmt.Errorf("error when create ovn ca.crt and ca.key: %v", err)
	}
	//ovn id:734fe6d7-49bc-40a2-8c08-7ec0fec1e9b0
	cnAndsan := fmt.Sprintf("ovn id:%s",uuid.New().String())
	certconfig := pkiutil.CertConfig{
		Config: certutil.Config{
			CommonName: cnAndsan,
			Organization: []string{"Open vSwitch"},
			AltNames: certutil.AltNames{
				DNSNames: []string{cnAndsan},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		},
		PublicKeyAlgorithm: x509.RSA,
	}
	cert, key, err :=pkiutil.NewCertAndKey(cacert, cakey, &certconfig)
	if err != nil {
		fmt.Errorf("error when create ovn ovn-cert.pem and ovn-privkey.pem: %v", err)
	}
	keyBytes, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		fmt.Errorf("error when marshal ovn ovn-privkey.pem: %v", err)
	}
	ovncerts := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeadmconstants.OvnCertsSecret,
			Namespace: metav1.NamespaceSystem,
		},
		Data: map[string][]byte{
			"ca.crt":  pkiutil.EncodeCertPEM(cacert),
			"ovn.crt": pkiutil.EncodeCertPEM(cert),
			"ovn.key": keyBytes,
		},
	}

	fmt.Println(ovncerts.Name)

}
