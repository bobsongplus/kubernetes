package haproxy

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

func CreateHaproxyStaticPod(cfg *kubeadmapi.InitConfiguration) error {
	var port string
	if strings.IndexAny(cfg.ControlPlaneEndpoint,":") == -1 {
		port = string(cfg.LocalAPIEndpoint.BindPort)
	} else {
		port = strings.Split(cfg.ControlPlaneEndpoint,":")[1]
	}
	haproxyCfgBytes, err := kubeadmutil.ParseTemplateWithDelims(HAProxyCfg, struct{ Port string }{
		Port: port,
	})
	if err != nil {
		return fmt.Errorf("error parsing haproxy.cfg error: %v", err)
	}
	haproxyCfgFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.HaproxyDirectory, DefaultHaproxyCfg)
	if err := ioutil.WriteFile(haproxyCfgFile, haproxyCfgBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write haproxy.cfg file %s", haproxyCfgFile)
	}
	haproxyBytes, err := kubeadmutil.ParseTemplate(HAProxy, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error parsing haproxy template error: %v", err)
	}
	haproxyFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.ManifestsSubDirName, DefaultHaproxy)
	if err := ioutil.WriteFile(haproxyFile, haproxyBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write haproxy static pod to the file %s", haproxyFile)
	}
	return nil
}
