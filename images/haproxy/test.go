package main

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
)

var config = `
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
kubernetesversion: v1.18.0
arch: amd64
imageRepository: 192.168.1.52/k8s.gcr.io
controlPlaneEndpoint: 192.168.1.19
apiServer:
  timeoutForControlPlane: 5m0s
  certSANs: ["api.k8s.io"]
etcd:
  local:
    extraArgs:
      election-timeout: "5000"
      heartbeat-interval: "500"
      max-request-bytes: "3145728"
      quota-backend-bytes: "8589934592"
    serverCertSANs: ["api.k8s.io","192.168.1.19"]
dns:
  type: CoreDNS
Masters: ["192.168.1.27","192.168.1.28"]
LoadBalances: ["192.168.1.27"]
networking:
  Plugin: calico`

func main() {
	// clustercfg := &kubeadmapi.ClusterConfiguration{}
	// if err := yaml.Unmarshal([]byte(config), &clustercfg); err != nil {
	// 	klog.Error(err)
	// }
	// klog.Info(clustercfg.Arch)
	// klog.Info(clustercfg.KubernetesVersion)
	// klog.Info(clustercfg.Masters)
	// klog.Info(clustercfg.LoadBalances)
	clustercfg := &kubeadmapi.ClusterConfiguration{}
	// if err := yaml.Unmarshal([]byte(config), &clustercfg); err != nil {
	// Decode the bytes into the internal struct. Under the hood, the bytes will be unmarshalled into the
	// right external version, defaulted, and converted into the internal version.
	if err := runtime.DecodeInto(kubeadmscheme.Codecs.UniversalDecoder(), []byte(config), clustercfg); err != nil {
		klog.Error(err)
		return
	}
	klog.Info(clustercfg.ImageRepository)
	klog.Info(clustercfg.LoadBalances)
	klog.Infof("%v", clustercfg.Masters)
	klog.Info(clustercfg.ControlPlaneEndpoint)

}
