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

package controlplane

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	utilsnet "k8s.io/utils/net"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/features"
	"k8s.io/kubernetes/cmd/kubeadm/app/images"
	certphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	staticpodutil "k8s.io/kubernetes/cmd/kubeadm/app/util/staticpod"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/users"
)

// CreateInitStaticPodManifestFiles will write all static pod manifest files needed to bring up the control plane.
func CreateInitStaticPodManifestFiles(manifestDir, patchesDir string, cfg *kubeadmapi.InitConfiguration, isDryRun bool) error {
	klog.V(1).Infoln("[control-plane] creating static Pod files")
	return CreateStaticPodFiles(manifestDir, patchesDir, &cfg.ClusterConfiguration, &cfg.LocalAPIEndpoint, isDryRun, kubeadmconstants.KubeAPIServer, kubeadmconstants.KubeControllerManager, kubeadmconstants.KubeScheduler)
}

// GetStaticPodSpecs returns all staticPodSpecs actualized to the context of the current configuration
// NB. this methods holds the information about how kubeadm creates static pod manifests.
func GetStaticPodSpecs(cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint) map[string]v1.Pod {
	// Get the required hostpath mounts
	mounts := getHostPathVolumesForTheControlPlane(cfg)

	// Prepare static pod specs
	staticPodSpecs := map[string]v1.Pod{
		kubeadmconstants.KubeAPIServer: staticpodutil.ComponentPod(v1.Container{
			Name:            kubeadmconstants.KubeAPIServer,
			Image:           images.GetKubernetesImage(kubeadmconstants.KubeAPIServer, cfg),
			ImagePullPolicy: v1.PullIfNotPresent,
			Command:         getAPIServerCommand(cfg, endpoint),
			VolumeMounts:    staticpodutil.VolumeMountMapToSlice(mounts.GetVolumeMounts(kubeadmconstants.KubeAPIServer)),
			LivenessProbe:   staticpodutil.LivenessProbe(staticpodutil.GetAPIServerProbeAddress(endpoint), "/livez", int(endpoint.BindPort), v1.URISchemeHTTPS),
			ReadinessProbe:  staticpodutil.ReadinessProbe(staticpodutil.GetAPIServerProbeAddress(endpoint), "/readyz", int(endpoint.BindPort), v1.URISchemeHTTPS),
			StartupProbe:    staticpodutil.StartupProbe(staticpodutil.GetAPIServerProbeAddress(endpoint), "/livez", int(endpoint.BindPort), v1.URISchemeHTTPS, cfg.APIServer.TimeoutForControlPlane),
			Resources:       staticpodutil.ComponentResources("250m"),
			Env:             kubeadmutil.GetProxyEnvVars(),
		}, mounts.GetVolumes(kubeadmconstants.KubeAPIServer),
			map[string]string{kubeadmconstants.KubeAPIServerAdvertiseAddressEndpointAnnotationKey: endpoint.String()}),
		kubeadmconstants.KubeControllerManager: staticpodutil.ComponentPod(v1.Container{
			Name:            kubeadmconstants.KubeControllerManager,
			Image:           images.GetKubernetesImage(kubeadmconstants.KubeControllerManager, cfg),
			ImagePullPolicy: v1.PullIfNotPresent,
			Command:         getControllerManagerCommand(cfg),
			VolumeMounts:    staticpodutil.VolumeMountMapToSlice(mounts.GetVolumeMounts(kubeadmconstants.KubeControllerManager)),
			LivenessProbe:   staticpodutil.LivenessProbe(staticpodutil.GetControllerManagerProbeAddress(cfg), "/healthz", kubeadmconstants.KubeControllerManagerPort, v1.URISchemeHTTPS),
			StartupProbe:    staticpodutil.StartupProbe(staticpodutil.GetControllerManagerProbeAddress(cfg), "/healthz", kubeadmconstants.KubeControllerManagerPort, v1.URISchemeHTTPS, cfg.APIServer.TimeoutForControlPlane),
			Resources:       staticpodutil.ComponentResources("200m"),
			Env:             kubeadmutil.GetProxyEnvVars(),
		}, mounts.GetVolumes(kubeadmconstants.KubeControllerManager), nil),
		kubeadmconstants.KubeScheduler: staticpodutil.ComponentPod(v1.Container{
			Name:            kubeadmconstants.KubeScheduler,
			Image:           images.GetKubernetesImage(kubeadmconstants.KubeScheduler, cfg),
			ImagePullPolicy: v1.PullIfNotPresent,
			Command:         getSchedulerCommand(cfg),
			VolumeMounts:    staticpodutil.VolumeMountMapToSlice(mounts.GetVolumeMounts(kubeadmconstants.KubeScheduler)),
			LivenessProbe:   staticpodutil.LivenessProbe(staticpodutil.GetSchedulerProbeAddress(cfg), "/healthz", kubeadmconstants.KubeSchedulerPort, v1.URISchemeHTTPS),
			StartupProbe:    staticpodutil.StartupProbe(staticpodutil.GetSchedulerProbeAddress(cfg), "/healthz", kubeadmconstants.KubeSchedulerPort, v1.URISchemeHTTPS, cfg.APIServer.TimeoutForControlPlane),
			Resources:       staticpodutil.ComponentResources("100m"),
			Env:             kubeadmutil.GetProxyEnvVars(),
		}, mounts.GetVolumes(kubeadmconstants.KubeScheduler), nil),
	}

	// Expose kube-scheduler metrics for prometheus to scrape
	//KubeScheduler := staticPodSpecs[kubeadmconstants.KubeScheduler]
	//KubeScheduler.Annotations = map[string]string{"prometheus.io/scrape": "true"}
	//port := v1.ContainerPort{Name: "scrape", ContainerPort: 10251}
	//KubeScheduler.Spec.Containers[0].Ports = []v1.ContainerPort{port}
	//staticPodSpecs[kubeadmconstants.KubeScheduler] = KubeScheduler
	return staticPodSpecs
}

