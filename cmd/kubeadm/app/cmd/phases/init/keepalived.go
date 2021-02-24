/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package phases

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kpphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/keepalived"
)

// NewKeepAlivedPhase creates a kubeadm workflow phase that install and configure keepalived.
func NewKeepAlivedPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "keepalived",
		Short: "Generate static pod manifest file for local keepalived",
		Long:  cmdutil.MacroCommandLongDescription,
		Run:   runKeepAlived,
	}
}

func runKeepAlived(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("keepalived phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	if cfg.ControlPlaneEndpoint == "" {
		fmt.Println("[keepalived] Skipping Kubernetes High Availability Cluster")
		klog.V(1).Infoln("[keepalived] Skipping Kubernetes High Availability Cluster")
		return nil
	}
	if !data.DryRun() {
		if err := os.MkdirAll(filepath.Join(kubeadmconstants.KubernetesDir, kpphase.DefaultKeepalivedDir), 0700); err != nil {
			return errors.Wrapf(err, "failed to create keepalived directory %q", kpphase.DefaultKeepalivedDir)
		}
	} else {
		fmt.Printf("[dryrun] Would ensure that %q directory is present\n", kpphase.DefaultKeepalivedDir)
	}
	fmt.Printf("[keepalived] Creating static Pod manifest for local keepalived in %q\n", data.ManifestDir())
	var loadbalances []string
	if cfg.Masters != nil {
		loadbalances = cfg.Masters
	}
	if cfg.LoadBalances != nil {
		loadbalances = cfg.LoadBalances
	}

	keepalivedConfigPath := filepath.Join(kubeadmconstants.KubernetesDir, kpphase.DefaultKeepalivedDir, kpphase.DefaultKeepalivedConfig)
	if err := kpphase.GenerateKeepalivedConfig(loadbalances, cfg.ControlPlaneEndpoint, keepalivedConfigPath); err != nil {
		return errors.Wrapf(err, "failed to create keepalived config %q", keepalivedConfigPath)
	}
	keepalivedManifestFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.ManifestsSubDirName, "keepalived.yaml")
	if err := kpphase.CreateLocalKeepalivedStaticPodManifestFile(cfg, keepalivedManifestFile); err != nil {
		return errors.Wrapf(err, "failed to create keepalived manifest %q", keepalivedManifestFile)
	}
	return nil
}
