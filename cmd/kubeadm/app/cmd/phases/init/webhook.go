package phases

import (
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
)

// NewWebhookPhase creates a kubeadm workflow phase that register kubernetes cluster to Kubernetes Enterprise Platform.
func NewWebhookPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "webhook",
		Short: "register kubernetes cluster to Kubernetes Enterprise Platform",
		Long:  cmdutil.MacroCommandLongDescription,
		Run:   runWebhook,
	}
}

func runWebhook(c workflow.RunData) error {
	data, ok := c.(InitData)
	if !ok {
		return errors.New("webhook phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	if cfg.ApiServerUrl == "" || cfg.ApiServerCredential == "" {
		klog.V(1).Infoln("[webhook] Skipping register kubernetes cluster")
		return nil
	}
	return controlplane.Hook(cfg)
}
