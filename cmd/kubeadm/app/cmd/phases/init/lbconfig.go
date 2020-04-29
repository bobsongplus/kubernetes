/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package phases

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

// NewLbconfigPhase creates a kubeadm workflow phase that register kubernetes cluster to Kubernetes Enterprise Platform.
func NewlbConfigPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "config",
		Short: "generate high avaliability configmap in kube-system",
		Long:  cmdutil.MacroCommandLongDescription,
		Run:   runlbConfig,
	}
}

func runlbConfig(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("lbconfig phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	if cfg.ControlPlaneEndpoint == "" {
		klog.Info("[lbconfig] skip creating in kube-system, because configPlaneEndpoint is empty")
		return nil
	}
	vip := cfg.ControlPlaneEndpoint
	if host, _, err := kubeadmutil.ParseHostPort(cfg.ControlPlaneEndpoint); err == nil {
		vip = host
	} else {
		return errors.Wrapf(err, "error parsing cluster controlPlaneEndpoint %q", vip)
	}
	config := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "lbconfig",
		},
		Data: map[string]string{
			"vip":          vip,
			"loadBalances": strings.Join(cfg.LoadBalances, ","),
			"masters":      strings.Join(cfg.Masters, ","),
		},
	}

	client, err := data.Client()
	if err != nil {
		klog.Error(err)
		return err
	}
	if _, err := client.CoreV1().ConfigMaps("kube-system").Create(context.TODO(), config, metav1.CreateOptions{}); err != nil {
		klog.Error(err)
		return errors.Wrapf(err, "lbconfig phase creating configmap in kube-system error")
	}

	return nil
}
