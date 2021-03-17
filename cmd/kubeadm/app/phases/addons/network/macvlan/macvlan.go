/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-02-05  @author weiwei@tenxcloud.com
 */
package macvlan

import (
	"fmt"
	"runtime"

	apps "k8s.io/api/apps/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

// macvlan + dhcp daemon （no subnet）

// macvlan + whereabout  (need subnet)

func CreateMacVlanAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 2: create dhcp containers
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})

	if err != nil {
		return fmt.Errorf("error when parsing dhcp daemonset template: %v", err)
	}

	if err := createDHCPDaemon(daemonSetBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: dhcp-daemon")
	return nil
}

func createDHCPDaemon(daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create dhcp  daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode dhcp daemon daemonset %v", err)
	}

	// Create the DaemonSet for flannel or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}