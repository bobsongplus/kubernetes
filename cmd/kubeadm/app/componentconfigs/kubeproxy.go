/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package componentconfigs

import (
	clientset "k8s.io/client-go/kubernetes"
	kubeproxyconfig "k8s.io/kube-proxy/config/v1alpha1"
	netutils "k8s.io/utils/net"
	"strings"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmapiv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

const (
	// KubeProxyGroup is a pointer to the used API group name for the kube-proxy config
	KubeProxyGroup = kubeproxyconfig.GroupName

	// kubeproxyKubeConfigFileName is used during defaulting. It's here so it can be accessed from the tests.
	kubeproxyKubeConfigFileName = "/var/lib/kube-proxy/kubeconfig.conf"
)

// kubeProxyHandler is the handler instance for the kube-proxy component config
var kubeProxyHandler = handler{
	GroupVersion: kubeproxyconfig.SchemeGroupVersion,
	AddToScheme:  kubeproxyconfig.AddToScheme,
	CreateEmpty: func() kubeadmapi.ComponentConfig {
		return &KubeProxyConfig{
			configBase: configBase{
				GroupVersion: kubeproxyconfig.SchemeGroupVersion,
			},
		}
	},
	fromCluster: kubeProxyConfigFromCluster,
}

func kubeProxyConfigFromCluster(h *handler, clientset clientset.Interface, _ *kubeadmapi.ClusterConfiguration) (kubeadmapi.ComponentConfig, error) {
	return h.fromConfigMap(clientset, kubeadmconstants.KubeProxyConfigMap, kubeadmconstants.KubeProxyConfigMapKey, false)
}

// kubeProxyConfig implements the kubeadmapi.ComponentConfig interface for kube-proxy
type KubeProxyConfig struct {
	configBase
	Config kubeproxyconfig.KubeProxyConfiguration
}

func (kp *KubeProxyConfig) DeepCopy() kubeadmapi.ComponentConfig {
	result := &KubeProxyConfig{}
	kp.configBase.DeepCopyInto(&result.configBase)
	kp.Config.DeepCopyInto(&result.Config)
	return result
}

func (kp *KubeProxyConfig) Marshal() ([]byte, error) {
	return kp.configBase.Marshal(&kp.Config)
}

func (kp *KubeProxyConfig) Unmarshal(docmap kubeadmapi.DocumentMap) error {
	return kp.configBase.Unmarshal(docmap, &kp.Config)
}

func kubeProxyDefaultBindAddress(localAdvertiseAddress string) string {
	ip := netutils.ParseIPSloppy(localAdvertiseAddress)
	if ip.To4() != nil {
		return kubeadmapiv1.DefaultProxyBindAddressv4
	}
	return kubeadmapiv1.DefaultProxyBindAddressv6
}

func (kp *KubeProxyConfig) Get() interface{} {
	return &kp.Config
}

func (kp *KubeProxyConfig) Set(cfg interface{}) {
	kp.Config = *cfg.(*kubeproxyconfig.KubeProxyConfiguration)
}

func (kp *KubeProxyConfig) Default(cfg *kubeadmapi.ClusterConfiguration, localAPIEndpoint *kubeadmapi.APIEndpoint, _ *kubeadmapi.NodeRegistrationOptions) {
	const kind = "KubeProxyConfiguration"

	// The below code is necessary because while KubeProxy may be defined, the user may not
	// have defined any feature-gates, thus FeatureGates will be nil and the later insertion
	// of any feature-gates will cause a panic.
	if kp.Config.FeatureGates == nil {
		kp.Config.FeatureGates = map[string]bool{}
	}

	defaultBindAddress := kubeProxyDefaultBindAddress(localAPIEndpoint.AdvertiseAddress)
	if kp.Config.BindAddress == "" {
		kp.Config.BindAddress = defaultBindAddress
	} else if kp.Config.BindAddress != defaultBindAddress {
		warnDefaultComponentConfigValue(kind, "bindAddress", defaultBindAddress, kp.Config.BindAddress)
	}

	if kp.Config.ClusterCIDR == "" && cfg.Networking.PodSubnet != "" {
		kp.Config.ClusterCIDR = cfg.Networking.PodSubnet
	} else if cfg.Networking.PodSubnet != "" && kp.Config.ClusterCIDR != cfg.Networking.PodSubnet {
		warnDefaultComponentConfigValue(kind, "clusterCIDR", cfg.Networking.PodSubnet, kp.Config.ClusterCIDR)
	}
	// override subnet
	plugins := strings.Split(cfg.Networking.Plugin,",")
	if len(plugins) == 2 {
		if plugins[1] == kubeadmconstants.Flannel {
			kp.Config.ClusterCIDR = cfg.Networking.PodExtraSubnet
		}
	}

	if kp.Config.ClientConnection.Kubeconfig == "" {
		kp.Config.ClientConnection.Kubeconfig = kubeproxyKubeConfigFileName
	} else if kp.Config.ClientConnection.Kubeconfig != kubeproxyKubeConfigFileName {
		warnDefaultComponentConfigValue(kind, "clientConnection.kubeconfig", kubeproxyKubeConfigFileName, kp.Config.ClientConnection.Kubeconfig)
	}
}

// Mutate is NOP for the kube-proxy config
func (kp *KubeProxyConfig) Mutate() error {
	return nil
}
