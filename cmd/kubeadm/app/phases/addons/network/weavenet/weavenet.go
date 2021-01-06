/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package weavenet

import (
	"fmt"
	"runtime"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/output/scheme"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

func CreateWeaveNetAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {

	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode weavenet clusterroles %v", err)
	}

	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		klog.Error(err)
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode weavenet clusterrolebindings %v", err)
	}
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		klog.Error(err)
		return err
	}
	role := &rbac.Role{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(Role), role); err != nil {
		return fmt.Errorf("unable to decode weavenet role %v", err)
	}
	if err := apiclient.CreateOrUpdateRole(client, role); err != nil {
		klog.Error(err)
		return err
	}
	roleBinding := &rbac.RoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(RoleBinding), roleBinding); err != nil {
		return fmt.Errorf("unable to decode weavenet rolebindings %v", err)
	}
	if err := apiclient.CreateOrUpdateRoleBinding(client, roleBinding); err != nil {
		klog.Error(err)
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode weavenet serviceAccount %v", err)
	}

	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		klog.Error(err)
		return err
	}

	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Arch, Version, PodSubnet string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
		PodSubnet:       defaultSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing weavenet daemonset template: %v", err)
	}

	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode weavenet daemonset %v", err)
	}

	// Create the DaemonSet for weavenet or update it in case it already exists
	if err := apiclient.CreateOrUpdateDaemonSet(client, daemonSet); err != nil {
		klog.Error(err)
		return err
	}
	fmt.Println("[addons] Applied essential addon: weave")
	return nil
}
