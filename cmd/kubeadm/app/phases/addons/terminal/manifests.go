/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package terminal

const (
	//This DaemonSet is used to WebTerminal installation.
	DaemonSet = `
kind: DaemonSet
apiVersion: apps/v1
metadata:
  labels:
    k8s-app: kubectl
    use: webt
  name: kubectl
  namespace: kube-system
spec:
  updateStrategy:
    type: OnDelete
  selector:
    matchLabels:
      k8s-app: kubectl
      use: webt
  template:
    metadata:
      labels:
        k8s-app: kubectl
        use: webt
    spec:
      serviceAccountName: kubectl
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      tolerations:
      - operator: Exists
      containers:
        - name: kubectl
          image: {{ .ImageRepository }}/kubectl:{{ .Version }}
          resources:
            limits:
              cpu: 500m
              memory: 512Mi
            requests:
              cpu: 200m
              memory: 512Mi
          volumeMounts:
          - name: cri-socket
            mountPath: {{.RuntimePath}}
          - name: localtime
            mountPath: /etc/localtime
          - mountPath: /etc/resolv.conf
            name: resolv
          - mountPath: /etc/kubernetes/manifests/
            name: k8s
      volumes:
      - name: cri-socket
        hostPath:
          path: {{.RuntimePath}}
          type: Socket
      - name: localtime
        hostPath:
          path: /etc/localtime
          type: FileOrCreate
      - hostPath:
          path: /etc/resolv.conf
          type: FileOrCreate
        name: resolv
      - hostPath:
          path: /etc/kubernetes/manifests/
          type: DirectoryOrCreate
        name: k8s
`

	// for kubectl
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubectl
  namespace: kube-system`

	ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:kubectl
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: kubectl
  namespace: kube-system`
)
