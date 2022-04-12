package calico

import (
	"fmt"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

func CreateCalicoOperatorAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: create calico operator
	operatorDeploymentBytes, err := kubeadmutil.ParseTemplate(OperatorDeployment, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         OperatorVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico operator deployment template: %v", err)
	}
	if err := createCalicoOperator(operatorDeploymentBytes, client); err != nil {
		return err
	}
	//PHASE 2: create calico Components(calico-node,calico-apiserver,typha,kube-controller)
	calicoBootstraperJobBytes, err := kubeadmutil.ParseTemplate(BootstraperJob, struct{ ImageRepository, PodSubnet, Version string }{
		ImageRepository: cfg.ImageRepository,
		PodSubnet:       defaultSubnet,
		Version:         BootstraperVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico bootstraper job template: %v", err)
	}
	if err := createCalicoBootstraper(calicoBootstraperJobBytes, client); err != nil {
		return fmt.Errorf("error when create calico & apiserver: %v", err)
	}
	fmt.Println("[addons] Applied essential addon: calico")
	return nil
}

func createCalicoOperator(deploymentBytes []byte, client clientset.Interface) error {

	//PHASE 1: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(OperatorClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterroles %v", err)
	}

	// Create the ClusterRoles for calico operator or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(OperatorClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for calico operator or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	//PHASE 2: Create the ServiceAccount for calico operator or update it in case it already exists
	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(OperatorServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode calico operator serviceAccount %v", err)
	}

	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create calico operator deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode calico operator deployment %v", err)
	}

	// Create the deploy for calico operator or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createCalicoBootstraper(JobBytes []byte, client clientset.Interface) error {
	//PHASE 1:  RBAC used with calico-operator
	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(BootstraperServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode calico bootstraper  serviceAccount %v", err)
	}
	// Create the serviceAccount for calico bootstraper  or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 2 : create job to configure calico bootstraper
	job := &batch.Job{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), JobBytes, job); err != nil {
		return fmt.Errorf("unable to decode calico bootstraper Job %v", err)
	}
	return apiclient.CreateOrUpdateJob(client, job)
}
