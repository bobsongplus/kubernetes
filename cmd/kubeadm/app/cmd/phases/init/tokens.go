/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package phases

import (
	"github.com/pkg/errors"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
)

// NewTokenAuthFilePhase creates a kubeadm workflow phase that creates /etc/kubernetes/pki/tokens.csv.
func NewTokenAuthFilePhase() workflow.Phase {
	return workflow.Phase{
		Name:   "token-auth",
		Short:  "Generates tokens.csv file necessary to kubernetes authentication",
		Long:   cmdutil.MacroCommandLongDescription,
		Hidden: true,
		Run:    runTokenAuth,
	}
}

func runTokenAuth(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("token-auth phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	var tokenSecret string
	for _, bt := range cfg.BootstrapTokens {
		if len(bt.Token.Secret) != 0 {
			tokenSecret = bt.Token.Secret
		}
		break
	}
	return controlplane.CreateTokenAuthFile(cfg.CertificatesDir, tokenSecret)
}
