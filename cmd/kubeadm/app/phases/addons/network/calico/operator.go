package calico

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"

	"sigs.k8s.io/yaml"
)

func CreateCalicoOperatorAddon(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, client clientset.Interface, dynamic dynamic.Interface) error {
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
	if err := createCalicoComponents(defaultSubnet, cfg, dynamic); err != nil {
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

func createCalicoComponents(defaultSubnet string, cfg *kubeadmapi.InitConfiguration, dynamic dynamic.Interface) error {
	// 1. calico installation
	var calicoGVR = schema.GroupVersionResource{Group: "operator.tigera.io", Version: "v1", Resource: "installations"}
	installation := &unstructured.Unstructured{}
	//PHASE 1: create calico operator
	imageRepository := strings.Split(cfg.GetControlPlaneImageRepository(), "/")
	InstallationBytes, err := kubeadmutil.ParseTemplate(Installation, struct{ Registry, ImagePath, PodSubnet string }{
		Registry:  imageRepository[0],
		ImagePath: imageRepository[1],
		PodSubnet: defaultSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico installation template: %v", err)
	}
	if err := yaml.Unmarshal(InstallationBytes, installation); err != nil {
		return fmt.Errorf("can't unmarshal YAML: %v", err)
	}
	if _, err := dynamic.Resource(calicoGVR).Create(context.TODO(), installation, metav1.CreateOptions{}); err != nil {
		return err
	}
	// 2 calico apiserver
	var apiserverGVR = schema.GroupVersionResource{Group: "operator.tigera.io", Version: "v1", Resource: "apiservers"}
	apiserver := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(APIServer), apiserver); err != nil {
		return fmt.Errorf("can't unmarshal YAML: %v", err)
	}
	if _, err := dynamic.Resource(apiserverGVR).Create(context.TODO(), apiserver, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
