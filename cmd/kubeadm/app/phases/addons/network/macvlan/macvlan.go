/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-02-05  @author weiwei@tenxcloud.com
 */
package macvlan

import (
	"fmt"
	batch "k8s.io/api/batch/v1"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

// macvlan + dhcp daemon （no subnet）

// macvlan + whereabout  (need subnet)

type IPAMType string

const (
	DHCP IPAMType = "dhcp"

	WhereAbout IPAMType = "whereabout"
)

var defaultIPAM IPAMType = WhereAbout

func CreateMacVlanAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {

	if defaultIPAM == DHCP {
		fmt.Println("[addons] Applied essential addon: macvlan with dhcp")
		return createDHCPDaemon(cfg, client)
	} else {
		daemonSetBytes, err := kubeadmutil.ParseTemplate(WhereAboutsDaemonSet, struct{ ImageRepository, Version string }{
			ImageRepository: cfg.GetControlPlaneImageRepository(),
			Version:         WhereAboutsVersion,
		})
		if err != nil {
			return fmt.Errorf("error when parsing whereabouts daemonset template: %v", err)
		}
		if err := createWhereAboutsDaemonSet(daemonSetBytes, client); err != nil {
			return err
		}

		cronJobBytes, err := kubeadmutil.ParseTemplate(WhereAboutsReconciler, struct{ ImageRepository, Version string }{
			ImageRepository: cfg.GetControlPlaneImageRepository(),
			Version:         WhereAboutsVersion,
		})
		if err != nil {
			return fmt.Errorf("error when parsing whereabouts cronjob template: %v", err)
		}

		jobBytes, err := kubeadmutil.ParseTemplate(WhereAboutsJob, struct{ ImageRepository, PodSubnet, Version string }{
			ImageRepository: cfg.GetControlPlaneImageRepository(),
			PodSubnet:       defaultSubnet,
			Version:         WhereAboutsBootstrapterVersion,
		})
		if err != nil {
			return fmt.Errorf("error when parsing whereabouts job template: %v", err)
		}
		if err := createWhereAboutsJobs(cronJobBytes, jobBytes, client); err != nil {
			return err
		}
		fmt.Println("[addons] Applied essential addon: macvlan with whereabout")
	}
	return nil
}

func createWhereAboutsDaemonSet(daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode whereabouts clusterroles %v", err)
	}
	// Create the ClusterRoles for whereabouts create or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}
	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode whereabouts clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for whereabouts create or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode whereabouts serviceAccount %v", err)
	}

	// Create the serviceaccount for whereabouts create or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create whereabouts daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode whereabouts daemonset %v", err)
	}
	// Create the DaemonSet for whereabouts or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}

func createWhereAboutsJobs(cronJobBytes, jobBytes []byte, client clientset.Interface) error {
	job := &batch.Job{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), jobBytes, job); err != nil {
		return fmt.Errorf("unable to decode whereabouts Job %v", err)
	}
	// Create the Job for whereabouts or update it in case it already exists
	if err := apiclient.CreateOrUpdateJob(client, job); err != nil {
		return fmt.Errorf("unable to create whereabouts Job %v", err)
	}

	cronJob := &batch.CronJob{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), cronJobBytes, cronJob); err != nil {
		return fmt.Errorf("unable to decode whereabouts cronJob %v", err)
	}
	// Create the CronJob for whereabouts or update it in case it already exists
	return apiclient.CreateOrUpdateCronJob(client, cronJob)
}

func createDHCPDaemon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: create dhcp containers
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DHCPDaemonSet, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         CNIPluginVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing dhcp daemonset template: %v", err)
	}
	//PHASE 2: create dhcp  daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode dhcp daemon daemonset %v", err)
	}
	// Create the DaemonSet for flannel or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}
