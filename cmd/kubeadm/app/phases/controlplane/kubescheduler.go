/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package controlplane

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

/*
 (1) MostRequestedPriority
 (2) LeastRequestedPriority
 (3) LeastRequestedPriority BalancedResourceAllocation
*/

const (

	// policy.cfg structure
	// "k8s.io/kubernetes/pkg/scheduler/apis/config"
	// https://v1-18.docs.kubernetes.io/docs/reference/scheduling/policies/
	KubeScheduler = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-scheduler
  namespace: kube-system
data:
  policy.cfg: |-
    {
        "kind" : "Policy",
        "apiVersion" : "v1",
        "predicates" : [
            {"name" : "PodFitsHostPorts"},
            {"name" : "PodFitsResources"},
            {"name" : "NoVolumeZoneConflict"},
            {"name" : "NoDiskConflict"},
            {"name" : "CheckVolumeBinding"},
            {"name" : "MatchNodeSelector"},
            {"name" : "PodToleratesNodeTaints"},
            {"name" : "MatchInterPodAffinity"},
            {"name" : "HostName"}
            ],
        "priorities" : [
            {"name" : "LeastRequestedPriority", "weight" : 1},
            {"name" : "BalancedResourceAllocation", "weight" : 1},
            {"name" : "SelectorSpreadPriority", "weight" : 1},
            {"name" : "InterPodAffinityPriority", "weight" : 1},
            {"name" : "NodeAffinityPriority", "weight" : 1},
            {"name" : "NodePreferAvoidPodsPriority", "weight" : 1},
            {"name" : "TaintTolerationPriority", "weight" : 1},
            {"name" : "ImageLocalityPriority", "weight" : 1},
            {"name" : "ServiceSpreadingPriority", "weight" : 1},
            {"name" : "EvenPodsSpreadPriority", "weight" : 1}
            ],
        "hardPodAffinitySymmetricWeight" : 10,
        "alwaysCheckAllPredicates" : false
     }`
)

// CreateSchedulerPolicy creates the kube-scheduler ConfigMap
// in kube-system namespace if it doesn't exist already
func CreateSchedulerPolicy(client clientset.Interface) error {
	//PHASE 1: create kube-scheduler ConfigMap in kube-system namespace
	configMapBytes, err := kubeadmutil.ParseTemplate(KubeScheduler, nil)
	if err != nil {
		return fmt.Errorf("error when parsing kube-scheduler configmap template: %v", err)
	}
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), configMapBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode kube-scheduler configmap %v", err)
	}
	// Create or update the ConfigMap in the kube-system namespace
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return fmt.Errorf("unable to create kube-scheduler configmap %v", err)
	}
	//Deprecated see k8s.io/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#537
	// rbacv1helpers.NewRule(Read...).Groups(legacyGroup).Resources("configmaps").RuleOrDie()
	return nil
}