// CreateStaticPodFiles creates all the requested static pod files.
func CreateStaticPodFiles(manifestDir, patchesDir string, cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint, isDryRun bool, componentNames ...string) error {
	// gets the StaticPodSpecs, actualized for the current ClusterConfiguration
	klog.V(1).Infoln("[control-plane] getting StaticPodSpecs")
	specs := GetStaticPodSpecs(cfg, endpoint)

	var usersAndGroups *users.UsersAndGroups
	var err error
	if features.Enabled(cfg.FeatureGates, features.RootlessControlPlane) {
		if isDryRun {
			fmt.Printf("[control-plane] Would create users and groups for %+v to run as non-root\n", componentNames)
		} else {
			usersAndGroups, err = staticpodutil.GetUsersAndGroups()
			if err != nil {
				return errors.Wrap(err, "failed to create users and groups")
			}
		}
	}

	// creates required static pod specs
	for _, componentName := range componentNames {
		// retrieves the StaticPodSpec for given component
		spec, exists := specs[componentName]
		if !exists {
			return errors.Errorf("couldn't retrieve StaticPodSpec for %q", componentName)
		}

		// print all volumes that are mounted
		for _, v := range spec.Spec.Volumes {
			klog.V(2).Infof("[control-plane] adding volume %q for component %q", v.Name, componentName)
		}

		if features.Enabled(cfg.FeatureGates, features.RootlessControlPlane) {
			if isDryRun {
				fmt.Printf("[control-plane] Would update static pod manifest for %q to run run as non-root\n", componentName)
			} else {
				if usersAndGroups != nil {
					if err := staticpodutil.RunComponentAsNonRoot(componentName, &spec, usersAndGroups, cfg); err != nil {
						return errors.Wrapf(err, "failed to run component %q as non-root", componentName)
					}
				}
			}
		}

		// if patchesDir is defined, patch the static Pod manifest
		if patchesDir != "" {
			patchedSpec, err := staticpodutil.PatchStaticPod(&spec, patchesDir, os.Stdout)
			if err != nil {
				return errors.Wrapf(err, "failed to patch static Pod manifest file for %q", componentName)
			}
			spec = *patchedSpec
		}

		// writes the StaticPodSpec to disk
		if err := staticpodutil.WriteStaticPodToDisk(componentName, manifestDir, spec); err != nil {
			return errors.Wrapf(err, "failed to create static pod manifest file for %q", componentName)
		}
		// write the AuditPolicyFile to disk
		if err := WriteAuditPolicyToDisk(cfg); err != nil {
			return errors.Wrapf(err, "failed to create audit policy file to disk")
		}
		klog.V(1).Infof("[control-plane] wrote static Pod manifest for component %q to %q\n", componentName, kubeadmconstants.GetStaticPodFilepath(componentName, manifestDir))
	}

	return nil
}

