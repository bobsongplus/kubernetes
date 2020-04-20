/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package haproxy

var (
	Version         = "2.1.4"
	haproxyManifest = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  labels:
    component: haproxy
    tier: control-plane
  name: kube-haproxy
  namespace: kube-system
spec:
  containers:
  - image: {{ .ImageRepository }}/haproxy-{{.Arch}}:{{.Version}}
    command:
    - haproxy
    args: ["-db", "-f", "/usr/local/etc/haproxy/haproxy.cfg"]
    imagePullPolicy: IfNotPresent
    name: haproxy
    volumeMounts:
    - mountPath: /usr/local/etc/haproxy
      name: config
  enableServiceLinks: true
  hostNetwork: true
  priority: 0
  restartPolicy: Always
  volumes:
  - hostPath:
      path: /etc/kubernetes/haproxy
      type: ""
    name: config
`
)
