package flannel

/**
 *  docker pull flannelcni/flannel:v0.17.0
 *  docker pull flannelcni/flannel-cni-plugin:v1.0.1
 *  https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/backends.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml
 */

const (
	Version = "v0.17.0"

	ConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: flannel-cfg
  namespace: kube-system
  labels:
    tier: node
    app: flannel
data:
  cni-conf.json: |
    {
      "name": "flannel",
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
    component: flannel
    k8s-app: flannel
spec:
  updateStrategy:
    type: OnDelete
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
      hostNetwork: true
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - operator: Exists
      serviceAccountName: flannel
      initContainers:
      - name: install-cni-plugin
        image: {{ .ImageRepository }}/flannel-cni-plugin:v1.0.1
        command:
        - cp
        args:
        - -f
        - /flannel
        - /opt/cni/bin/flannel
        volumeMounts:
        - name: cni-plugin
          mountPath: /opt/cni/bin
      - name: install-cni-cfg
        image: {{ .ImageRepository }}/flannel:{{ .Version }}
        command:
        - cp
        args:
        - -f
        - /etc/kube-flannel/cni-conf.json
        - /etc/cni/net.d/50-flannel.conflist
        volumeMounts:
        - name: cni
          mountPath: /etc/cni/net.d
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
      containers:
      - name: kube-flannel
        image: {{ .ImageRepository }}/flannel:{{ .Version }}
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
            add: ["NET_ADMIN", "NET_RAW"]
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
        - name: xtables-lock
          mountPath: /run/xtables.lock
        - name: etc-resolv-conf
          mountPath: /etc/resolv.conf
          readOnly: true
      volumes:
        - name: run
          hostPath:
            path: /run/flannel
        - name: cni-plugin
          hostPath:
            path: /opt/cni/bin
        - name: cni
          hostPath:
            path: /etc/cni/net.d
        - name: flannel-cfg
          configMap:
            name: flannel-cfg
        - name: xtables-lock
          hostPath:
            path: /run/xtables.lock
            type: FileOrCreate
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
apiVersion: rbac.authorization.k8s.io/v1
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
apiVersion: rbac.authorization.k8s.io/v1
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
