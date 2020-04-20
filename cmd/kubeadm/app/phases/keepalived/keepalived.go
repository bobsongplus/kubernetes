/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package keepalived

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"runtime"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	"github.com/pkg/errors"
	"k8s.io/klog"
)

func GenerateKeepalivedConfig(lbs []string, vip, keepalivedPath string) error {
	if host, _, err := kubeadmutil.ParseHostPort(vip); err == nil {
		vip = host
	} else {
		return errors.Wrapf(err, "error parsing cluster controlPlaneEndpoint %q", vip)
	}

	hostIP, interf, err := ChooseHostInterface()
	if err != nil {
		return fmt.Errorf("failed to select a host interface: %v", err)
	}
	nodes := make([]string, 0)
	for _, lb := range lbs {
		if lb == hostIP.String() {
			continue
		}
		nodes = append(nodes, lb)
	}
	kpalived := keepalived{
		HostIP:    hostIP.String(),
		VIP:       vip,
		Peers:     nodes,
		Interface: interf,
	}
	t, err := template.New("keepalived").Parse(keepalivedTmpl)
	if err != nil {
		klog.Error(err)
		return err
	}
	bb := new(bytes.Buffer)
	if err := t.Execute(bb, kpalived); err != nil {
		klog.Error(err)
		return err
	}
	cfgBytes, err := ioutil.ReadAll(bb)
	if err != nil {
		klog.Error(err)
		return err
	}
	if err := ioutil.WriteFile(keepalivedPath, cfgBytes, 0644); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

func CreateLocalKeepalivedStaticPodManifestFile(cfg *kubeadmapi.InitConfiguration, keepalivedManifestFile string) error {
	keepalivedPodBytes, err := kubeadmutil.ParseTemplate(keepalivedManifest, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         version,
	})
	if err != nil {
		return fmt.Errorf("error parsing haproxy  template error: %v", err)
	}
	if err := ioutil.WriteFile(keepalivedManifestFile, keepalivedPodBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write haproxy static pod to the file %q", keepalivedManifestFile)
	}
	return nil

}
