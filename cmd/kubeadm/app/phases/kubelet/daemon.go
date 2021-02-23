/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package kubelet

import (
	"bytes"
	"fmt"
	"runtime"

	"k8s.io/klog"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"os"
	"path/filepath"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"
)

var (
	kubeletServicePath     = "/etc/systemd/system"
	ServiceName            = "kubelet"
	ConfigName             = "10-kubeadm.conf"
	kubeletServiceConfPath = kubeletServicePath + "/" + ServiceName + ".service.d"
)

func TryInstallKubelet(cfg *kubeadmapi.ClusterConfiguration) error {
	// PHASE 1: Write Kubelet Service to /etc/systemd/system/kubelet.service
	err := writeKubeletService(cfg.ImageRepository, cfg.KubernetesVersion)
	if err != nil {
		fmt.Println("[kubelet-install] Write kubelet service to /etc/systemd/system/kubelet.service failed.")
		return err
	}
	// PHASE 2: If we notice that the kubelet service is inactive, try to start it
	init, err := initsystem.GetInitSystem()
	if err != nil {
		fmt.Println("[kubelet-install] No supported init system detected, won't ensure kubelet is running.")
		return err
	} else {
		fmt.Println("[kubelet-install] Starting the kubelet service")
		if err := init.ServiceStart(ServiceName); err != nil {
			fmt.Printf("[kubelet-install] WARNING: Unable to start the kubelet service: [%v]\n", err)
			fmt.Println("[kubelet-install] WARNING: Please ensure kubelet is running manually.")
			return err
		}

		if !init.ServiceIsEnabled(ServiceName) {
			if init.ServiceEnable(ServiceName) {
				fmt.Println("[kubelet-install] kubelet service is enabled.")
			} else {
				fmt.Printf("[kubelet-install] kubelet service enabled failed: enable command: [%v]\n.", init.EnableCommand(ServiceName))
			}
		}
	}
	return nil
}

// /etc/systemd/system/kubelet.service
//TODO auto read from location, shouldn't in the code
func writeKubeletService(imageRepository, kubernetesVersion string) error {
	kubeletservice := `
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=network.target docker.service

[Service]
ExecStart=/usr/bin/kubelet
Restart=on-failure
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`
	buf := bytes.Buffer{}
	buf.WriteString(kubeletservice)
	filename := filepath.Join(kubeletServicePath, ServiceName+".service")
	writeFile(buf, filename)
	if _, err := os.Stat(kubeletServiceConfPath); os.IsNotExist(err) {
		if err := os.MkdirAll(kubeletServiceConfPath, 0755); err != nil {
			klog.Error(err)
			return err
		}
	}
	buf = bytes.Buffer{}
	buf.WriteString("[Service]\n")
	buf.WriteString("Environment=\"KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf \"\n")
	buf.WriteString("Environment=\"KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml  \"\n")
	buf.WriteString("EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env \n")
	buf.WriteString("ExecStartPre=/usr/bin/docker run --rm -v /opt/tmp:/opt/tmp   ")
	buf.WriteString(fmt.Sprintf("%s/kubelet-%s:%s", imageRepository, runtime.GOARCH, kubernetesVersion))
	buf.WriteString(" /bin/sh -c \"mkdir -p /opt/tmp/bin && cp /opt/cni/bin/* /opt/tmp/bin/ && cp /usr/bin/nsenter /opt/tmp \" \n")
	buf.WriteString("ExecStartPre=/bin/sh -c \"mkdir -p /opt/cni/bin && cp -r /opt/tmp/bin/ /opt/cni/ && cp /opt/tmp/nsenter /usr/bin/ && rm -r /opt/tmp\"\n")
	buf.WriteString("ExecStartPre=/bin/sh -c \"docker inspect kubelet >/dev/null 2>&1 && docker rm -f kubelet || true \" \n")
	buf.WriteString("ExecStart= \n")
	buf.WriteString("ExecStart=/bin/sh -c \"docker run --name kubelet --net=host --cpu-period=500000 --cpu-quota=1000000 --cpu-shares=1024 --memory=1g --privileged --pid=host -v /:/rootfs:ro ")
	buf.WriteString("-v /dev:/dev -v /var/log:/var/log:shared -v /var/lib/docker/:/var/lib/docker:rw  ")
	buf.WriteString("-v /var/lib/kubelet/:/var/lib/kubelet:shared -v /etc/kubernetes:/etc/kubernetes:ro ")
	buf.WriteString("-v /etc/cni:/etc/cni:rw -v /sys:/sys:ro -v /var/run:/var/run:rw -v /opt/cni/bin/:/opt/cni/bin/ ")
	buf.WriteString("-v /srv/kubernetes:/srv/kubernetes:ro ")
	buf.WriteString(fmt.Sprintf("%s/kubelet-%s:%s", imageRepository, runtime.GOARCH, kubernetesVersion))
	buf.WriteString(" nsenter --target=1 --mount --wd=./ -- ./kubelet")
	buf.WriteString(" $KUBELET_KUBECONFIG_ARGS  $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS --housekeeping-interval 10s\" \n")
	buf.WriteString("ExecStop=/usr/bin/docker stop kubelet \n")
	buf.WriteString("ExecStopPost=/usr/bin/docker rm -f kubelet \n")
	buf.WriteString("Restart=on-failure \n")
	buf.WriteString("StartLimitInterval=0 \n")
	buf.WriteString("RestartSec=10 \n")
	buf.WriteString("\n")
	buf.WriteString("[Install]\n")
	buf.WriteString("WantedBy=multi-user.target\n")
	buf.WriteString("\n")
	return writeFile(buf, kubeletServiceConfPath+"/"+ConfigName)
}

func writeFile(buf bytes.Buffer, fileName string) error {
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(buf.Bytes()), fileName); err != nil {
		return fmt.Errorf("[kubelet-install] failed to create kubelet file for (%q) [%v] \n", fileName, err)
	} else {
		fmt.Printf("[kubelet-install] Write kubelet configuration to %q Successfully.\n", fileName)
	}
	return nil
}
