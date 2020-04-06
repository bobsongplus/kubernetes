/*
Copyright 2018 The Kubernetes Authors.

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

package phases

import (
	"github.com/pkg/errors"

	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/options"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	dnsaddon "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/dns"
	networkaddon "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network"
	proxyaddon "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/proxy"
	serviceproxyaddon "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/serviceproxy"
	terminaladdon "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/terminal"
)

var (
	coreDNSAddonLongDesc = cmdutil.LongDesc(`
		Install the CoreDNS addon components via the API server.
		Please note that although the DNS server is deployed, it will not be scheduled until CNI is installed.
		`)

	kubeProxyAddonLongDesc = cmdutil.LongDesc(`
		Install the kube-proxy addon components via the API server.
		`)

	terminalAddonLongDesc = cmdutil.LongDesc(`
		Installs the web-terminal addon components via the API server.
		`)

	networkAddonLongDesc = cmdutil.LongDesc(`
		Installs the network addon components via the API server.
		`)

	storageAddonLongDesc = cmdutil.LongDesc(`
		Installs the storage addon components via the API server.
		`)
)

// NewAddonPhase returns the addon Cobra command
func NewAddonPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "addon",
		Short: "Install required addons for passing Conformance tests",
		Long:  cmdutil.MacroCommandLongDescription,
		Phases: []workflow.Phase{
			{
				Name:           "all",
				Short:          "Install all the addons",
				InheritFlags:   getAddonPhaseFlags("all"),
				RunAllSiblings: true,
			},
			{
				Name:         "coredns",
				Short:        "Install the CoreDNS addon to a Kubernetes cluster",
				Long:         coreDNSAddonLongDesc,
				InheritFlags: getAddonPhaseFlags("coredns"),
				Run:          runCoreDNSAddon,
			},
			{
				Name:         "kube-proxy",
				Short:        "Install the kube-proxy addon to a Kubernetes cluster",
				Long:         kubeProxyAddonLongDesc,
				InheritFlags: getAddonPhaseFlags("kube-proxy"),
				Run:          runKubeProxyAddon,
			},
			{
				Name:         "terminal",
				Short:        "Installs the web-terminal addon to a Kubernetes cluster",
				Long:         terminalAddonLongDesc,
				InheritFlags: getAddonPhaseFlags("terminal"),
				Run:          runTerminalAddon,
			},
			{
				Name:         "network",
				Short:        "Installs the network addon to a Kubernetes cluster",
				Long:         networkAddonLongDesc,
				InheritFlags: getAddonPhaseFlags("network"),
				Run:          runNetworkAddon,
			},
			{
				Name:         "storage",
				Short:        "Installs the storage addon to a Kubernetes cluster",
				Long:         storageAddonLongDesc,
				InheritFlags: getAddonPhaseFlags("storage"),
				Run:          runStorageAddon,
			},
		},
	}
}

func getInitData(c workflow.RunData) (*kubeadmapi.InitConfiguration, clientset.Interface, error) {
	data, ok := c.(InitData)
	if !ok {
		return nil, nil, errors.New("addon phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	client, err := data.Client()
	if err != nil {
		return nil, nil, err
	}
	return cfg, client, err
}

// runCoreDNSAddon installs CoreDNS addon to a Kubernetes cluster
func runCoreDNSAddon(c workflow.RunData) error {
	cfg, client, err := getInitData(c)
	if err != nil {
		return err
	}
	return dnsaddon.EnsureDNSAddon(&cfg.ClusterConfiguration, client)
}

// runKubeProxyAddon installs KubeProxy addon to a Kubernetes cluster
func runKubeProxyAddon(c workflow.RunData) error {
	cfg, client, err := getInitData(c)
	if err != nil {
		return err
	}
	return proxyaddon.EnsureProxyAddon(&cfg.ClusterConfiguration, &cfg.LocalAPIEndpoint, client)
}

// runNetworkAddon installs network addon to a Kubernetes cluster
func runTerminalAddon(c workflow.RunData) error {
	cfg, client, err := getInitData(c)
	if err != nil {
		return err
	}
	return terminaladdon.EnsureTerminalAddon(&cfg.ClusterConfiguration, client)
}

// runNetworkAddon installs network addon to a Kubernetes cluster
func runNetworkAddon(c workflow.RunData) error {
	cfg, client, err := getInitData(c)
	if err != nil {
		return err
	}
	err = serviceproxyaddon.EnsureServiceProxyAddon(&cfg.ClusterConfiguration, client)
	if err != nil {
		return err
	}
	return networkaddon.EnsureNetworkAddons(cfg, client)
}

// runStorageAddon installs storage addon to a Kubernetes cluster
func runStorageAddon(c workflow.RunData) error {
	//TODO: FIXME
	return nil
}

func getAddonPhaseFlags(name string) []string {
	flags := []string{
		options.CfgPath,
		options.KubeconfigPath,
		options.KubernetesVersion,
		options.ImageRepository,
	}
	if name == "all" || name == "kube-proxy" {
		flags = append(flags,
			options.APIServerAdvertiseAddress,
			options.ControlPlaneEndpoint,
			options.APIServerBindPort,
			options.NetworkingPodSubnet,
		)
	}
	if name == "all" || name == "coredns" {
		flags = append(flags,
			options.FeatureGatesString,
			options.NetworkingDNSDomain,
			options.NetworkingServiceSubnet,
		)
	}
	return flags
}
