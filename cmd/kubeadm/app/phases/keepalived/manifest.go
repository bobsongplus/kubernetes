/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package keepalived

const version = "2.0.20"

var keepalivedManifest = `
kind: Pod
apiVersion: v1
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  labels:
    component: keepalived
    tier: control-plane
  name: kube-keepalived
  namespace: kube-system
spec:
  hostNetwork: true
  containers:
  - name: kube-keepalived
    args:
    - --copy-service
    image: {{ .ImageRepository }}/keepalived-{{.Arch}}:{{.Version}}
    volumeMounts:
    - mountPath: /container/service/keepalived/assets/keepalived.conf
      name: config
    resources:
      requests:
        cpu: 100m
    securityContext:
      privileged: true
      capabilities:
        add:
        - NET_ADMIN
        add:
        - NET_BROADCAST
        add:
        - NET_RAW
  volumes:
  - hostPath:
      path: /etc/kubernetes/keepalived/keepalived.conf
      type: "File"
    name: config
`