// getAPIServerCommand builds the right API server command from the given config object and version
func getAPIServerCommand(cfg *kubeadmapi.ClusterConfiguration, localAPIEndpoint *kubeadmapi.APIEndpoint) []string {
	defaultArguments := map[string]string{
		"advertise-address":                localAPIEndpoint.AdvertiseAddress,
		"enable-admission-plugins":         "NodeRestriction,PodSecurityPolicy,Priority",
		"service-cluster-ip-range":         cfg.Networking.ServiceSubnet,
		"service-account-key-file":         filepath.Join(cfg.CertificatesDir, kubeadmconstants.ServiceAccountPublicKeyName),
		"service-account-signing-key-file": filepath.Join(cfg.CertificatesDir, kubeadmconstants.ServiceAccountPrivateKeyName),
		"service-account-issuer":           fmt.Sprintf("https://kubernetes.default.svc.%s", cfg.Networking.DNSDomain),
		"client-ca-file":                   filepath.Join(cfg.CertificatesDir, kubeadmconstants.CACertName),
		"tls-cert-file":                    filepath.Join(cfg.CertificatesDir, kubeadmconstants.APIServerCertName),
		"tls-private-key-file":             filepath.Join(cfg.CertificatesDir, kubeadmconstants.APIServerKeyName),
		"kubelet-client-certificate":       filepath.Join(cfg.CertificatesDir, kubeadmconstants.APIServerKubeletClientCertName),
		"kubelet-client-key":               filepath.Join(cfg.CertificatesDir, kubeadmconstants.APIServerKubeletClientKeyName),
		"token-auth-file":                  filepath.Join(cfg.CertificatesDir, "tokens.csv"),
		"enable-bootstrap-token-auth":      "true",
		"secure-port":                      fmt.Sprintf("%d", localAPIEndpoint.BindPort),
		"allow-privileged":                 "true",
		"kubelet-preferred-address-types":  "InternalIP,ExternalIP,Hostname",
		// add options to configure the front proxy.  Without the generated client cert, this will never be useable
		// so add it unconditionally with recommended values
		"requestheader-username-headers":         "X-Remote-User",
		"requestheader-group-headers":            "X-Remote-Group",
		"requestheader-extra-headers-prefix":     "X-Remote-Extra-",
		"requestheader-client-ca-file":           filepath.Join(cfg.CertificatesDir, kubeadmconstants.FrontProxyCACertName),
		"requestheader-allowed-names":            "front-proxy-client",
		"proxy-client-cert-file":                 filepath.Join(cfg.CertificatesDir, kubeadmconstants.FrontProxyClientCertName),
		"proxy-client-key-file":                  filepath.Join(cfg.CertificatesDir, kubeadmconstants.FrontProxyClientKeyName),
		"watch-cache":                            "true",
		"default-watch-cache-size":               "1500",
		"event-ttl":                              "1h0m0s",
		"max-requests-inflight":                  "800",
		"max-mutating-requests-inflight":         "400",
		"kubelet-timeout":                        "5s",
		"default-not-ready-toleration-seconds":   "60",
		"default-unreachable-toleration-seconds": "60",
		"tls-cipher-suites":                      "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_128_GCM_SHA256",
		"audit-policy-file":                      filepath.Join(cfg.CertificatesDir, kubeadmconstants.AuditPolicyConfigFileName),
		"audit-log-format":                       "json",
		"audit-log-path":                         filepath.Join(kubeadmconstants.AuditVolumePath, kubeadmconstants.AuditLogFileName),
		"audit-log-maxage":                       "30",
		"audit-log-maxbackup":                    "10",
		"audit-log-maxsize":                      "100",
		"profiling":                              "false",
	}
	if certphase.UseEncryption {
		defaultArguments["encryption-provider-config"] = filepath.Join(cfg.CertificatesDir, kubeadmconstants.EncryptionConfigFileName)
	}

	command := []string{"kube-apiserver"}

	// If the user set endpoints for an external etcd cluster
	if cfg.Etcd.External != nil {
		defaultArguments["etcd-servers"] = strings.Join(cfg.Etcd.External.Endpoints, ",")

		// Use any user supplied etcd certificates
		if cfg.Etcd.External.CAFile != "" {
			defaultArguments["etcd-cafile"] = cfg.Etcd.External.CAFile
		}
		if cfg.Etcd.External.CertFile != "" && cfg.Etcd.External.KeyFile != "" {
			defaultArguments["etcd-certfile"] = cfg.Etcd.External.CertFile
			defaultArguments["etcd-keyfile"] = cfg.Etcd.External.KeyFile
		}
	} else {
		// Default to etcd static pod on localhost
		// localhost IP family should be the same that the AdvertiseAddress
		etcdLocalhostAddress := "127.0.0.1"
		if utilsnet.IsIPv6String(localAPIEndpoint.AdvertiseAddress) {
			etcdLocalhostAddress = "::1"
		}
		defaultArguments["etcd-servers"] = fmt.Sprintf("https://%s", net.JoinHostPort(etcdLocalhostAddress, strconv.Itoa(kubeadmconstants.EtcdListenClientPort)))
		defaultArguments["etcd-cafile"] = filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdCACertName)
		defaultArguments["etcd-certfile"] = filepath.Join(cfg.CertificatesDir, kubeadmconstants.APIServerEtcdClientCertName)
		defaultArguments["etcd-keyfile"] = filepath.Join(cfg.CertificatesDir, kubeadmconstants.APIServerEtcdClientKeyName)

		// Apply user configurations for local etcd
		if cfg.Etcd.Local != nil {
			if value, ok := cfg.Etcd.Local.ExtraArgs["advertise-client-urls"]; ok {
				defaultArguments["etcd-servers"] = value
			}
		}
	}

	if cfg.APIServer.ExtraArgs == nil {
		cfg.APIServer.ExtraArgs = map[string]string{}
	}
	cfg.APIServer.ExtraArgs["authorization-mode"] = getAuthzModes(cfg.APIServer.ExtraArgs["authorization-mode"])
	command = append(command, kubeadmutil.BuildArgumentListFromMap(defaultArguments, cfg.APIServer.ExtraArgs)...)

	return command
}

