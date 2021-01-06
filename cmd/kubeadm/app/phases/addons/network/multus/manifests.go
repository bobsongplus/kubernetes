package multus

const (
	MultusVersion = "v3.6"

	MultusControllerVersion = "v1.0"

	ClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: multus
rules:
  - apiGroups: ["k8s.cni.cncf.io"]
    resources:
      - '*'
    verbs:
      - '*'
  - apiGroups:
      - apiextensions.k8s.io
    resources:
      - customresourcedefinitions
    verbs:
      - create
      - update
      - get
  - apiGroups:
      - ""
    resources:
      - pods
      - configmaps
      - pods/status
    verbs:
      - get
      - update
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
  name: multus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: multus
subjects:
  - kind: ServiceAccount
    name: multus
    namespace: kube-system
`

	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: multus
  namespace: kube-system
`

	ConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: multus-cfg
  namespace: kube-system
  labels:
    app: multus
data:
  cni-conf.json: |
    {
      "cniVersion": "0.3.1",
      "name": "multus",
      "type": "multus",
      "logLevel": "error",
      "logFile": "/var/log/multus.log",
      "capabilities": {
        "portMappings": true
      },
      "clusterNetwork": "{{.MasterPlugin}}",
      "systemNamespaces": ["kube-system", "kube-public", "kube-node-lease"],
      "multusNamespace": "kube-system",
      "kubeconfig": "/etc/cni/net.d/multus.d/multus.kubeconfig"
    }
`

	DaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: multus
  namespace: kube-system
  labels:
    app: multus
    name: multus
spec:
  selector:
    matchLabels:
      app: multus
      name: multus
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        app: multus
        name: multus
    spec:
      hostNetwork: true
      nodeSelector:
        kubernetes.io/arch: amd64
      tolerations:
        - operator: Exists
      serviceAccountName: multus
      containers:
        - name: multus
          image: {{ .ImageRepository }}/multus-{{.Arch}}:{{ .Version }}
          command: ["/entrypoint.sh"]
          args:
            - "--multus-conf-file=/tmp/multus-conf/00-multus.conf"
            - "--multus-log-level=error"
            - "--multus-log-file=/var/log/multus.log"
            - "--cni-version=0.3.1"
          resources:
            requests:
              cpu: "100m"
              memory: "50Mi"
            limits:
              cpu: "100m"
              memory: "50Mi"
          securityContext:
            privileged: true
          volumeMounts:
            - name: cni
              mountPath: /host/etc/cni/net.d
            - name: cnibin
              mountPath: /host/opt/cni/bin
            - name: multus-cfg
              mountPath: /tmp/multus-conf
      volumes:
        - name: cni
          hostPath:
            path: /etc/cni/net.d
        - name: cnibin
          hostPath:
            path: /opt/cni/bin
        - name: multus-cfg
          configMap:
            name: multus-cfg
            items:
              - key: cni-conf.json
                path: 00-multus.conf
`

	Deployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: multus-controller
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: multus-controller
  template:
    metadata:
      labels:
        app: multus-controller
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      serviceAccountName: multus
      tolerations:
      - operator: Exists
      containers:
      - name: multus
        image: {{ .ImageRepository }}/multus-controller-{{.Arch}}:{{ .Version }}
        args:
          - "--plugins={{ .Plugins }}"
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
          limits:
            cpu: "100m"
            memory: "50Mi"
`

)
