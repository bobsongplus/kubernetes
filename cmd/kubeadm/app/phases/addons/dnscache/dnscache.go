package dnscache

import (
	"fmt"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/componentconfigs"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/features"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	kubeproxyconfig "k8s.io/kubernetes/pkg/proxy/apis/config"
	"runtime"
)

func CreateNodeDnsCacheAddOn(cfg *kubeadmapi.ClusterConfiguration, client clientset.Interface) error {
	dnsip, err := kubeadmconstants.GetDNSIP(cfg.Networking.ServiceSubnet, features.Enabled(cfg.FeatureGates, features.IPv6DualStack))
	if err != nil {
		return err
	}
	kubeProxyConfig, ok := cfg.ComponentConfigs[componentconfigs.KubeProxyGroup]
	if !ok {
		return errors.New("no kube-proxy component config found in the active component config set")
	}
	proxy, _ := kubeProxyConfig.(*componentconfigs.KubeProxyConfig)
	var dnsServerAddress, localIP string
	if string(proxy.Config.Mode) == string(kubeproxyconfig.ProxyModeIPVS) {
		dnsServerAddress = ""
		localIP = kubeadmconstants.NodeLocalDNSAddress
	} else if string(proxy.Config.Mode) == string(kubeproxyconfig.ProxyModeIPTables) {
		dnsServerAddress = dnsip.String()
		localIP = fmt.Sprintf("%s,%s", kubeadmconstants.NodeLocalDNSAddress, dnsServerAddress)
	}
	//PHASE 1: create node dns cache configmap
	configMapBytes, err := kubeadmutil.ParseTemplate(ConfigMap, struct{ DNSDomain, LocalDNSAddress, DNSServerAddress string }{
		DNSDomain:        cfg.Networking.DNSDomain,
		LocalDNSAddress:  kubeadmconstants.NodeLocalDNSAddress,
		DNSServerAddress: dnsServerAddress,
	})
	if err != nil {
		return fmt.Errorf("error when parsing coredns cache configmap template: %v", err)
	}
	daemonSetBytes, err := kubeadmutil.ParseTemplate(CoreDnsCache, struct{ ImageRepository, Arch, Version, LocalDNSAddress, LocalIP string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         DnsCacheVersion,
		LocalDNSAddress: kubeadmconstants.NodeLocalDNSAddress,
		LocalIP:         localIP,
	})
	if err != nil {
		return fmt.Errorf("error when parsing coredns cache daemonset template: %v", err)
	}
	if err := createNodeDnsCache(daemonSetBytes, configMapBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: coredns-cache")
	return nil
}

func createNodeDnsCache(daemonSetBytes, configBytes []byte, client clientset.Interface) error {

	//PHASE 1: create ConfigMap for dns cache
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode dns cache configmap %v", err)
	}

	// Create the ConfigMap for dns cache or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}

	//PHASE 3: create dns cache daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode dns cache daemonset %v", err)
	}

	// Create the DaemonSet for calico node or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}
