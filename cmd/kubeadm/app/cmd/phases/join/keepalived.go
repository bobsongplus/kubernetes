package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kpphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/keepalived"
)

// NewKeepAlivedPhase creates a kubeadm workflow phase that install and configure keepalived.
func NewKeepAlivedPhase() workflow.Phase {
	return workflow.Phase{
		Name:   "keepalived",
		Short:  "Install and configure keepalived, Writes keepalived service file.",
		Long:   cmdutil.MacroCommandLongDescription,
		Hidden: true,
		Run:    runKeepAlived,
	}
}

func runKeepAlived(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("keepalived phase invoked with an invalid data struct")
	}
	if data.Cfg().ControlPlane == nil {
		klog.V(1).Infoln("[keepalived] Skipping Kubernetes High Availability Cluster")
		return nil
	}
	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}
	client, err := data.ClientSet()
	if err != nil {
		klog.Error(err)
		return err
	}
	lbconfig, err := client.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "lbconfig", metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}
	loadBalances := strings.Split(lbconfig.Data["loadBalances"], ",")
	masters := strings.Split(lbconfig.Data["masters"], ",")
	vip := lbconfig.Data["vip"]
	if loadBalances == nil {
		loadBalances = masters
	}

	if err := os.MkdirAll(filepath.Join(kubeadmconstants.KubernetesDir, kpphase.DefaultKeepalivedDir), 0700); err != nil {
		return errors.Wrapf(err, "failed to create keepalived directory %q", kpphase.DefaultKeepalivedDir)
	}
	fmt.Printf("[keepalived] Creating static Pod manifest for local keepalived in manifests \n")
	keepalivedConfigPath := filepath.Join(kubeadmconstants.KubernetesDir, kpphase.DefaultKeepalivedDir, kpphase.DefaultKeepalivedConfig)
	if err := kpphase.GenerateKeepalivedConfig(loadBalances, vip, keepalivedConfigPath); err != nil {
		return errors.Wrapf(err, "failed to create keepalived config %q", keepalivedConfigPath)
	}
	keepalivedManifestFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.ManifestsSubDirName, "keepalived.yaml")
	if err := kpphase.CreateLocalKeepalivedStaticPodManifestFile(cfg, keepalivedManifestFile); err != nil {
		return errors.Wrapf(err, "failed to create keepalived manifest %q", keepalivedManifestFile)
	}
	return nil
}
