/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package calico

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/calico"
	"strings"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

// kube-controller
// calico-node
// calico-typha 2
// calico-apiserver

func CreateCalicoAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 0: create calico typha
	typhaDeploymentBytes, err := kubeadmutil.ParseTemplate(Typha, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico typha deployment template: %v", err)
	}
	if err := createCalicoTypha(typhaDeploymentBytes, client); err != nil {
		return err
	}
	//PHASE 1: create calico node containers
	var iPAutoDetection, iP6AutoDetection, assignIpv4, assignIpv6 string
	if kubeadmconstants.GetNetworkMode(defaultSubnet) == kubeadmconstants.NetworkIPV6Mode { // only ipv6
		iPAutoDetection = "none"
		iP6AutoDetection = "autodetect"
		assignIpv4 = "false"
		assignIpv6 = "true"
	} else if kubeadmconstants.GetNetworkMode(defaultSubnet) == kubeadmconstants.NetworkDualStackMode { // ipv4 & ipv6
		iPAutoDetection = "autodetect"
		iP6AutoDetection = "autodetect"
		assignIpv4 = "true"
		assignIpv6 = "true"
	} else { // ipv4
		iPAutoDetection = "autodetect"
		iP6AutoDetection = "none"
		assignIpv4 = "true"
		assignIpv6 = "false"
	}
	// Generate ControlPlane Endpoints
	controlPlaneEndpoint, err := kubeadmutil.GetControlPlaneEndpoint(cfg.ControlPlaneEndpoint, &cfg.LocalAPIEndpoint)
	if err != nil {
		return err
	}
	if err = createKubernetesServicesEndpoint(controlPlaneEndpoint, client); err != nil {
		return err
	}
	nodeConfigMapBytes, err := kubeadmutil.ParseTemplate(NodeConfigMap, struct{ IPAutoDetection, IP6AutoDetection, AssignIpv4, AssignIpv6 string }{
		IPAutoDetection:  iPAutoDetection,
		IP6AutoDetection: iP6AutoDetection,
		AssignIpv4:       assignIpv4,
		AssignIpv6:       assignIpv6,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico cni configmap template: %v", err)
	}
	cniDaemonSetBytes, err := kubeadmutil.ParseTemplate(Node, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico node daemonset template: %v", err)
	}
	if err := createCalicoNode(cniDaemonSetBytes, nodeConfigMapBytes, client); err != nil {
		return err
	}
	//PHASE 2: create calico kube controllers containers
	kubeControllerDeploymentBytes, err := kubeadmutil.ParseTemplate(KubeController, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing kube controllers deployment template: %v", err)
	}
	if err := createKubeControllers(kubeControllerDeploymentBytes, client); err != nil {
		return err
	}
	//PHASE 3: create calico job to configure ip pool
	if err := createCalicoBootstraper(defaultSubnet, cfg.GetControlPlaneImageRepository(), client); err != nil {
		return err
	}
	//PHASE 4: create calico apiserver containers
	apiServerDeploymentBytes, err := kubeadmutil.ParseTemplate(APIServerDeployment, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico apiserver deployment template: %v", err)
	}
	if err := createCalicoAPIServer(apiServerDeploymentBytes, client); err != nil {
		return err
	}

	fmt.Println("[addons] Applied essential addon: calico")
	return nil
}

func createKubernetesServicesEndpoint(controlPlaneEndpoint string, client clientset.Interface) error {
	//PHASE 1: create ConfigMap for calico kubernetesServicesEndpoint
	hostPort := strings.ReplaceAll(controlPlaneEndpoint, "https://", "")
	host := strings.Split(hostPort, ":")[0]
	k8sSvcEndpoint := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes-services-endpoint",
			Namespace: metav1.NamespaceSystem,
		},
		Data: map[string]string{
			"KUBERNETES_SERVICE_HOST":       host,
			"KUBERNETES_SERVICE_PORT":       "6443",
			"KUBERNETES_SERVICE_PORT_HTTPS": "6443",
		},
	}
	// Create the ConfigMap for Calico kubernetesServicesEndpoint or update it in case it already exists
	return apiclient.CreateOrUpdateConfigMap(client, k8sSvcEndpoint)
}

func createCalicoNode(daemonSetBytes, configBytes []byte, client clientset.Interface) error {

	//PHASE 1: create ConfigMap for calico CNI
	cniConfigMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configBytes, cniConfigMap); err != nil {
		return fmt.Errorf("unable to decode Calico CNI configmap %v", err)
	}

	// Create the ConfigMap for Calico CNI or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, cniConfigMap); err != nil {
		return err
	}

	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode calico node clusterroles %v", err)
	}

	// Create the ClusterRoles for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode calico node clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode calico node serviceAccount %v", err)
	}

	// Create the ConfigMap for CoreDNS or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create calico daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode calico node daemonset %v", err)
	}

	// Create the DaemonSet for calico node or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}

