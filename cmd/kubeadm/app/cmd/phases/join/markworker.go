package phases

import (
	"github.com/pkg/errors"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/options"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	markcontrolplanephase "k8s.io/kubernetes/cmd/kubeadm/app/phases/markcontrolplane"
)

var (
	markWorkerExample = cmdutil.Examples(`
		# Applies worker label to a specific node
		kubeadm join phase mark-worker --node-name nodeName
		`)
)

// NewMarkWorkerPhase creates a kubeadm workflow phase that implements mark-worker checks.
func NewMarkWorkerPhase() workflow.Phase {
	return workflow.Phase{
		Name:    "mark-worker",
		Short:   "Mark a node as a worker",
		Example: markWorkerExample,
		InheritFlags: []string{
			options.NodeName,
		},
		Run: runMarkWorker,
	}
}

// runMarkWorker executes mark-worker checks logic.
func runMarkWorker(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("mark-worker phase invoked with an invalid data struct")
	}
	if data.Cfg().ControlPlane != nil {
		klog.V(1).Infoln("[mark-worker] Skipping mark worker")
		return nil
	}
	client, err := bootstrapClient(data)
	if err != nil {
		return err
	}
	return markcontrolplanephase.MarkWorker(client, "")
}
