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

package phases

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/options"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeletphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/kubelet"
)

var (
	kubeletInstallPhaseExample = cmdutil.Examples(`
		# Writes a dynamic environment file with kubelet flags from a InitConfiguration file.
		kubeadm init phase kubelet-install --config config.yaml
		`)

	kubeletStartPhaseExample = cmdutil.Examples(`
		# Writes a dynamic environment file with kubelet flags from a InitConfiguration file.
		kubeadm init phase kubelet-start --config config.yaml
		`)
)

// NewKubeletPhase creates a kubeadm workflow phase that install and (re)start kubelet on a node.
func NewKubeletPhase() workflow.Phase {
	phase := workflow.Phase{
		Name:  "kubelet",
		Short: "Install and Writes kubelet settings and (re)starts the kubelet",
		Long:  cmdutil.MacroCommandLongDescription,
		Phases: []workflow.Phase{
			{
				Name:    "kubelet-install",
				Short:   "Install and configure kubelet",
				Long:    "Install and configure kubelet, Writes kubelet service file.",
				Example: kubeletInstallPhaseExample,
				Run:     runKubeletInstall,
			}, {
				Name:    "kubelet-start",
				Short:   "Writes kubelet settings and (re)starts the kubelet",
				Long:    "Writes a file with KubeletConfiguration and an environment file with node specific kubelet settings, and then (re)starts kubelet.",
				Example: kubeletStartPhaseExample,
				Run:     runKubeletStart,
				InheritFlags: []string{
					options.CfgPath,
					options.NodeCRISocket,
					options.NodeName,
				},
			},
		},
	}
	return phase
}

// runKubeletInstall executes kubelet install logic.
func runKubeletInstall(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("kubelet-install phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	return kubeletphase.TryInstallKubelet(&cfg.ClusterConfiguration)
}

// runKubeletStart executes kubelet start logic.
func runKubeletStart(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("kubelet-start phase invoked with an invalid data struct")
	}

	// First off, configure the kubelet. In this short timeframe, kubeadm is trying to stop/restart the kubelet
	// Try to stop the kubelet service so no race conditions occur when configuring it
	if !data.DryRun() {
		klog.V(1).Infoln("Stopping the kubelet")
		kubeletphase.TryStopKubelet()
	}

	// Write env file with flags for the kubelet to use. We do not need to write the --register-with-taints for the control-plane,
	// as we handle that ourselves in the mark-control-plane phase
	// TODO: Maybe we want to do that some time in the future, in order to remove some logic from the mark-control-plane phase?
	if err := kubeletphase.WriteKubeletDynamicEnvFile(&data.Cfg().ClusterConfiguration, &data.Cfg().NodeRegistration, false, data.KubeletDir()); err != nil {
		return errors.Wrap(err, "error writing a dynamic environment file for the kubelet")
	}

	// Write the kubelet configuration file to disk.
	if err := kubeletphase.WriteConfigToDisk(&data.Cfg().ClusterConfiguration, data.KubeletDir()); err != nil {
		return errors.Wrap(err, "error writing kubelet configuration to disk")
	}

	// Try to start the kubelet service in case it's inactive
	if !data.DryRun() {
		fmt.Println("[kubelet-start] Starting the kubelet")
		kubeletphase.TryStartKubelet()
	}

	return nil
}
