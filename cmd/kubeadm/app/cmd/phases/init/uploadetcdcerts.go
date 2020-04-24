/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020-02-06  @author fupeng
 */

package phases

import (
	"fmt"

	"github.com/pkg/errors"

	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/options"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/copycerts"
)

// NewUploadEtcdCertsPhase returns the uploadEtcdCerts phase
func NewUploadEtcdCertsPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "upload-etcd-metric-certs",
		Short: fmt.Sprintf("Upload etcd metric certificates to %s", kubeadmconstants.KubeadmCertsSecret),
		Long:  cmdutil.MacroCommandLongDescription,
		Run:   runUploadEtcdCerts,
		InheritFlags: []string{
			options.CfgPath,
			options.UploadCerts,
		},
	}
}

func runUploadEtcdCerts(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("upload-etcd-metric-certs phase invoked with an invalid data struct")
	}

	if !data.UploadCerts() {
		fmt.Printf("[upload-etcd-metric-certs] Skipping phase. Please see --%s\n", options.UploadCerts)
		return nil
	}
	client, err := data.Client()
	if err != nil {
		return err
	}

	if len(data.CertificateKey()) == 0 {
		certificateKey, err := copycerts.CreateCertificateKey()
		if err != nil {
			return err
		}
		data.SetCertificateKey(certificateKey)
	}
	if err := copycerts.UploadEtcdMetricClientCerts(client, data.Cfg()); err != nil {
		return errors.Errorf("can't create secret needed by prometheus")
	}
	fmt.Printf("[upload-etcd-metric-certs] create secret for prometheus metrics\n")
	return nil
}
