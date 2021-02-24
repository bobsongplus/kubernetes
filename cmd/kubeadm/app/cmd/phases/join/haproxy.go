/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	haproxyphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/haproxy"
)

func NewHaproxyPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "haproxy",
		Short: "Generate static pod manifest file for local haproxy",
		Long:  cmdutil.MacroCommandLongDescription,
		Phases: []workflow.Phase{
			newHaproxyLocalSubPhase(),
		},
	}
}
func newHaproxyLocalSubPhase() workflow.Phase {
	phase := workflow.Phase{
		Name:  "local",
		Short: "Generate the static Pod manifest file for a local, single-node local haproxy instance",
		Run:   runHaproxyPhaseLocal(),
	}
	return phase
}

// func runHaproxyPhaseLocal(c workflow.RunData) error {
func runHaproxyPhaseLocal() func(c workflow.RunData) error {
	return func(c workflow.RunData) error {
		data, ok := c.(JoinData)
		if !ok {
			return errors.New("haproxy phase invoked with an invalid data struct")
		}
		if data.Cfg().ControlPlane == nil {
			klog.V(1).Infoln("[haproxy] Skipping Kubernetes High Availability Cluster")
			return nil
		}

		cfg, err := data.InitCfg()
		if err != nil {
			return err
		}
		client, err := data.ClientSet()
		if err != nil {
			klog.Error(err)
			return err
		}
		lbconfig, err := client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(context.TODO(), "lbconfig", metav1.GetOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}

		var loadBalances []string

		if lbconfig.Data["masters"] != "" {
			loadBalances = strings.Split(lbconfig.Data["masters"], ",")
		}
		if lbconfig.Data["loadBalances"] != "" {
			loadBalances = strings.Split(lbconfig.Data["loadBalances"], ",")
		}

		if loadBalances != nil {
			if err := os.MkdirAll(filepath.Join(kubeadmconstants.KubernetesDir, haproxyphase.DefaultHaproxyDir), 0700); err != nil {
				return errors.Wrapf(err, "failed to create haproxy directory %q", haproxyphase.DefaultHaproxyDir)
			}
			fmt.Printf("[haproxy] Creating static Pod manifest for local haproxy in manifests\n")

			// save haproxy config into /etc/kubernetes/haproxy directory
			haproxyConfigPath := filepath.Join(kubeadmconstants.KubernetesDir, haproxyphase.DefaultHaproxyDir, haproxyphase.DefaultHaproxyConfig)
			if err := haproxyphase.CreateHaproxyConfig(cfg.LocalAPIEndpoint.BindPort, loadBalances, haproxyConfigPath); err != nil {
				return errors.Wrapf(err, "failed to create haproxy config %q", haproxyConfigPath)
			}
			// create haproxy manifest
			haproxyManifestFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.ManifestsSubDirName, haproxyphase.HaproxyManifestName)
			if err := haproxyphase.CreateLocalHaproxyStaticPodManifestFile(cfg, haproxyManifestFile); err != nil {
				return errors.Wrapf(err, "failed to create haproxy manifest %q", haproxyManifestFile)
			}
		}
		return nil
	}
	return nil
}
