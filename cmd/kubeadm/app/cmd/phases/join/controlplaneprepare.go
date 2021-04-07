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
	"fmt"

	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane/haproxy"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane/keepalived"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/options"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	certsphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/copycerts"
	kubeconfigphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/kubeconfig"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
)

var controlPlanePrepareExample = cmdutil.Examples(`
	# Prepares the machine for serving a control plane
	kubeadm join phase control-plane-prepare all
`)

// NewControlPlanePreparePhase creates a kubeadm workflow phase that implements the preparation of the node to serve a control plane
func NewControlPlanePreparePhase() workflow.Phase {
	return workflow.Phase{
		Name:    "control-plane-prepare",
		Short:   "Prepare the machine for serving a control plane",
		Example: controlPlanePrepareExample,
		Phases: []workflow.Phase{
			{
				Name:           "all [api-server-endpoint]",
				Short:          "Prepare the machine for serving a control plane",
				InheritFlags:   getControlPlanePreparePhaseFlags("all"),
				RunAllSiblings: true,
			},
			newControlPlanePrepareDownloadCertsSubphase(),
			newControlPlanePrepareCertsSubphase(),
			newControlPlanePrepareCreateEncryptphase(),
			newControlPlanePrepareKubeconfigSubphase(),
			newControlPlanePrepareControlPlaneSubphase(),
			newControlPlanePrepareTokenAuthSubphase(),
			newControlPlanePrepareHAProxySubphase(),
			newControlPlanePrepareKeepalivedSubphase(),
		},
	}
}

func getControlPlanePreparePhaseFlags(name string) []string {
	var flags []string
	switch name {
	case "all":
		flags = []string{
			options.APIServerAdvertiseAddress,
			options.APIServerBindPort,
			options.CfgPath,
			options.ControlPlane,
			options.NodeName,
			options.FileDiscovery,
			options.TokenDiscovery,
			options.TokenDiscoveryCAHash,
			options.TokenDiscoverySkipCAHash,
			options.TLSBootstrapToken,
			options.TokenStr,
			options.CertificateKey,
			options.Patches,
		}
	case "download-certs":
		flags = []string{
			options.CfgPath,
			options.ControlPlane,
			options.FileDiscovery,
			options.TokenDiscovery,
			options.TokenDiscoveryCAHash,
			options.TokenDiscoverySkipCAHash,
			options.TLSBootstrapToken,
			options.TokenStr,
			options.CertificateKey,
		}
	case "certs":
		flags = []string{
			options.APIServerAdvertiseAddress,
			options.CfgPath,
			options.ControlPlane,
			options.NodeName,
			options.FileDiscovery,
			options.TokenDiscovery,
			options.TokenDiscoveryCAHash,
			options.TokenDiscoverySkipCAHash,
			options.TLSBootstrapToken,
			options.TokenStr,
		}
	case "kubeconfig":
		flags = []string{
			options.CfgPath,
			options.ControlPlane,
			options.FileDiscovery,
			options.TokenDiscovery,
			options.TokenDiscoveryCAHash,
			options.TokenDiscoverySkipCAHash,
			options.TLSBootstrapToken,
			options.TokenStr,
			options.CertificateKey,
		}
	case "control-plane":
		flags = []string{
			options.APIServerAdvertiseAddress,
			options.APIServerBindPort,
			options.CfgPath,
			options.ControlPlane,
			options.Patches,
		}
	default:
		flags = []string{}
	}
	return flags
}

func newControlPlanePrepareDownloadCertsSubphase() workflow.Phase {
	return workflow.Phase{
		Name:         "download-certs [api-server-endpoint]",
		Short:        fmt.Sprintf("Downloads certificates shared among control-plane nodes from the %s Secret", kubeadmconstants.KubeadmCertsSecret),
		Run:          runControlPlanePrepareDownloadCertsPhaseLocal,
		InheritFlags: getControlPlanePreparePhaseFlags("download-certs"),
	}
}

