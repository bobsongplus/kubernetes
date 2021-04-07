/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package images

import (
	"fmt"
	"runtime"

	"k8s.io/klog/v2"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmapiv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/calico"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/flannel"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/ovn"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/weavenet"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane/haproxy"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane/keepalived"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

// GetGenericImage generates and returns a platform agnostic image (backed by manifest list)
func GetGenericImage(prefix, image, tag string) string {
	return fmt.Sprintf("%s/%s-%s:%s", prefix, image, runtime.GOARCH, tag)
}

// GetKubernetesImage generates and returns the image for the components managed in the Kubernetes main repository,
// including the control-plane components and kube-proxy.
func GetKubernetesImage(image string, cfg *kubeadmapi.ClusterConfiguration) string {
	repoPrefix := cfg.GetControlPlaneImageRepository()
	kubernetesImageTag := kubeadmutil.KubernetesVersionToImageTag(cfg.KubernetesVersion)
	image = fmt.Sprintf("%s-%s", image, runtime.GOARCH)
	return GetGenericImage(repoPrefix, image, kubernetesImageTag)
}

// GetDNSImage generates and returns the image for CoreDNS.
func GetDNSImage(cfg *kubeadmapi.ClusterConfiguration) string {
	// DNS uses default image repository by default
	dnsImageRepository := cfg.ImageRepository
	// unless an override is specified
	if cfg.DNS.ImageRepository != "" {
		dnsImageRepository = cfg.DNS.ImageRepository
	}
	// Handle the renaming of the official image from "registry.k8s.io/coredns" to "registry.k8s.io/coredns/coredns
	if dnsImageRepository == kubeadmapiv1.DefaultImageRepository {
		dnsImageRepository = fmt.Sprintf("%s/coredns", dnsImageRepository)
	}
	// DNS uses an imageTag that corresponds to the DNS version matching the Kubernetes version
	dnsImageTag := constants.CoreDNSVersion

	// unless an override is specified
	if cfg.DNS.ImageTag != "" {
		dnsImageTag = cfg.DNS.ImageTag
	}

	imageName := fmt.Sprintf("%s-%s", constants.CoreDNSImageName, runtime.GOARCH)
	return GetGenericImage(dnsImageRepository, imageName, dnsImageTag)
}

// GetEtcdImage generates and returns the image for etcd
func GetEtcdImage(cfg *kubeadmapi.ClusterConfiguration) string {
	// Etcd uses default image repository by default
	etcdImageRepository := cfg.ImageRepository
	// unless an override is specified
	if cfg.Etcd.Local != nil && cfg.Etcd.Local.ImageRepository != "" {
		etcdImageRepository = cfg.Etcd.Local.ImageRepository
	}
	// Etcd uses an imageTag that corresponds to the etcd version matching the Kubernetes version
	etcdImageTag := constants.DefaultEtcdVersion
	etcdVersion, warning, err := constants.EtcdSupportedVersion(constants.SupportedEtcdVersion, cfg.KubernetesVersion)
	if err == nil {
		etcdImageTag = etcdVersion.String()
	}
	if warning != nil {
		klog.Warningln(warning)
	}
	// unless an override is specified
	if cfg.Etcd.Local != nil && cfg.Etcd.Local.ImageTag != "" {
		etcdImageTag = cfg.Etcd.Local.ImageTag
	}
	return GetGenericImage(etcdImageRepository, constants.Etcd, etcdImageTag)
}

// GetNetworkingImage generates and returns the image for networking
func GetNetworkingImage(cfg *kubeadmapi.ClusterConfiguration) []string {
	imgs := []string{}
	repoPrefix := cfg.GetControlPlaneImageRepository()
	if cfg.Networking.Plugin == constants.Calico {
		imgs = append(imgs, GetGenericImage(repoPrefix, "node", calico.Version))
		imgs = append(imgs, GetGenericImage(repoPrefix, "kube-controllers", calico.Version))
		imgs = append(imgs, GetGenericImage(repoPrefix, "cni", calico.Version))
		imgs = append(imgs, GetGenericImage(repoPrefix, "ctl", calico.Version))
	} else if cfg.Networking.Plugin == constants.Flannel {
		imgs = append(imgs, GetGenericImage(repoPrefix, "flannel", flannel.Version))
	} else if cfg.Networking.Plugin == constants.Ovn {
		imgs = append(imgs, GetGenericImage(repoPrefix, "ovn", ovn.Version))
	} else if cfg.Networking.Plugin == constants.Weave {
		imgs = append(imgs, GetGenericImage(repoPrefix, "weave-kube", weavenet.Version))
		imgs = append(imgs, GetGenericImage(repoPrefix, "weave-npc", weavenet.Version))
	}
	// HA
	if len(cfg.ControlPlaneEndpoint) != 0 {
		imgs = append(imgs, GetGenericImage(repoPrefix, "haproxy", haproxy.Version))
		imgs = append(imgs, GetGenericImage(repoPrefix, "keepalived", keepalived.Version))
	}
	return imgs
}

// GetControlPlaneImages returns a list of container images kubeadm expects to use on a control plane node
func GetControlPlaneImages(cfg *kubeadmapi.ClusterConfiguration) []string {
	imgs := []string{}

	// start with core kubernetes images
	imgs = append(imgs, GetKubernetesImage(constants.KubeAPIServer, cfg))
	imgs = append(imgs, GetKubernetesImage(constants.KubeControllerManager, cfg))
	imgs = append(imgs, GetKubernetesImage(constants.KubeScheduler, cfg))
	imgs = append(imgs, GetKubernetesImage(constants.KubeProxy, cfg))
	imgs = append(imgs, GetKubernetesImage(constants.Kubelet, cfg))
	imgs = append(imgs, GetKubernetesImage(constants.Kubectl, cfg))

	// pause is not available on the ci image repository so use the default image repository.
	imgs = append(imgs, GetPauseImage(cfg))

	// if etcd is not external then add the image as it will be required
	if cfg.Etcd.Local != nil {
		imgs = append(imgs, GetEtcdImage(cfg))
	}

	// Append the appropriate DNS images
	imgs = append(imgs, GetDNSImage(cfg))

	imgs = append(imgs, GetNetworkingImage(cfg)...)
	return imgs
}

// GetPauseImage returns the image for the "pause" container
func GetPauseImage(cfg *kubeadmapi.ClusterConfiguration) string {
	return GetGenericImage(cfg.ImageRepository, "pause", constants.PauseVersion)
}
