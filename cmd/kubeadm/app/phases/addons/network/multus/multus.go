package multus

import (
	"fmt"
	"runtime"
	"strings"

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

/*
  1. create network-attachment-definitions
  2. copy all cni conf to /etc/cni/multus/net.d
  3. create cr
*/

/* ideas:
1. multus daemonset create all cni conf
   defacts: all daemon would create crd/cr
2. multus job
   defacts:
3. multus controller
*/

func CreateMultusAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	deploymentBytes, err := kubeadmutil.ParseTemplate(Deployment, struct{ ImageRepository, Arch, Version, Plugins string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         MultusControllerVersion,
		Plugins:         cfg.Networking.Plugin,
	})
	if err := createMultusController(deploymentBytes, client); err != nil {
		return err
	}
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         MultusVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing multus daemonset template: %v", err)
	}
	configMapBytes, err := kubeadmutil.ParseTemplate(ConfigMap, struct{ MasterPlugin string }{
		MasterPlugin: strings.Split(cfg.Networking.Plugin, ",")[0],
	})
	if err := createMultus(daemonSetBytes, configMapBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: multus")
	return nil
}

func createMultus(daemonSetBytes, configMapBytes []byte, client clientset.Interface) error {
	//PHASE 1: create ConfigMap for multus
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configMapBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode multus configmap %v", err)
	}

	// Create the ConfigMap for multus or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}
	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode multus clusterroles %v", err)
	}

	// Create the ClusterRoles for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode multus clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for multus or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode multus serviceAccount %v", err)
	}

	// Create the ConfigMap for multus or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create multus daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode multus daemonset %v", err)
	}

	// Create the DaemonSet for multus or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}

// just tricks
func createMultusController(deploymentBytes []byte, client clientset.Interface) error {
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode multus controller deployment %v", err)
	}
	// Create muluts controller or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}