// getAuthzModes gets the authorization-related parameters to the api server
// Node,RBAC is the default mode if nothing is passed to kubeadm. User provided modes override
// the default.
func getAuthzModes(authzModeExtraArgs string) string {
	defaultMode := []string{
		kubeadmconstants.ModeNode,
		kubeadmconstants.ModeRBAC,
	}

	if len(authzModeExtraArgs) > 0 {
		mode := []string{}
		for _, requested := range strings.Split(authzModeExtraArgs, ",") {
			if isValidAuthzMode(requested) {
				mode = append(mode, requested)
			} else {
				klog.Warningf("ignoring unknown kube-apiserver authorization-mode %q", requested)
			}
		}

		// only return the user provided mode if at least one was valid
		if len(mode) > 0 {
			if !compareAuthzModes(defaultMode, mode) {
				klog.Warningf("the default kube-apiserver authorization-mode is %q; using %q",
					strings.Join(defaultMode, ","),
					strings.Join(mode, ","),
				)
			}
			return strings.Join(mode, ",")
		}
	}
	return strings.Join(defaultMode, ",")
}

// compareAuthzModes compares two given authz modes and returns false if they do not match
func compareAuthzModes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, m := range a {
		if m != b[i] {
			return false
		}
	}
	return true
}

func isValidAuthzMode(authzMode string) bool {
	allModes := []string{
		kubeadmconstants.ModeNode,
		kubeadmconstants.ModeRBAC,
		kubeadmconstants.ModeWebhook,
		kubeadmconstants.ModeABAC,
		kubeadmconstants.ModeAlwaysAllow,
		kubeadmconstants.ModeAlwaysDeny,
	}

	for _, mode := range allModes {
		if authzMode == mode {
			return true
		}
	}
	return false
}

