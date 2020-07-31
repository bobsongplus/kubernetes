/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package haproxy

var (
	Version         = "2.2.2"
	haproxyManifest = `
apiVersion: v1
kind: Pod
metadata:
  labels:
    k8s-app: haproxy
    tier: control-plane
  name: kube-haproxy
  namespace: kube-system
spec:
  containers:
  - image: {{ .ImageRepository }}/haproxy-{{.Arch}}:{{.Version}}
    imagePullPolicy: IfNotPresent
    name: haproxy
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /liveness
        port: 33305
      initialDelaySeconds: 60
      timeoutSeconds: 15
    volumeMounts:
    - mountPath: /usr/local/etc/haproxy
      name: config
  hostNetwork: true
  priorityClassName: system-cluster-critical
  restartPolicy: Always
  volumes:
  - hostPath:
      path: /etc/kubernetes/haproxy
      type: DirectoryOrCreate
    name: config
`
)
