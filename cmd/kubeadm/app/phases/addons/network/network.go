package network

import (
	"fmt"
	"strings"

	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/calico"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/canal"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/flannel"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/multus"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/ovn"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/ovnkube"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/weavenet"
)


func EnsureNetworkAddons(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	var err error
	plugins := strings.Split(cfg.Networking.Plugin, ",")
	if len(plugins) == 1 { // single network plugin
		return pickNetworkPlugin(cfg.Networking.Plugin, cfg.Networking.PodSubnet, cfg, client)
	} else if len(plugins) == 2 { // multus network plugin
		if err = pickNetworkPlugin(plugins[0], cfg.Networking.PodSubnet, cfg, client); err != nil {
			return fmt.Errorf("init master network plugins failed ")
		}
		if err = pickNetworkPlugin(plugins[1], cfg.Networking.PodExtraSubnet, cfg, client); err != nil {
			return fmt.Errorf("init extra network plugins failed")
		}
		err = multus.CreateMultusAddon(cfg, client)
	} else {
		return fmt.Errorf("too many network plugins are not yet supported")
	}
	return err
}

func pickNetworkPlugin(networkPlugin, defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	switch networkPlugin {
	case kubeadmconstants.Calico:
		if err := calico.CreateCalicoAddon(defaultSubnet, cfg, client); err != nil {
			return fmt.Errorf("error setup calico addon: %v", err)
		}
	case kubeadmconstants.Flannel:
		if err := flannel.CreateFlannelAddon(defaultSubnet, cfg, client); err != nil {
			return fmt.Errorf("error setup flannel addon: %v", err)
		}
	case kubeadmconstants.Weave:
		if err := weavenet.CreateWeaveNetAddon(defaultSubnet, cfg, client); err != nil {
			return fmt.Errorf("error setup weave addon: %v", err)
		}
	case kubeadmconstants.Ovn:
		if err := ovn.CreateOvnAddon(defaultSubnet, cfg, client); err != nil {
			return fmt.Errorf("error setup ovn addon: %v", err)
		}
	//Deprecated
	case kubeadmconstants.OvnKube:
		if err := ovnkube.CreateOvnKubeAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup ovnkube addon: %v", err)
		}
	case kubeadmconstants.Canal:
		if err := canal.CreateCanalAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup canal addon: %v", err)
		}
	default:
		fmt.Printf("[%s] network plugin is not yet supported, setting calico by default.\n", cfg.Networking.Plugin)
		if err := calico.CreateCalicoAddon(defaultSubnet, cfg, client); err != nil {
			return fmt.Errorf("error setup calico addon: %v", err)
		}
	}
	return nil
}
