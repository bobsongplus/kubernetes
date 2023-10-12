/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package calico

const (
	Version = "v3.24.6"

	//This ConfigMap is used to configure a self-hosted Calico installation.
	NodeConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-config
  namespace: kube-system
data:
  etcd_endpoints: {{ .EtcdEndPoints }}
  etcd_ca: "/calico-secrets/etcd-ca"
  etcd_cert: "/calico-secrets/etcd-cert"
  etcd_key: "/calico-secrets/etcd-key"
  calico_backend: "bird"
  chain_insert_mode: "Append"
  veth_mtu: "0"
  ip: {{ .IPAutoDetection }}
  ip_autodetection_method: "kubernetes-internal-ip"
  ip6: {{ .IP6AutoDetection }}
  ip6_autodetection_method: "kubernetes-internal-ip"
  cni_network_config: |-
    {
        "name": "calico",
        "cniVersion": "0.3.1",
        "plugins": [
          {
            "type": "calico",
            "etcd_endpoints": "__ETCD_ENDPOINTS__",
            "etcd_key_file": "__ETCD_KEY_FILE__",
            "etcd_cert_file": "__ETCD_CERT_FILE__",
            "etcd_ca_cert_file": "__ETCD_CA_CERT_FILE__",
            "log_level": "__LOG_LEVEL__",
            "log_file_path": "__LOG_FILE_PATH__",
            "mtu": __CNI_MTU__,
            "ipam": {
                "type": "calico-ipam",
                "assign_ipv4": "{{ .AssignIpv4 }}",
                "assign_ipv6": "{{ .AssignIpv6 }}"
             },
            "policy": {
                  "type": "k8s"
             },
            "kubernetes": {
                "kubeconfig": "/etc/cni/net.d/calico-kubeconfig"
             }
          },{
             "type": "portmap",
             "snat": true,
             "capabilities": {
                "portMappings": true
              }
          },{
             "type": "tuning",
             "sysctl": {
                 "net.core.somaxconn": "1024"
              }
          },{
             "type": "bandwidth",
             "capabilities": {
               "bandwidth": true
              }
          }
        ]
    }`

	// This manifest installs the calico/node container,
	// as well as the Calico CNI plugins and network config on
	// each master and worker node in a Kubernetes cluster.
	Node = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    network: calico
    k8s-app: calico
    component: calico
spec:
  selector:
    matchLabels:
      network: calico
      k8s-app: calico
      component: calico
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        network: calico
        k8s-app: calico
        component: calico
    spec:
      nodeSelector:
        kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
      - operator: Exists
      serviceAccountName: calico-node
      terminationGracePeriodSeconds: 0
      priorityClassName: system-node-critical
      initContainers:
        - name: install-cni
          image: {{ .ImageRepository }}/cni:{{ .Version }}
          imagePullPolicy: IfNotPresent
          command: ["/opt/cni/bin/install"]
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
          envFrom:
            - configMapRef:
                name: kubernetes-services-endpoint
                optional: true
          env:
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_endpoints
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: cni_network_config
            - name: CNI_CONF_NAME
              value: "10-calico.conflist"
            - name: CNI_MTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            - name: LOG_LEVEL
              value: "error"
            - name: SLEEP
              value: "false"
            - name: UPDATE_CNI_BINARIES
              value: "false"
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
            - mountPath: /calico-secrets
              name: etcd-certs
          securityContext:
            privileged: true
      containers:
        - name: calico-node
          image: {{ .ImageRepository }}/node:{{ .Version }}
          envFrom:
            - configMapRef:
                name: kubernetes-services-endpoint
                optional: true
          env:
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_endpoints
            - name: ETCD_CA_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_ca
            - name: ETCD_KEY_FILE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_key
            - name: ETCD_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_cert
            - name: CALICO_NETWORKING_BACKEND
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: calico_backend
            - name: IP
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip
            - name: IP6
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip6
            - name: IP_AUTODETECTION_METHOD
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip_autodetection_method
            - name: IP6_AUTODETECTION_METHOD
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip6_autodetection_method
            - name: CLUSTER_TYPE
              value: "k8s,bgp"
            - name: CALICO_K8S_NODE_REF
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            - name: NO_DEFAULT_POOLS
              value: "true"
            - name: FELIX_IPV6SUPPORT
              value: "false"
            - name: FELIX_IPINIPMTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            - name: FELIX_VXLANMTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            - name: FELIX_WIREGUARDMTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            - name: FELIX_CHAININSERTMODE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: chain_insert_mode
            - name: CALICO_STARTUP_LOGLEVEL
              value: WARNING
            - name: BGP_LOGSEVERITYSCREEN
              value: warning
            - name: FELIX_LOGSEVERITYSCREEN
              value: WARNING
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            limits:
              cpu: 300m
              memory: 256Mi
            requests:
              cpu: 300m
              memory: 256Mi
          lifecycle:
            preStop:
              exec:
                command:
                - /bin/calico-node
                - -shutdown
          livenessProbe:
            exec:
              command:
                - /bin/calico-node
                - -felix-live
                - -bird-live
            periodSeconds: 30
            initialDelaySeconds: 60
            failureThreshold: 6
            timeoutSeconds: 20
          readinessProbe:
            exec:
              command:
                - /bin/calico-node
                - -felix-ready
                - -bird-ready
            periodSeconds: 30
            timeoutSeconds: 20
          volumeMounts:
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
              readOnly: false
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /run/xtables.lock
              name: xtables-lock
              readOnly: false
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /var/lib/calico
              name: var-lib-calico
              readOnly: false
            - mountPath: /calico-secrets
              name: etcd-certs
            - mountPath: /sys/fs/
              name: sysfs
              mountPropagation: Bidirectional
            - name: cni-log-dir
              mountPath: /var/log/calico/cni
              readOnly: true
      volumes:
        - name: lib-modules
          hostPath:
            path: /lib/modules
            type: DirectoryOrCreate
        - name: xtables-lock
          hostPath:
            path: /run/xtables.lock
            type: FileOrCreate
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
            type: DirectoryOrCreate
        - name: var-lib-calico
          hostPath:
            path: /var/lib/calico
            type: DirectoryOrCreate
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
            type: DirectoryOrCreate
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
            type: DirectoryOrCreate
        - name: sysfs
          hostPath:
            path: /sys/fs/
            type: DirectoryOrCreate
        - name: cni-log-dir
          hostPath:
            path: /var/log/calico/cni
            type: DirectoryOrCreate
        - name: etcd-certs
          secret:
            secretName: etcd-certs
            defaultMode: 0400
`

	// This manifest installs the calico/kube-controllers container on each master.
	// using kube-controllers only if you're using the etcd Datastore
	// See https://github.com/projectcalico/kube-controllers
	//     https://docs.projectcalico.org/archive/v3.18/reference/kube-controllers/configuration
	//     https://github.com/kubernetes/contrib/tree/master/election

	//The calico/kube-controllers container includes the following controllers:
	//1> policy controller: watches network policies and programs Calico policies.
	//2> profile controller: watches namespaces and programs Calico profiles.
	//3> workloadendpoint controller: watches for changes to pod labels and updates Calico workload endpoints.
	//4> node controller: watches for the removal of Kubernetes nodes and removes corresponding data from Calico.
	//5> serviceAccount controller: implements the Controller interface for managing Kubernetes service account
	//   and syncing them to the Calico datastore as Profiles.
	KubeController = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: calico-kube-controller
  namespace: kube-system
  labels:
    network: calico
    k8s-app: calico-kube-controller
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      network: calico
      k8s-app: calico-kube-controller
  template:
    metadata:
      labels:
        network: calico
        k8s-app: calico-kube-controller
    spec:
      hostNetwork: true
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
        - operator: Exists
      serviceAccountName: calico-kube-controller
      containers:
      - name: kube-controller
        image: {{ .ImageRepository }}/kube-controllers:{{ .Version }}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 200m
            memory: 512Mi
          requests:
            cpu: 200m
            memory: 512Mi
        env:
          - name: ETCD_ENDPOINTS
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_endpoints
          - name: ETCD_CA_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_ca
          - name: ETCD_KEY_FILE
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_key
          - name: ETCD_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_cert
          - name: AUTO_HOST_ENDPOINTS
            value: enabled
          - name: LOG_LEVEL
            value: error
        livenessProbe:
          exec:
            command:
            - /usr/bin/check-status
            - -l
          periodSeconds: 30
          initialDelaySeconds: 60
          failureThreshold: 6
          timeoutSeconds: 20
        readinessProbe:
          exec:
            command:
            - /usr/bin/check-status
            - -r
          periodSeconds: 30
          timeoutSeconds: 20
        volumeMounts:
          - mountPath: /calico-secrets
            name: etcd-certs
            readOnly: true
      volumes:
        - name: etcd-certs
          secret:
            secretName: etcd-certs
            defaultMode: 0440
`

	CtlConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}
  namespace: kube-system
  labels:
    networking.projectcalico.org/name: {{ .Name }}
  annotations:
    networking.kubernetes.io/pod-cidr: {{ .PodSubnet }}
    networking.kubernetes.io/service-cidr: {{ .ServiceSubnet }}
    networking.projectcalico.org/cidr: {{ .PodSubnet }}
    networking.projectcalico.org/name: {{ .Name }}
data:
  ippool.yaml: |-
    apiVersion: projectcalico.org/v3
    kind: IPPool
    metadata:
      name: {{ .Name }}
    spec:
      cidr: {{ .PodSubnet }}
      ipipMode: Never
      vxlanMode: Always
      natOutgoing: true
`

	CtlJob = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    k8s-app: {{ .Name }}
  name: configure-{{ .Name }}
  namespace: kube-system
spec:
  completions: 1
  parallelism: 1
  ttlSecondsAfterFinished: 3600
  template:
    metadata:
      labels:
        k8s-app: {{ .Name }}
    spec:
      containers:
      - args:
        - apply
        - -f
        - /etc/config/calico/ippool.yaml
        env:
        - name: ETCD_ENDPOINTS
          value: https://127.0.0.1:2379
        - name: ETCD_CA_CERT_FILE
          value: /calico-secrets/etcd-ca
        - name: ETCD_CERT_FILE
          value: /calico-secrets/etcd-cert
        - name: ETCD_KEY_FILE
          value: /calico-secrets/etcd-key
        image: {{ .ImageRepository }}/ctl:{{ .Version }}
        imagePullPolicy: IfNotPresent
        name: configure
        volumeMounts:
        - mountPath: /etc/config
          name: config-volume
        - mountPath: /calico-secrets
          name: etcd-certs
          readOnly: true
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
      tolerations:
        - operator: Exists
      restartPolicy: OnFailure
      volumes:
      - configMap:
          defaultMode: 0420
          items:
          - key: ippool.yaml
            path: calico/ippool.yaml
          name: {{ .Name }}
        name: config-volume
      - name: etcd-certs
        secret:
          secretName: etcd-certs
          defaultMode: 0400
`
	// for calico/node
	CalicoClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:calico-node
rules:
  - apiGroups: [""]
    resources:
      - pods
      - nodes
      - namespaces
    verbs:
      - get
  - apiGroups: ["discovery.k8s.io"]
    resources:
      - endpointslices
    verbs:
      - watch 
      - list
  - apiGroups: [""]
    resources:
      - endpoints
      - services
    verbs:
      - watch
      - list
  - apiGroups: [""]
    resources:
      - configmaps
    verbs:
      - get
  - apiGroups: [""]
    resources:
      - nodes/status
    verbs:
      - patch
  - apiGroups: [""]
    resources:
      - serviceaccounts/token
    resourceNames:
      - calico-node
    verbs:
      - create
`

	CalicoServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-node
  namespace: kube-system`

	CalicoClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:calico-node
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico-node
subjects:
- kind: ServiceAccount
  name: calico-node
  namespace: kube-system`

	// for calico/kube-controllers
	CalicoControllersClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:calico-kube-controller
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - namespaces
  - nodes
  - serviceaccounts
  verbs:
  - watch
  - list
  - get
- apiGroups:
  - networking.k8s.io
  resources:
  - networkpolicies
  verbs:
  - watch
  - list
`

	CalicoControllersServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-kube-controller
  namespace: kube-system
`

	CalicoControllersClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:calico-kube-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico-kube-controller
subjects:
- kind: ServiceAccount
  name: calico-kube-controller
  namespace: kube-system
`
)