// getControllerManagerCommand builds the right controller manager command from the given config object and version
func getControllerManagerCommand(cfg *kubeadmapi.ClusterConfiguration) []string {

	kubeconfigFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.ControllerManagerKubeConfigFileName)
	caFile := filepath.Join(cfg.CertificatesDir, kubeadmconstants.CACertName)

	defaultArguments := map[string]string{
		"bind-address":                     "127.0.0.1",
		"leader-elect":                     "true",
		"kubeconfig":                       kubeconfigFile,
		"authentication-kubeconfig":        kubeconfigFile,
		"authorization-kubeconfig":         kubeconfigFile,
		"client-ca-file":                   caFile,
		"requestheader-client-ca-file":     filepath.Join(cfg.CertificatesDir, kubeadmconstants.FrontProxyCACertName),
		"root-ca-file":                     caFile,
		"service-account-private-key-file": filepath.Join(cfg.CertificatesDir, kubeadmconstants.ServiceAccountPrivateKeyName),
		"cluster-signing-cert-file":        caFile,
		"cluster-signing-key-file":         filepath.Join(cfg.CertificatesDir, kubeadmconstants.CAKeyName),
		"use-service-account-credentials":  "true",
		"controllers":                      "*,bootstrapsigner,tokencleaner",
	}

	// If using external CA, pass empty string to controller manager instead of ca.key/ca.crt path,
	// so that the csrsigning controller fails to start
	if res, _ := certphase.UsingExternalCA(cfg); res {
		defaultArguments["cluster-signing-key-file"] = ""
		defaultArguments["cluster-signing-cert-file"] = ""
	}

	// Let the controller-manager allocate Node CIDRs for the Pod network.
	// Each node will get a subspace of the address CIDR provided with --pod-network-cidr.
	if cfg.Networking.PodSubnet != "" {
		defaultArguments["allocate-node-cidrs"] = "true"
		defaultArguments["cluster-cidr"] = cfg.Networking.PodSubnet
		if cfg.Networking.ServiceSubnet != "" {
			defaultArguments["service-cluster-ip-range"] = cfg.Networking.ServiceSubnet
		}
	}
	//// override subnet
	plugins := strings.Split(cfg.Networking.Plugin, ",")
	if len(plugins) == 2 {
		if plugins[1] == kubeadmconstants.Flannel {
			defaultArguments["cluster-cidr"] = cfg.Networking.PodExtraSubnet
		}
	}

	// Set cluster name
	if cfg.ClusterName != "" {
		defaultArguments["cluster-name"] = cfg.ClusterName
	}

	command := []string{"kube-controller-manager"}
	command = append(command, kubeadmutil.BuildArgumentListFromMap(defaultArguments, cfg.ControllerManager.ExtraArgs)...)

	return command
}

