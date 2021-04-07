package keepalived

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

func CreateKeepalivedStaticPod(cfg *kubeadmapi.InitConfiguration) error {
	hostIP, interf, err := ChooseHostInterface()
	if err != nil {
		return fmt.Errorf("failed to find host interface: %v", err)
	}
	keepalivedCfgBytes, err := kubeadmutil.ParseTemplateWithDelims(KeepalivedCfg, struct{ Interface, Priority, VIP, HostIP string }{
		Interface: interf,
		Priority:  generatePriority(),
		VIP:       strings.Split(cfg.ControlPlaneEndpoint, ":")[0],
		HostIP:    hostIP.String(),
	})
	if err != nil {
		return fmt.Errorf("error parsing keepalived.conf error: %v", err)
	}
	keepalivedCfgFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.KeepalivedDirectory, DefaultKeepalivedCfg)
	if err := ioutil.WriteFile(keepalivedCfgFile, keepalivedCfgBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write keepalived.conf file %s", keepalivedCfgFile)
	}
	keepalivedBytes, err := kubeadmutil.ParseTemplate(Keepalived, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error parsing keepalived template error: %v", err)
	}
	keepalivedFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.ManifestsSubDirName, DefaultKeepalived)
	if err := ioutil.WriteFile(keepalivedFile, keepalivedBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write keepalived static pod to the file %s", keepalivedFile)
	}
	return nil
}

func generatePriority() string {
	rand.Seed(time.Now().UnixNano())
	priority := rand.Intn(255)
	if priority != 0 {
		return strconv.Itoa(priority)
	}
	return strconv.Itoa(rand.Intn(255))
}
