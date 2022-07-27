/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package terminal

import (
	"context"
	"fmt"
	"strings"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

const (
	dockerSocket = " /var/run/docker.sock"
)

func EnsureTerminalAddon(cfg *kubeadmapi.ClusterConfiguration, client clientset.Interface) error {
	//PHASE 1: create terminal containers
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Version, RuntimePath string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         cfg.KubernetesVersion,
		RuntimePath:     getRuntimePath(client),
	})
	if err != nil {
		return fmt.Errorf("error when parsing kubectl daemonset template: %v", err)
	}
	if err := createTerminal(daemonSetBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: terminal")
	return nil
}

func createTerminal(daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create RBAC rules
	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kubectl clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for kubectl or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode kubectl serviceAccount %v", err)
	}

	// Create the ConfigMap for kubectl or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 2: create kubectl daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode kubectl daemonset %v", err)
	}

	// Create the DaemonSet for kubectl or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}
func getRuntimePath(client clientset.Interface) string {
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(map[string]string{"node-role.kubernetes.io/master": ""})).String()})
	if err != nil {
		klog.Errorf("getting cri socket error: %v ,using default docker socket: %s.", err, dockerSocket)
		return dockerSocket
	}
	if nodes == nil || nodes.Items == nil {
		klog.Errorf("getting zero master nodes,using docker socket: %s", dockerSocket)
		return dockerSocket
	}
	criSocket := nodes.Items[0].Annotations[constants.AnnotationKubeadmCRISocket]
	if strings.Contains(criSocket, "containerd") {
		return criSocket
	}
	return dockerSocket
}
