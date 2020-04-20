package markcontrolplane

import (
	"fmt"
	"os"

	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

// MarkWorker label the worker and sets the worker label
func MarkWorker(client clientset.Interface, nodeName string) error {
	fmt.Printf("[mark-worker] Marking the node %s as worker by adding the label \"%s=''\"\n", nodeName, constants.LabelNodeRoleWorker)
	if nodeName == "" {
		if hostname, err := os.Hostname(); err == nil {
			nodeName = hostname
		}
	}
	return apiclient.PatchNode(client, nodeName, func(n *v1.Node) {
		markWorker(n)
	})
}

func markWorker(n *v1.Node) {
	if _, ok := n.ObjectMeta.Labels[constants.LabelNodeRoleWorker]; !ok {
		n.ObjectMeta.Labels[constants.LabelNodeRoleWorker] = ""
	}
}
