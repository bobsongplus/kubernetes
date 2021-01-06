/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package weavenet

const (
	Version = "2.6.2"

	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: weave-net
  labels:
    name: weave-net
  namespace: kube-system
`
	ClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: weave-net
  labels:
    name: weave-net
rules:
  - apiGroups:
      - ''
    resources:
      - pods
      - namespaces
      - nodes
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - networking.k8s.io
    resources:
      - networkpolicies
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ''
    resources:
    - nodes/status
    verbs:
    - patch
    - update
`
	ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: weave-net
  labels:
    name: weave-net
roleRef:
  kind: ClusterRole
  name: weave-net
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: weave-net
    namespace: kube-system
`
	Role = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: weave-net
  labels:
    name: weave-net
  namespace: kube-system
rules:
  - apiGroups:
      - ''
    resourceNames:
      - weave-net
    resources:
      - configmaps
    verbs:
      - get
      - update
  - apiGroups:
      - ''
    resources:
      - configmaps
    verbs:
      - create
`
	RoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: weave-net
  labels:
    name: weave-net
  namespace: kube-system
roleRef:
  kind: Role
  name: weave-net
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: weave-net
    namespace: kube-system
`
	DaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: weave-net
  labels:
    name: weave-net
  namespace: kube-system
spec:
  minReadySeconds: 5
  selector:
    matchLabels:
      name: weave-net
      component: weave-net
      k8s-app: weave-net
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        name: weave-net
        component: weave-net
        k8s-app: weave-net
    spec:
      containers:
        - name: weave
          command:
            - /home/weave/launch.sh
          env:
            - name: HOSTNAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
            - name: NO_MASQ_LOCAL
              value: "1"
            - name: IPALLOC_RANGE
              value: {{ .PodSubnet }}
          image: {{ .ImageRepository }}/weave-kube-{{ .Arch }}:{{ .Version }}
          readinessProbe:
            httpGet:
              host: 127.0.0.1
              path: /status
              port: 6784
          resources:
            requests:
              cpu: 10m
          securityContext:
            privileged: true
          volumeMounts:
          - name: weavedb
            mountPath: /weavedb
          - name: cni-bin
            mountPath: /host/opt
          - name: cni-bin2
            mountPath: /host/home
          - name: cni-conf
            mountPath: /host/etc
          - name: dbus
            mountPath: /host/var/lib/dbus
          - name: lib-modules
            mountPath: /lib/modules
          - name: xtables-lock
            mountPath: /run/xtables.lock
        - name: weave-npc
          env:
            - name: HOSTNAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
          image: {{ .ImageRepository }}/weave-npc-{{ .Arch }}:{{ .Version }}
          resources:
            requests:
              cpu: 10m
          securityContext:
            privileged: true
          volumeMounts:
            - name: xtables-lock
              mountPath: /run/xtables.lock
      dnsPolicy: ClusterFirstWithHostNet
      hostNetwork: true
      hostPID: true
      priorityClassName: system-node-critical
      restartPolicy: Always
      securityContext:
        seLinuxOptions: {}
      serviceAccountName: weave-net
      tolerations:
      - operator: Exists
      volumes:
      - name: weavedb
        hostPath:
          path: /var/lib/weave
      - name: cni-bin
        hostPath:
          path: /opt
      - name: cni-bin2
        hostPath:
          path: /home
      - name: cni-conf
        hostPath:
          path: /etc
      - name: dbus
        hostPath:
          path: /var/lib/dbus
      - name: lib-modules
        hostPath:
          path: /lib/modules
      - name: xtables-lock
        hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
`
)
