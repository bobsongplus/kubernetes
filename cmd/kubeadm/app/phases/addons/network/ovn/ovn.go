/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 2021-01-04  @author weiwei@tenxcloud.com
 */
package ovn

import (
	"crypto/x509"
	"fmt"
	"runtime"

	"github.com/google/uuid"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/patchnode"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

func CreateOvnAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 0: label kubernetes master with ovn.io/role=central
	//TODO: FIXME show used openebs to store ovn-nb/ovn-sb data
	labels := map[string]string{"ovn.io/role": "central"}
	err := patchnode.LabelNode(client, cfg.NodeRegistration.Name, labels)
	if err != nil {
		return fmt.Errorf("error when label kubernetes master: %v", err)
	}
	//PHASE 1: create native ovn
	ovnCentralDeploymentBytes, err := kubeadmutil.ParseTemplate(OvnCentralDeployment, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn-central deployment template: %v", err)
	}
	ovnHostDaemonSetBytes, err := kubeadmutil.ParseTemplate(OvnHostDaemonSet, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn-host daemonset template: %v", err)
	}
	if err := createNativeOvn(ovnCentralDeploymentBytes, ovnHostDaemonSetBytes, client); err != nil {
		return err
	}
	//PHASE 2: create custom ovn
	ovnControllerDeploymentBytes, err := kubeadmutil.ParseTemplate(OvnControllerDeployment, struct{ ImageRepository, Arch, Version, PodSubnet, NodeSubnet string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
		PodSubnet:       defaultSubnet,
		NodeSubnet:      cfg.Networking.NodeSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn-controller deployment template: %v", err)
	}
	ovnDaemonDaemonSetBytes, err := kubeadmutil.ParseTemplate(OvnDaemonDaemonSet, struct{ ImageRepository, Arch, Version, ServiceSubnet string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
		ServiceSubnet:   cfg.Networking.ServiceSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn-daemon daemonset template: %v", err)
	}
	ovnInspectorDaemonSetBytes, err := kubeadmutil.ParseTemplate(OvnInspectorDaemonSet, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn-inspector daemonset template: %v", err)
	}
	if err := createCustomOvn(ovnControllerDeploymentBytes, ovnDaemonDaemonSetBytes, ovnInspectorDaemonSetBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: ovn")
	return nil
}

func createNativeOvn(OvnCentralDeploymentBytes, OvnHostDaemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 0: create ovn certs ?
    err := createOvnPki(client)
	if err != nil {
		return fmt.Errorf("error when create ovn pki: %v", err)
	}
	//PHASE 1: create ConfigMap for ovn-controller
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ConfigMap), configMap); err != nil {
		return fmt.Errorf("unable to decode ovn configmap %v", err)
	}

	// Create the ConfigMap for ovn-controller or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}

	//PHASE 2: create RBAC rules
	clusterRole := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRole); err != nil {
		return fmt.Errorf("unable to decode ovn  clusterrole %v", err)
	}

	// Create the ClusterRole for ovn or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRole); err != nil {
		return err
	}

	clusterRoleBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRoleBinding); err != nil {
		return fmt.Errorf("unable to decode ovn clusterRoleBinding %v", err)
	}

	// Create the ClusterRoleBinding for ovn or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRoleBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode ovn serviceAccount %v", err)
	}

	// Create the ServiceAccount for ovn or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create native ovn service
	if err := createService([]string{OvnNBService, OvnSBService, OvnExporterService}, client); err != nil {
		return err
	}

	//PHASE 4: create ovn-central deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), OvnCentralDeploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode ovn-host deployment %v", err)
	}
	// Create the Deployment for ovn-central or update it in case it already exists
	if err := apiclient.CreateOrUpdateDeployment(client, deployment); err != nil {
		return err
	}
	//TODO: FIXME should wait ovn-central ready ??

	//PHASE 5: create ovn-host daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), OvnHostDaemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode ovn-host daemonset %v", err)
	}
	// Create the DaemonSet for ovn-host or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}

func createCustomOvn(OvnControllerDeploymentBytes, OvnDaemonDaemonSetBytes, OvnInspectorDaemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create custom ovn service
	if err := createService([]string{OvnControllerService, OvnDaemonService, OvnInspectorService}, client); err != nil {
		return err
	}
	//PHASE 2: create ovn-deployment deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), OvnControllerDeploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode ovn-controller deplayment %v", err)
	}
	// Create the Deployment for ovn-controller or update it in case it already exists
	if err := apiclient.CreateOrUpdateDeployment(client, deployment); err != nil {
		return err
	}
	//PHASE 3: create ovn-daemon daemonSet
	ovndaemon := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), OvnDaemonDaemonSetBytes, ovndaemon); err != nil {
		return fmt.Errorf("unable to decode ovn-daemon daemonset %v", err)
	}
	// Create the Deployment for ovn-daemon or update it in case it already exists
	if err := apiclient.CreateOrUpdateDaemonSet(client, ovndaemon); err != nil {
		return err
	}
	//PHASE 4: create ovn-inspector daemonSet
	ovninspector := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), OvnInspectorDaemonSetBytes, ovninspector); err != nil {
		return fmt.Errorf("unable to decode ovn-inspector daemonset %v", err)
	}
	// Create the DaemonSet for ovn-inspector or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, ovninspector)
}

func createService(services []string, client clientset.Interface) error {
	service := &v1.Service{}
	for _, s := range services {
		if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(s), service); err != nil {
			return fmt.Errorf("unable to decode ovn service %v", err)
		}
		// Create the Service for ovn or update it in case it already exists
		if err := apiclient.CreateOrUpdateService(client, service); err != nil {
			return err
		}
	}
	return nil
}

func createOvnPki(client clientset.Interface) error {
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
	return apiclient.CreateOrUpdateSecret(client, ovncerts)
}
