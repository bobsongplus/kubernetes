/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package phases

import (
	"github.com/pkg/errors"

	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/copycerts"
)

// Run join generate etcd client certificate
func NewEtcdClientCertsPhase() workflow.Phase {
	return workflow.Phase{
		Name:   "etcd-client-certs",
		Short:  "Run join generate etcd client certificate",
		Long:   "Run join generate etcd client certificate.",
		Hidden: true,
		Run:    runEtcdClientCerts,
	}
}

func runEtcdClientCerts(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("etcd-client-certs phase invoked with an invalid data struct")
	}

	if data.Cfg().ControlPlane != nil {
		klog.V(1).Infoln("[etcd-client-certs] Skipping create etcd client certs")
		return nil
	}

	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}

	client, err := bootstrapClient(data)
	if err != nil {
		return err
	}

	return copycerts.CreateEtcdClientCerts(client, cfg)
}
