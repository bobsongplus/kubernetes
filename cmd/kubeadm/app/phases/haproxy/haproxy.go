/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package haproxy

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"runtime"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	DefaultHaproxyDir    = "haproxy"
	DefaultHaproxyConfig = "haproxy.cfg"
	HaproxyManifestName  = "haproxy.yaml"
)

var HaproxyTemplate = `global
  stats timeout 30s
  daemon
  ssl-default-bind-ciphers ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:RSA+AESGCM:RSA+AES:!aNULL:!MD5:!DSS
  ssl-default-bind-options no-sslv3

defaults
  log global
  mode  http
  option  httplog
  option  dontlognull
  timeout connect 5000
  timeout client  3600000
  timeout server  3600000
  timeout http-request 15s
  timeout http-keep-alive 15s

frontend liveness
  bind 127.0.0.1:33305
  mode http
  option httplog
  monitor-uri /liveness

listen stats
  bind    *:9000
  mode    http
  stats   enable
  stats   hide-version
  stats   uri       /stats
  stats   refresh   30s
  stats   realm     Haproxy\ Statistics
  stats   auth      admin:haproxy-ha

frontend k8s-api
  bind *:6443
  mode tcp
  option tcplog
  tcp-request inspect-delay 5s
  default_backend k8s-api

frontend etcd
  bind *:12379
  mode tcp
  option tcplog
  tcp-request inspect-delay 5s
  default_backend etcd

backend etcd
  mode tcp
  option tcplog
  option ssl-hello-chk
  balance roundrobin
  default-server inter 10s downinter 5s rise 2 fall 2 slowstart 60s maxconn 250 maxqueue 256 weight 100
  {{range .}}server {{ .Name }} {{ .IP }}:2379 check
  {{end}}

backend k8s-api
  mode tcp
  option tcplog
  option ssl-hello-chk
  balance roundrobin
  default-server inter 10s downinter 5s rise 2 fall 2 slowstart 60s maxconn 250 maxqueue 256 weight 100
  {{range .}}server {{ .Name }} {{ .IP }}:{{ .Port}} check
  {{end}}`

type Master struct {
	Name string
	IP   string
	Port int32
}

// lbs is loadbalance ip list, haproxy_cfg is haproxy config path
func CreateHaproxyConfig(port int32, lbs []string, haproxy_cfg string) error {
	masters := make([]Master, 0)
	for k, lb := range lbs {
		masters = append(masters, Master{
			Name: fmt.Sprintf("master-%d", k),
			IP:   lb,
			Port: port,
		})
	}
	t, err := template.New("haproxy").Parse(HaproxyTemplate)
	if err != nil {
		klog.Error(err)
		return err
	}
	bb := new(bytes.Buffer)
	if err := t.Execute(bb, masters); err != nil {
		klog.Error(err)
		return err
	}
	cfgBytes, err := ioutil.ReadAll(bb)
	if err != nil {
		klog.Error(err)
		return err
	}
	if err := ioutil.WriteFile(haproxy_cfg, cfgBytes, 0644); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

func CreateLocalHaproxyStaticPodManifestFile(cfg *kubeadmapi.InitConfiguration, haproxyManifestFile string) error {
	haproxyPodBytes, err := kubeadmutil.ParseTemplate(haproxyManifest, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error parsing haproxy  template error: %v", err)
	}
	if err := ioutil.WriteFile(haproxyManifestFile, haproxyPodBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write haproxy static pod to the file %q", haproxyManifestFile)
	}
	return nil
}
