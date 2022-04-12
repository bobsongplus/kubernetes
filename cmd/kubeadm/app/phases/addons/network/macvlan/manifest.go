package macvlan

const (
	WhereAboutsVersion             = "latest"
	WhereAboutsBootstrapterVersion = "v1.0"
	CNIPluginVersion               = "v1.1.1"

	DHCPDaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    k8s-app: dhcp-daemon
  name: dhcp-daemon
  namespace: kube-system
spec:
  updateStrategy:
    type: OnDelete
  selector:
    matchLabels:
      k8s-app: dhcp-daemon
  template:
    metadata:
      labels:
        k8s-app: dhcp-daemon
    spec:
      initContainers:
        - name: install-cni
          command: ["/install-cni.sh"]
          image: {{ .ImageRepository }}/dhcp-daemon:{{ .Version }}
          env:
            - name: CNI_CONF_NAME
              value: "50-macvlan.conflist"
          volumeMounts:
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
        - name: clean-sock
          image: {{ .ImageRepository }}/dhcp-daemon:{{ .Version }}
          command: ["/bin/sh"]
          args: ["-c", "rm -f /host/run/cni/dhcp.sock"]
          volumeMounts:
            - name: sock
              mountPath: /host/run/cni
      containers:
        - name: dhcp
          command:
            - /dhcp
            - daemon
            - -hostprefix=/host
            - -broadcast=true
            - -timeout=30s
          image: {{ .ImageRepository }}/dhcp-daemon:{{ .Version }}
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              cpu: "1"
              memory: 512Mi
            requests:
              cpu: 500m
              memory: 256Mi
          securityContext:
            privileged: true
          volumeMounts:
            - name: sock
              mountPath: /host/run/cni
            - name: proc
              mountPath: /host/proc
            - name: netns
              mountPath: /host/var/run/netns
              mountPropagation: HostToContainer
            - name: localtime
              mountPath: /etc/localtime
      hostNetwork: true
      restartPolicy: Always
      tolerations:
        - operator: Exists
      volumes:
        - name: sock
          hostPath:
            path: /run/cni
            type: DirectoryOrCreate
        - name: proc
          hostPath:
            path: /proc
            type: Directory
        - name: netns
          hostPath:
            path: /run/netns
            type: DirectoryOrCreate
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
            type: DirectoryOrCreate
        - name: localtime
          hostPath:
            path: /etc/localtime
            type: FileOrCreate
`

	WhereAboutsReconciler = `
apiVersion: batch/v1
kind: CronJob
metadata:
  name: whereabouts-reconciler
  namespace: kube-system
  labels:
    network: macvlan
    k8s-app: whereabouts-reconciler
spec:
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 0
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      backoffLimit: 0
      ttlSecondsAfterFinished: 300
      template:
        metadata:
          labels:
            network: macvlan
            k8s-app: whereabouts-reconciler
        spec:
          priorityClassName: "system-node-critical"
          serviceAccountName: whereabouts
          tolerations:
          - operator: Exists
          containers:
            - name: whereabouts
              image: {{ .ImageRepository }}/whereabouts:{{ .Version }}
              resources:
                requests:
                  cpu: "100m"
                  memory: "100Mi"
              command:
                - /ip-reconciler
                - -log-level=verbose
              volumeMounts:
                - name: cni-net-dir
                  mountPath: /host/etc/cni/net.d
          volumes:
            - name: cni-net-dir
              hostPath:
                path: /etc/cni/net.d
          restartPolicy: OnFailure

`
	WhereAboutsDaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: whereabouts
  namespace: kube-system
  labels:
    network: macvlan
    k8s-app: whereabouts
spec:
  selector:
    matchLabels:
      network: macvlan
      k8s-app: whereabouts
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        network: macvlan
        k8s-app: whereabouts
    spec:
      hostNetwork: true      
      serviceAccountName: whereabouts
      tolerations:
      - operator: Exists
      containers:
      - name: whereabouts
        command: [ "/bin/sh" ]
        args:
          - -c
          - >
            SLEEP=false /install-cni.sh &&
            /ip-control-loop -log-level debug
        image: {{ .ImageRepository }}/whereabouts:{{ .Version }}
        env:
        - name: WHEREABOUTS_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        resources:
          requests:
            cpu: "200m"
            memory: "256Mi"
          limits:
            cpu: "200m"
            memory: "256Mi"
        securityContext:
          privileged: true
        volumeMounts:
        - name: cnibin
          mountPath: /host/opt/cni/bin
        - name: cni-net-dir
          mountPath: /host/etc/cni/net.d
      volumes:
        - name: cnibin
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
`
	//WhereAboutsJob  init crd and ippool
	WhereAboutsJob = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    k8s-app: whereabouts-bootstraper
  name: whereabouts-bootstraper
  namespace: kube-system
spec:
  completions: 1
  parallelism: 1
  ttlSecondsAfterFinished: 3600
  template:
    metadata:
      labels:
        k8s-app: whereabouts-bootstraper
        network:  macvlan
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      serviceAccountName: whereabouts
      containers:
        - args:
            - --cidr={{ .PodSubnet }}
            - --kubeconfig=/etc/kubernetes/admin.conf
          image: {{ .ImageRepository }}/whereabouts-bootstraper:{{ .Version }}
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

	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: whereabouts
  namespace: kube-system
`
	ClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:whereabouts
rules:
- apiGroups:
  - whereabouts.cni.cncf.io
  resources:
  - ippools
  - overlappingrangeipreservations
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - create
  - get
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - '*'
- apiGroups: [""]
  resources:
  - pods
  verbs:
  - list
  - watch
- apiGroups: ["k8s.cni.cncf.io"]
  resources:
    - network-attachment-definitions
  verbs:
    - get
    - list
    - watch
- apiGroups:
  - ""
  - events.k8s.io
  resources:
    - events
  verbs:
  - create
  - patch
  - update
- apiGroups:
    - policy
  resources:
    - podsecuritypolicies
  verbs:
    - use
  resourceNames:
    - system
`
	ClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:whereabouts
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:whereabouts
subjects:
- kind: ServiceAccount
  name: whereabouts
  namespace: kube-system
`
)