func createKubeControllers(deploymentBytes []byte, client clientset.Interface) error {
	//PHASE 1: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoControllersClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterroles %v", err)
	}

	// Create the ClusterRoles for kube controllers or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoControllersClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for kube controllers or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoControllersServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode kube controllers serviceAccount %v", err)
	}

	// Create the ServiceAccount for kube controller or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 2: create kube controller deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode kube controllers deployment %v", err)
	}

	// Create the DaemonSet for calico kube controllers or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createCalicoTypha(deploymentBytes []byte, client clientset.Interface) error {

	//PHASE 1: create typha service
	LabelsAndSelector := map[string]string{"k8s-app": "calico-typha"}
	typhaService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "calico-typha",
			Namespace: metav1.NamespaceSystem,
			Labels:    LabelsAndSelector,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					// we should not change this port name to others, because calico-felix use this port name to discovery calico-typha instances
					// https://github.com/projectcalico/calico/blob/v3.23.1/felix/daemon/daemon.go#L334
					// https://github.com/projectcalico/calico/blob/v3.23.1/typha/pkg/discovery/discovery.go#L100
					Name:       "calico-typha",
					Port:       5473,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromString("calico-typha"),
				},
			},
			Selector: LabelsAndSelector,
		},
	}
	if err := apiclient.CreateOrUpdateService(client, typhaService); err != nil {
		return fmt.Errorf("unable to create calico-typha service %v", err)
	}
	//PHASE 2: create calico-typha deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode calico-typha deployment %v", err)
	}

	// Create the deployment for calico kube controllers or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createCalicoAPIServer(deploymentBytes []byte, client clientset.Interface) error {
	//PHASE 1: create typha service
	LabelsAndSelector := map[string]string{"k8s-app": "calico-apiserver"}
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "calico-apiserver",
			Namespace: metav1.NamespaceSystem,
			Labels:    LabelsAndSelector,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "apiserver",
					Port:       443,
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromInt(5443),
				},
			},
			Selector: LabelsAndSelector,
		},
	}
	if err := apiclient.CreateOrUpdateService(client, svc); err != nil {
		return fmt.Errorf("unable to create calico-typha deployment %v", err)
	}
	//PHASE 2: create calico apiserver apiservice with calico-bootstraper
	//PHASE 3: create calico apiserver networkpolicy
	protocol := v1.ProtocolTCP
	port := intstr.FromInt(5443)
	networkPoilcy := &networking.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "calico-apiserver",
			Namespace: metav1.NamespaceSystem,
		},
		Spec: networking.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: LabelsAndSelector,
			},
			Ingress: []networking.NetworkPolicyIngressRule{
				networking.NetworkPolicyIngressRule{
					Ports: []networking.NetworkPolicyPort{
						networking.NetworkPolicyPort{
							Protocol: &protocol,
							Port:     &port,
						},
					},
				},
			},
		},
	}
	if _, err := client.NetworkingV1().NetworkPolicies(metav1.NamespaceSystem).Create(context.TODO(), networkPoilcy, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("unable to create calico apiserver networkpolicy %v", err)
	}

	//PHASE 4: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(APIServerClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode  calico apiserver clusterroles %v", err)
	}

	// Create the ClusterRoles for  calico apiserver or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(APIServerClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode calico apiserver clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for  calico apiserver or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	clusterRolesBindingDelegator := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(APIServerClusterRoleBindingDelegator), clusterRolesBindingDelegator); err != nil {
		return fmt.Errorf("unable to decode calico apiserver clusterrolebindings %v", err)
	}
	// Create the ClusterRoleBindings for  calico apiserver or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBindingDelegator); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(APIServerServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode calico apiserver serviceAccount %v", err)
	}

	// Create the ServiceAccount for  calico apiserver or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 5: create calico apiserver deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode calico apiserver deployment %v", err)
	}

	// Create the deployment for calico apiserver or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createCalicoBootstraper(defaultSubnet, imageRepository string, client clientset.Interface) error {
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(BootstraperClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode calico bootstraper clusterroles %v", err)
	}

	// Create the ClusterRoles for calico bootstraper or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(BootstraperClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode calico bootstraper clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for calico bootstraper or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	//PHASE 2: Create the ServiceAccount for calico bootstraper or update it in case it already exists
	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(BootstraperServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode calico bootstraper serviceAccount %v", err)
	}

	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	bootstraperJobBytes, err := kubeadmutil.ParseTemplate(BootstraperJob, struct{ ImageRepository, Version, PodSubnet string }{
		ImageRepository: imageRepository,
		Version:         calico.BootstraperVersion,
		PodSubnet:       defaultSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing bootstraper job template: %v", err)
	}
	//PHASE 2 : create Job to configure calico ip pool
	job := &batch.Job{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), bootstraperJobBytes, job); err != nil {
		return fmt.Errorf("unable to decode bootstraper Job %v", err)
	}
	return apiclient.CreateOrUpdateJob(client, job)
}