func newControlPlanePrepareCertsSubphase() workflow.Phase {
	return workflow.Phase{
		Name:         "certs [api-server-endpoint]",
		Short:        "Generate the certificates for the new control plane components",
		Run:          runControlPlanePrepareCertsPhaseLocal, //NB. eventually in future we would like to break down this in sub phases for each cert or add the --csr option
		InheritFlags: getControlPlanePreparePhaseFlags("certs"),
	}
}

func newControlPlanePrepareKubeconfigSubphase() workflow.Phase {
	return workflow.Phase{
		Name:         "kubeconfig [api-server-endpoint]",
		Short:        "Generate the kubeconfig for the new control plane components",
		Run:          runControlPlanePrepareKubeconfigPhaseLocal, //NB. eventually in future we would like to break down this in sub phases for each kubeconfig
		InheritFlags: getControlPlanePreparePhaseFlags("kubeconfig"),
	}
}

func newControlPlanePrepareControlPlaneSubphase() workflow.Phase {
	return workflow.Phase{
		Name:          "control-plane",
		Short:         "Generate the manifests for the new control plane components",
		Run:           runControlPlanePrepareControlPlaneSubphase, //NB. eventually in future we would like to break down this in sub phases for each component
		InheritFlags:  getControlPlanePreparePhaseFlags("control-plane"),
		ArgsValidator: cobra.NoArgs,
	}
}

func newControlPlanePrepareTokenAuthSubphase() workflow.Phase {
	return workflow.Phase{
		Name:         "token-auth",
		Short:        "Generates tokens.csv file necessary to kubernetes authentication",
		Run:          runTokenAuth,
		InheritFlags: getControlPlanePreparePhaseFlags("all"),
	}
}

func newControlPlanePrepareHAProxySubphase() workflow.Phase {
	return workflow.Phase{
		Name:         "haproxy",
		Short:        "Generates haproxy manifest for the new control plane components",
		Run:          runHAProxy,
		InheritFlags: getControlPlanePreparePhaseFlags("all"),
	}
}

func newControlPlanePrepareKeepalivedSubphase() workflow.Phase {
	return workflow.Phase{
		Name:         "keepalived",
		Short:        "Generates keepalived manifest for the new control plane components",
		Run:          runKeepalived,
		InheritFlags: getControlPlanePreparePhaseFlags("all"),
	}
}

func newControlPlanePrepareCreateEncryptphase() workflow.Phase {
	//getControlPlanePreparePhaseFlags("certs"),
	return workflow.Phase{
		Name:         "encryption",
		Short:        "Generate encryption at rest token and config file for local cluster",
		Run:          runEncryption,
		InheritFlags: []string{options.CertificatesDir},
	}
}

func runTokenAuth(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("token-auth phase invoked with an invalid data struct")
	}
	// Skip if this is not a control plane
	if data.Cfg().ControlPlane == nil {
		return nil
	}
	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}
	var tokenSecret string
	if data.Cfg().Discovery.BootstrapToken != nil {
		bts, err := bootstraptokenv1.NewBootstrapTokenString(data.Cfg().Discovery.BootstrapToken.Token)
		if err != nil {
			klog.V(3).Infoln("[token-auth] tokenSecret is nil.")
			return err
		}
		tokenSecret = bts.Secret
	}
	return controlplane.CreateTokenAuthFile(cfg.CertificatesDir, tokenSecret)
}

func runHAProxy(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("haproxy phase invoked with an invalid data struct")
	}
	if data.Cfg().ControlPlane == nil {
		return nil
	}
	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.HaproxyDirectory), 0700); err != nil {
		return errors.Wrapf(err, "failed to create haproxy directory %q", kubeadmconstants.HaproxyDirectory)
	}
	return haproxy.CreateHaproxyStaticPod(cfg)
}

