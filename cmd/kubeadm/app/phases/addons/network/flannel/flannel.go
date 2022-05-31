package flannel

import (
	"fmt"
	"strings"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

func CreateFlannelAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 2: create flannel containers
	// Generate ControlPlane Endpoints
	controlPlaneEndpoint, err := kubeadmutil.GetControlPlaneEndpoint(cfg.ControlPlaneEndpoint, &cfg.LocalAPIEndpoint)
	if err != nil {
		return err
	}
	endpoints := strings.ReplaceAll(controlPlaneEndpoint, fmt.Sprintf("%d", cfg.LocalAPIEndpoint.BindPort), fmt.Sprintf("%d", kubeadmconstants.EtcdListenClientPort))
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Version, EtcdEndPoints string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         Version,
		EtcdEndPoints:   endpoints,
	})
	if err != nil {
		return fmt.Errorf("error when parsing flannel daemonset template: %v", err)
	}

	var enableIPv4, enableIPv6, ipv4Network, ipv6Network string
	if kubeadmconstants.GetNetworkMode(defaultSubnet) == kubeadmconstants.NetworkIPV6Mode { // only ipv6
		enableIPv4 = "false"
		enableIPv6 = "true"
		ipv4Network = "172.31.0.0/16" // because enableIPv4=false whatever ipv4Network
		ipv6Network = defaultSubnet
	} else if kubeadmconstants.GetNetworkMode(defaultSubnet) == kubeadmconstants.NetworkDualStackMode { // ipv4 & ipv6
		podSubnets := strings.Split(defaultSubnet, ",")
		enableIPv4 = "true"
		enableIPv6 = "true"
		ipv4Network = podSubnets[0]
		ipv6Network = podSubnets[1]
	} else { // only ipv4
		enableIPv4 = "true"
		enableIPv6 = "false"
		ipv4Network = defaultSubnet
		ipv6Network = "fc00:a51f:f4ae:6ec4:abc4:1234:3f9c::/112" // because enableIPv6=false whatever ipv6Network
	}
	configMapBytes, err := kubeadmutil.ParseTemplate(ConfigMap, struct{ EnableIPv4, EnableIPv6, IPv4Network, IPv6Network, Backend string }{
		EnableIPv4:  enableIPv4,
		EnableIPv6:  enableIPv6,
		IPv4Network: ipv4Network,
		IPv6Network: ipv6Network,
		Backend:     "vxlan", // vxlan,wireguard,ipip,udp,host-gw,ali-vpc,aws-vpc,gce,alloc
	})
	if err != nil {
		return fmt.Errorf("error when parsing flannel configmap template: %v", err)
	}

	if err := createFlannel(daemonSetBytes, configMapBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: flannel")
	return nil
}

func createFlannel(daemonSetBytes, configBytes []byte, client clientset.Interface) error {
	//PHASE 1: create ConfigMap for flannel
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode flannel configmap %v", err)
	}

	// Create the ConfigMap for flannel or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}
	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode flannel clusterroles %v", err)
	}

	// Create the ClusterRoles for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode flannel clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for flannel or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode flannel serviceAccount %v", err)
	}

	// Create the ConfigMap for flannel or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create flannel daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode flannel daemonset %v", err)
	}

	// Create the DaemonSet for flannel or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}
