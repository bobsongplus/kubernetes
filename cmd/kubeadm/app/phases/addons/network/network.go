package network

import (
	"fmt"

	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/ovn"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/weavenet"

	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/calico"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/canal"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/flannel"
)

const (
	// For backwards compatible, leave it empty if unset
	Calico string = "calico"
	// If nothing exists at the given path, an empty directory will be created there
	// as needed with file mode 0755, having the same group and ownership with Kubelet.
	Flannel string = "flannel"
	// A directory must exist at the given path
	Canal string = "canal"
	// If nothing exists at the given path, an empty file will be created there
	// as needed with file mode 0644, having the same group and ownership with Kubelet.
	Macvlan string = "macvlan"

	Cilium string = "cilium"

	Ovn string = "ovn"

	WeaveNet string = "weavenet"

	Multus string = "multus"
)

func EnsureNetworkAddons(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {

	switch cfg.Networking.Plugin {
	case Calico:
		if err := calico.CreateCalicoAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup calico addon: %v", err)
		}
	case Flannel:
		if err := flannel.CreateFlannelAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup flannel addon: %v", err)
		}
	case Canal:
		if err := canal.CreateCanalAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup canal addon: %v", err)
		}
	case Ovn:
		if err := ovn.CreateOvnAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup ovn addon: %v", err)
		}
	case WeaveNet:
		if err := weavenet.CreateWeaveNetAddon(cfg, client); err != nil {
			klog.Error(err)
			return fmt.Errorf("error setup weavenet addon: %v", err)
		}
	default:
		fmt.Printf("Network Plugin Setting: %s, no support, setting calico by default.\n", cfg.Networking.Plugin)
		if err := calico.CreateCalicoAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup calico addon: %v", err)
		}
	}
	return nil
}
