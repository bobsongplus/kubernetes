/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package phases

import (
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
)

// NewPolicyPhase creates a kubeadm workflow phase that creates kube-scheduler policy and  default podSecurityPolicy.
func NewPolicyPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "policy",
		Short: "Generates policies file necessary to kubernetes scheduler and security",
		Long:  cmdutil.MacroCommandLongDescription,
		Run:   runPolicy,
	}
}

func runPolicy(c workflow.RunData) error {
	_, client, err := getInitData(c)
	if err != nil {
		return err
	}
	if err := controlplane.CreateSchedulerPolicy(client); err != nil {
		return err
	}
	return controlplane.CreateDefaultPodSecurityPolicy(client)
}