func runKeepalived(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("keepalived phase invoked with an invalid data struct")
	}
	if data.Cfg().ControlPlane == nil {
		return nil
	}
	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.KeepalivedDirectory), 0700); err != nil {
		return errors.Wrapf(err, "failed to create keepalived directory %q", kubeadmconstants.KeepalivedDirectory)
	}
	return keepalived.CreateKeepalivedStaticPod(cfg)
}

func runControlPlanePrepareControlPlaneSubphase(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("control-plane-prepare phase invoked with an invalid data struct")
	}

	// Skip if this is not a control plane
	if data.Cfg().ControlPlane == nil {
		return nil
	}

	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}

	fmt.Printf("[control-plane] Using manifest folder %q\n", data.ManifestDir())

	// If we're dry-running, set CertificatesDir to default value to get the right cert path in static pod yaml
	if data.DryRun() {
		cfg.CertificatesDir = filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.DefaultCertificateDir)
	}

	for _, component := range kubeadmconstants.ControlPlaneComponents {
		fmt.Printf("[control-plane] Creating static Pod manifest for %q\n", component)
		err := controlplane.CreateStaticPodFiles(
			data.ManifestDir(),
			data.PatchesDir(),
			&cfg.ClusterConfiguration,
			&cfg.LocalAPIEndpoint,
			data.DryRun(),
			component,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func runControlPlanePrepareDownloadCertsPhaseLocal(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("download-certs phase invoked with an invalid data struct")
	}

	if data.Cfg().ControlPlane == nil {
		klog.V(1).Infoln("[download-certs] Skipping certs download")
		return nil
	}

	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}

	// If we're dry-running, download certs to tmp dir
	if data.DryRun() {
		cfg.CertificatesDir = data.CertificateWriteDir()
	}

	client, err := bootstrapClient(data)
	if err != nil {
		return err
	}

	if err := copycerts.DownloadCerts(client, cfg); err != nil {
		return errors.Wrap(err, "error downloading certs")
	}
	return nil
}

func runControlPlanePrepareCertsPhaseLocal(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("control-plane-prepare phase invoked with an invalid data struct")
	}

	// Skip if this is not a control plane
	if data.Cfg().ControlPlane == nil {
		return nil
	}

	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}

	fmt.Printf("[certs] Using certificateDir folder %q\n", cfg.CertificatesDir)

	// Generate missing certificates (if any)
	return certsphase.CreatePKIAssets(cfg)
}

func runControlPlanePrepareKubeconfigPhaseLocal(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("control-plane-prepare phase invoked with an invalid data struct")
	}

	// Skip if this is not a control plane
	if data.Cfg().ControlPlane == nil {
		return nil
	}

	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}

	fmt.Println("[kubeconfig] Generating kubeconfig files")
	fmt.Printf("[kubeconfig] Using kubeconfig folder %q\n", data.KubeConfigDir())

	// Generate kubeconfig files for controller manager, scheduler and for the admin/kubeadm itself
	// NB. The kubeconfig file for kubelet will be generated by the TLS bootstrap process in
	// following steps of the join --control-plane workflow
	if err := kubeconfigphase.CreateJoinControlPlaneKubeConfigFiles(data.KubeConfigDir(), cfg); err != nil {
		return errors.Wrap(err, "error generating kubeconfig files")
	}

	return nil
}

func runEncryption(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("control-plane-prepare phase invoked with an invalid data struct")
	}

	// Skip if this is not a control plane
	if data.Cfg().ControlPlane == nil {
		return nil
	}

	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}

	// create the new encryption token and config file (or use existing)
	return certsphase.CreateEncryptionTokenAndConfig(cfg.CertificatesDir)
}

func bootstrapClient(data JoinData) (clientset.Interface, error) {
	tlsBootstrapCfg, err := data.TLSBootstrapCfg()
	if err != nil {
		return nil, errors.Wrap(err, "unable to access the cluster")
	}
	client, err := kubeconfigutil.ToClientSet(tlsBootstrapCfg)
	if err != nil {
		return nil, errors.Wrap(err, "unable to access the cluster")
	}
	return client, nil
}
