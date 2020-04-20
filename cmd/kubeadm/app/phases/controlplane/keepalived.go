package controlplane

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"
)

var (
	KeepAlived        = "/etc/systemd/system/keepalived.service"
	ServiceName       = "keepalived"
	KeepAlivedEnvPath = "/etc/kubernetes"
	KeepAlivedEnv     = KeepAlivedEnvPath + "/keepalived.env"
)

func TryInstallKeepAlived(imageRepository string) error {
	// PHASE 1: Write keepalived Service to /etc/systemd/system/keepalived.service
	if err := writeKeepAlivedService(imageRepository, constants.KeepAlivedVersion); err != nil {
		fmt.Println("[keepalived] Write keepalived service to /etc/systemd/system/keepalived.service failed.")
		return err
	}
	// PHASE 2: If we notice that the keepalived service is inactive, try to start it
	init, err := initsystem.GetInitSystem()
	if err != nil {
		fmt.Println("[keepalived] No supported init system detected, won't ensure keepalived is running.")
		return err
	} else {
		fmt.Println("[keepalived] Starting the keepalived service")
		if err := init.ServiceStart(ServiceName); err != nil {
			fmt.Printf("[keepalived] WARNING: Unable to start the keepalived service: [%v]\n", err)
			fmt.Println("[keepalived] WARNING: Please ensure keepalived is running manually.")
			return err
		} else {
			if !init.ServiceIsEnabled(ServiceName) {
				init.EnableCommand(ServiceName)
				//fmt.Println("[keepalived] keepalived service is enabled.")
			}
		}
	}
	return nil
}

// 2. generate keepalived.service
// /etc/systemd/system/keepalived.service
func writeKeepAlivedService(imageRepository, KeepAlivedVersion string) error {
	buf := bytes.Buffer{}
	buf.WriteString("[Unit] \n")
	buf.WriteString("Description=LVS and VRRP High Availability Monitor \n")
	buf.WriteString("Documentation=https://www.keepalived.org \n")
	buf.WriteString("After=network-online.target docker.service \n")
	buf.WriteString(" \n")
	buf.WriteString("[Service] \n")
	buf.WriteString("ExecStartPre=/bin/bash -c \"docker inspect keepalived >/dev/null 2>&1 && docker rm -f keepalived || true \" \n")
	buf.WriteString("ExecStart=/bin/bash -c \"docker run --name keepalived --net=host --privileged  ")
	buf.WriteString(fmt.Sprintf("--env-file %s ", KeepAlivedEnv))
	buf.WriteString(fmt.Sprintf("%s/keepalived:%s", imageRepository, KeepAlivedVersion))
	buf.WriteString(" \" \n")
	buf.WriteString("ExecStop=/usr/bin/docker stop keepalived \n")
	buf.WriteString("ExecStopPost=/usr/bin/docker rm -f keepalived \n")
	buf.WriteString("Restart=on-failure \n")
	buf.WriteString("StartLimitInterval=0 \n")
	buf.WriteString("RestartSec=10 \n")
	buf.WriteString(" \n")
	buf.WriteString("[Install] \n")
	buf.WriteString("WantedBy=multi-user.target \n")
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(buf.Bytes()), KeepAlived); err != nil {
		return fmt.Errorf("failed to create %s.service file for [%v] \n", ServiceName, err)
	} else {
		fmt.Printf("[keepalived] Write %s service to %s Successfully.\n", ServiceName, KeepAlived)
	}
	return nil
}

// 3. generate keepalived.env
func GenerateKeepAlivedEnv(envs map[string]string) error {
	if len(envs) == 0 {
		fmt.Println("[keepalived] KeepAlivedEnv is empty.")
		return fmt.Errorf("KeepAlivedEnv is empty")
	}
	if _, err := os.Stat(KeepAlivedEnvPath); os.IsNotExist(err) {
		if err := os.MkdirAll(KeepAlivedEnvPath, 0755); err != nil {
			klog.Error(err)
			return err
		}
	}
	buf := bytes.Buffer{}
	for k, v := range envs {
		buf.WriteString(fmt.Sprintf("%s=%s", k, v))
		buf.WriteString("\n")
	}
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(buf.Bytes()), KeepAlivedEnv); err != nil {
		return fmt.Errorf("[keepalived] Failed to create %s file for [%v] \n", KeepAlivedEnv, err)
	} else {
		fmt.Printf("[keepalived] Write %s file Successfully.\n", KeepAlivedEnv)
	}
	return nil
}

// UploadEnv uploads the keepalived envs needed to init a new control plane.
func UploadEnvs(client clientset.Interface) error {
	//1. loadEnvFile
	envs, err := loadEnvFile(KeepAlivedEnv)
	if err != nil {
		return err
	}
	//2. uploadEnvs
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        constants.KeepAlivedConfigConfigMap,
			Namespace:   metav1.NamespaceSystem,
			Annotations: envs,
		},
	}
	return apiclient.CreateOrUpdateConfigMap(client, configMap)
}

// DownloadEnvs downloads the keepalived envs needed to join a new control plane.
func DownloadEnvs(client clientset.Interface) error {
	//1. downloadEnvs
	configMap, err := client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(context.TODO(), constants.KeepAlivedConfigConfigMap, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("[keepalived] configMap %s does not exist", constants.KeepAlivedConfigConfigMap)
		return errors.Wrapf(err, "configMap %s does not exist", constants.KeepAlivedConfigConfigMap)
	}
	//2. dumpEnvsToFile
	return GenerateKeepAlivedEnv(configMap.Annotations)
}

func loadEnvFile(fileName string) (map[string]string, error) {
	envs := make(map[string]string)
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		items := strings.Split(strings.TrimSpace(string(line)), "=")
		envs[items[0]] = items[1]
	}
	return envs, nil
}
