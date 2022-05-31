/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */

package calico

const (
	Version = "v3.23.2"

	//This ConfigMap is used to configure a self-hosted Calico installation.
	NodeConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-config
  namespace: kube-system
data:
  typha_service_name: "calico-typha"
  calico_backend: "bird"
  veth_mtu: "0"
  ip: {{ .IPAutoDetection }}
  ip_autodetection_method: "first-found"
  ip6: {{ .IP6AutoDetection }}
  ip6_autodetection_method: "first-found"
  cni_network_config: |-
    {
        "name": "calico",
        "cniVersion": "0.3.1",
        "plugins": [
          {
            "type": "calico",
            "datastore_type": "kubernetes",
            "nodename": "__KUBERNETES_NODE_NAME__",
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
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
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
            - name: DATASTORE_TYPE
              value: "kubernetes"
            - name: FELIX_TYPHAK8SSERVICENAME
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: typha_service_name
            - name: WAIT_FOR_DATASTORE
              value: "true"
            - name: NODENAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
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
              value: "true"
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
`

	// This manifest installs the calico/kube-controllers container on each master.
	// using kube-controllers only if you're using the etcd Datastore
	// See https://github.com/projectcalico/kube-controllers
	//     https://docs.projectcalico.org/archive/v3.22/reference/kube-controllers/configuration

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
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
        - operator: Exists
      serviceAccountName: calico-kube-controller
      priorityClassName: system-cluster-critical
      containers:
      - name: kube-controller
        image: {{ .ImageRepository }}/kube-controllers:{{ .Version }}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 200m
            memory: 256Mi
        env:
          - name: ENABLED_CONTROLLERS
            value: node
          - name: DATASTORE_TYPE
            value: kubernetes
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
          initialDelaySeconds: 60
          failureThreshold: 6
          periodSeconds: 30
          timeoutSeconds: 20
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
      - get
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
      - update
  - apiGroups: ["networking.k8s.io"]
    resources:
      - networkpolicies
    verbs:
      - watch
      - list
  - apiGroups: [""]
    resources:
      - pods
      - namespaces
      - serviceaccounts
    verbs:
      - list
      - watch
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - patch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - globalfelixconfigs
      - felixconfigurations
      - bgppeers
      - globalbgpconfigs
      - bgpconfigurations
      - ippools
      - ipreservations
      - ipamblocks
      - globalnetworkpolicies
      - globalnetworksets
      - networkpolicies
      - networksets
      - clusterinformations
      - hostendpoints
      - blockaffinities
      - caliconodestatuses
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ippools
      - felixconfigurations
      - clusterinformations
    verbs:
      - create
      - update
  - apiGroups: [ "crd.projectcalico.org" ]
    resources:
      - caliconodestatuses
    verbs:
      - update
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - bgpconfigurations
      - bgppeers
    verbs:
      - create
      - update
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
      - ipamblocks
      - ipamhandles
    verbs:
      - get
      - list
      - create
      - update
      - delete
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ipamconfigs
    verbs:
      - get
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
    verbs:
      - watch
  - apiGroups: ["apps"]
    resources:
      - daemonsets
    verbs:
      - get
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
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - watch
      - list
      - get
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ipreservations
    verbs:
      - list
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
      - ipamblocks
      - ipamhandles
    verbs:
      - get
      - list
      - create
      - update
      - delete
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ippools
    verbs:
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - hostendpoints
    verbs:
      - get
      - list
      - create
      - update
      - delete
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - clusterinformations
    verbs:
      - get
      - list
      - create
      - update
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - kubecontrollersconfigurations
    verbs:
      - get
      - create
      - update
      - watch
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

	// for calico-typha
	Typha = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: calico-typha
  namespace: kube-system
  labels:
    network: calico
    k8s-app: calico-typha
spec:
  replicas: 2
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      network: calico
      k8s-app: calico-typha
  template:
    metadata:
      labels:
        network: calico
        k8s-app: calico-typha
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: 'true'
    spec:
      nodeSelector:
        kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
      - operator: Exists
      serviceAccountName: calico-node
      priorityClassName: system-cluster-critical
      securityContext:
        fsGroup: 65534
      containers:
      - image: {{ .ImageRepository }}/typha:{{ .Version }}
        name: typha
        ports:
          - containerPort: 5473
            name: calico-typha
            protocol: TCP
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 200m
            memory: 256Mi
        envFrom:
          - configMapRef:
              name: kubernetes-services-endpoint
              optional: true
        env:
          - name: TYPHA_LOGSEVERITYSCREEN
            value: "error"
          - name: TYPHA_LOGFILEPATH
            value: "none"
          - name: TYPHA_LOGSEVERITYSYS
            value: "none"
          - name: TYPHA_CONNECTIONREBALANCINGMODE
            value: "kubernetes"
          - name: TYPHA_DATASTORETYPE
            value: "kubernetes"
          - name: TYPHA_HEALTHENABLED
            value: "true"
        livenessProbe:
          httpGet:
            path: /liveness
            port: 9098
            host: localhost
          periodSeconds: 30
          initialDelaySeconds: 30
          timeoutSeconds: 10
        securityContext:
          runAsNonRoot: true
          allowPrivilegeEscalation: false
        readinessProbe:
          httpGet:
            path: /readiness
            port: 9098
            host: localhost
          periodSeconds: 10
          timeoutSeconds: 10
`
	// for calico-apiserver
	APIServerDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: calico-apiserver
  name: calico-apiserver
  namespace: kube-system
spec:
  replicas: 2
  selector:
    matchLabels:
      network: calico
      k8s-app: calico-apiserver
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        network: calico
        k8s-app: calico-apiserver
    spec:
      containers:
      - name: calico-apiserver
        args:
        - --secure-port=5443
        - -v=1
        env:
        - name: DATASTORE_TYPE
          value: kubernetes
        image: {{ .ImageRepository }}/apiserver:{{ .Version }}
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 200m
            memory: 256Mi
        securityContext:
          privileged: false
          runAsUser: 0
        volumeMounts:
        - mountPath: /code/apiserver.local.config/certificates
          name: calico-apiserver
      dnsPolicy: ClusterFirst
      nodeSelector:
        kubernetes.io/os: linux
      restartPolicy: Always
      serviceAccountName: calico-apiserver
      tolerations:
      - operator: Exists
      volumes:
      - name: calico-apiserver
        secret:
          secretName: calico-apiserver
`

	APIServerServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-apiserver 
  namespace: kube-system
`

	APIServerClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:calico-apiserver
rules:
- apiGroups:
  - extensions
  - networking.k8s.io
  - ""
  resources:
  - networkpolicies
  - nodes
  - namespaces
  - pods
  - serviceaccounts
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - crd.projectcalico.org
  resources:
  - globalnetworkpolicies
  - networkpolicies
  - clusterinformations
  - hostendpoints
  - globalnetworksets
  - networksets
  - bgpconfigurations
  - bgppeers
  - felixconfigurations
  - kubecontrollersconfigurations
  - ippools
  - ipreservations
  - ipamblocks
  - blockaffinities
  - caliconodestatuses
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - delete
- apiGroups:
    - ""
  resourceNames:
    - extension-apiserver-authentication
  resources:
    - configmaps
  verbs:
    - list
    - watch
    - get
- apiGroups:
    - rbac.authorization.k8s.io
  resources:
    - clusterroles
    - clusterrolebindings
    - roles
    - rolebindings
  verbs:
    - get
    - list
    - watch
- apiGroups:
    - admissionregistration.k8s.io
  resources:
    - mutatingwebhookconfigurations
    - validatingwebhookconfigurations
  verbs:
    - get
    - list
    - watch
- apiGroups:
  - policy
  resourceNames:
  - system
  resources:
  - podsecuritypolicies
  verbs:
  - use
`

	APIServerClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:calico-apiserver
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico-apiserver
subjects:
- kind: ServiceAccount
  name: calico-apiserver
  namespace: kube-system
`

	APIServerClusterRoleBindingDelegator = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:calico-apiserver-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: calico-apiserver
  namespace: kube-system
`

	// for calico-bootstraper
	BootstraperServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-bootstraper
  namespace: kube-system
`

	BootstraperClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:calico-bootstraper
rules:
  - apiGroups:
      - apiextensions.k8s.io
    resources:
      - customresourcedefinitions
    verbs:
      - create
      - update
      - get
  - apiGroups:
      - operator.tigera.io
    resources:
      - installations
      - apiservers
    verbs:
      - create
  - apiGroups:
      - apiregistration.k8s.io
    resources:
      - apiservices
    verbs:
      - create
  - apiGroups:
      - crd.projectcalico.org
    resources:
      - ippools
    verbs:
      - create
  - apiGroups:
      - policy
    resourceNames:
      - system
    resources:
      - podsecuritypolicies
    verbs:
      - use
`

	BootstraperClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: calico-bootstraper
subjects:
  - kind: ServiceAccount
    name: calico-bootstraper
    namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico-bootstraper
`

	BootstraperJob = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    k8s-app: calico-bootstraper
  name: calico-bootstraper
  namespace: kube-system
spec:
  completions: 1
  parallelism: 1
  ttlSecondsAfterFinished: 3600
  template:
    metadata:
      labels:
        k8s-app: calico-bootstraper
        network:  calico
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      serviceAccountName: calico-bootstraper
      containers:
        - args:
            - --cidr={{ .PodSubnet }}
            - --kubeconfig=/etc/kubernetes/admin.conf
            - --imageRepository={{ .ImageRepository }}
          image: {{ .ImageRepository }}/calico-bootstraper:{{ .Version }}
          imagePullPolicy: IfNotPresent
          name: bootstraper
          volumeMounts:
            - name: kubeconfig
              mountPath: /etc/kubernetes
              readOnly: true
      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
      tolerations:
        - operator: Exists
      restartPolicy: OnFailure
      volumes:
        - name: kubeconfig
          hostPath:
            path: /etc/kubernetes
            type: DirectoryOrCreate
`
)