// getSchedulerCommand builds the right scheduler command from the given config object and version
func getSchedulerCommand(cfg *kubeadmapi.ClusterConfiguration) []string {
	kubeconfigFile := filepath.Join(kubeadmconstants.KubernetesDir, kubeadmconstants.SchedulerKubeConfigFileName)
	defaultArguments := map[string]string{
		"bind-address":              "127.0.0.1",
		"leader-elect":              "true",
		"kubeconfig":                kubeconfigFile,
		"authentication-kubeconfig": kubeconfigFile,
		"authorization-kubeconfig":  kubeconfigFile,
		"profiling":                 "false",
	}

	command := []string{"kube-scheduler"}
	command = append(command, kubeadmutil.BuildArgumentListFromMap(defaultArguments, cfg.Scheduler.ExtraArgs)...)
	return command
}

// inspired by https://github.com/kubernetes/kubernetes/blob/v1.20.5/cluster/gce/gci/configure-helper.sh#L1112
const (
	AuditPolicy = `apiVersion: audit.k8s.io/v1
kind: Policy
omitStages:
  - "RequestReceived"
rules:
  - level: None
    nonResourceURLs:
      - /healthz*
      - /version
      - /swagger*
  - level: None
    users: ["system:serviceaccount:system-monitoring:vm-operator"]
    verbs: ["update"]
    resources:
      - group: ""
        resources: ["configmaps","serviceaccounts"]
      - group: "policy"
        resources: ["podsecuritypolicies"]
      - group: "apps"
        resources: ["statefulsets"]
  - level: None
    users: ["system:serviceaccount:kube-system:nfs-provisioner"]
    verbs: ["update"]
    resources:
      - group: ""
        resources: ["configmaps"]
  - level: None
    resources:
      - group: ""
        resources: ["events"]
  - level: None
    resources:
      - group: "authorization.k8s.io"
      - group: "rbac.authorization.k8s.io"
      - group: "authentication.k8s.io"
  - level: None
    verbs: ["update","patch"]
    resources:
      - group: ""
        resources: ["nodes/status", "pods/status", "services/status", "endpoints"]
      - group: "apps"
        resources: ["statefulsets/status", "deployments/status", "daemonsets/status"]
  - level: None
    verbs: ["get", "list", "watch"]
    resources:
      - group: ""
      - group: "admissionregistration.k8s.io"
      - group: "apiextensions.k8s.io"
      - group: "apiregistration.k8s.io"
      - group: "apps"
      - group: "authentication.k8s.io"
      - group: "authorization.k8s.io"
      - group: "autoscaling"
      - group: "batch"
      - group: "certificates.k8s.io"
      - group: "extensions"
      - group: "metrics.k8s.io"
      - group: "networking.k8s.io"
      - group: "node.k8s.io"
      - group: "policy"
      - group: "rbac.authorization.k8s.io"
      - group: "scheduling.k8s.io"
      - group: "storage.k8s.io"
  - level: RequestResponse
    resources:
      - group: ""
      - group: "admissionregistration.k8s.io"
      - group: "apiextensions.k8s.io"
      - group: "apiregistration.k8s.io"
      - group: "apps"
      - group: "authentication.k8s.io"
      - group: "authorization.k8s.io"
      - group: "autoscaling"
      - group: "batch"
      - group: "certificates.k8s.io"
      - group: "extensions"
      - group: "metrics.k8s.io"
      - group: "networking.k8s.io"
      - group: "node.k8s.io"
      - group: "policy"
      - group: "rbac.authorization.k8s.io"
      - group: "settings.k8s.io"
      - group: "scheduling.k8s.io"
      - group: "storage.k8s.io"
    omitStages:
      - "RequestReceived"
`
)

func WriteAuditPolicyToDisk(cfg *kubeadmapi.ClusterConfiguration) error {
	// creates /var/log/apiserver audit log folder if not already exists
	if err := os.MkdirAll(kubeadmconstants.AuditVolumePath, 0700); err != nil {
		return errors.Wrapf(err, "Failed to create audit log directory %q", kubeadmconstants.AuditVolumePath)
	}
	fileName := filepath.Join(cfg.CertificatesDir, kubeadmconstants.AuditPolicyConfigFileName)
	return ioutil.WriteFile(fileName, []byte(AuditPolicy), os.FileMode(0600))
}
