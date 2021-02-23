/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package flannel

/**
 *  quay.io/coreos/flannel:v0.11.0
 *  quay.io/coreos/flannel:v0.11.0-amd64
 *  https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/backends.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml
 */

const (
	Version = "v0.12.0"

	ConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: kube-flannel-cfg
  namespace: kube-system
  labels:
    tier: node
    app: flannel
data:
  cni-conf.json: |
    {
      "name": "k8s",
      "cniVersion":"0.3.1",
      "plugins": [
        {
          "type": "flannel",
          "delegate": {
            "hairpinMode": true,
            "isDefaultGateway": true
          }
        },{
          "type": "portmap",
          "capabilities": {
            "portMappings": true
          }
        },{
           "type": "tuning",
           "sysctl": {
               "net.core.somaxconn": "512"
            }
        },{
           "type": "bandwidth",
           "capabilities": {
              "bandwidth": true
            }
        }
      ]
    }
  net-conf.json: |
    {
      "Network": "{{ .PodSubnet }}",
      "Backend": {
        "Type": "{{ .Backend }}"
      }
    }`

	DaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: flannel
  namespace: kube-system
  labels:
    tier: node
    k8s-app: flannel
spec:
  selector:
    matchLabels:
      k8s-app: flannel
  template:
    metadata:
      labels:
        tier: node
        k8s-app: flannel
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: kubernetes.io/os
                    operator: In
                    values:
                      - linux
                  - key: kubernetes.io/arch
                    operator: In
                    values:
                      - {{.Arch}}
      hostNetwork: true
      nodeSelector:
        beta.kubernetes.io/arch: {{.Arch}}
        beta.kubernetes.io/os: linux
      tolerations:
      - operator: Exists
      serviceAccountName: flannel
      initContainers:
      - name: install-cni
        image: {{ .ImageRepository }}/flannel-{{.Arch}}:{{ .Version }}
        command:
        - cp
        args:
        - -f
        - /etc/kube-flannel/cni-conf.json
        - /etc/cni/net.d/10-flannel.conflist
        volumeMounts:
        - name: cni
          mountPath: /etc/cni/net.d
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
      containers:
      - name: kube-flannel
        image: {{ .ImageRepository }}/flannel-{{.Arch}}:{{ .Version }}
        command:
        - /opt/bin/flanneld
        args:
        - --ip-masq
        - --kube-subnet-mgr
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
          limits:
            cpu: "100m"
            memory: "50Mi"
        securityContext:
          privileged: false
          capabilities:
            add: ["NET_ADMIN"]
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: run
          mountPath: /run/flannel
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
        - name: etc-resolv-conf
          mountPath: /etc/resolv.conf
          readOnly: true
      volumes:
        - name: run
          hostPath:
            path: /run/flannel
        - name: cni
          hostPath:
            path: /etc/cni/net.d
        - name: flannel-cfg
          configMap:
            name: kube-flannel-cfg
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate
`

	// for flannel
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flannel
  namespace: kube-system
`

	ClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
  - apiGroups: ["policy"]
    resourceNames: ["system"]
    resources: ["podsecuritypolicies"]
    verbs: ["use"]
`

	ClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flannel
subjects:
- kind: ServiceAccount
  name: flannel
  namespace: kube-system
`
)
