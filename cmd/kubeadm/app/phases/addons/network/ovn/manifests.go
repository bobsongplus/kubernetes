/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 2021-01-04  @author weiwei@tenxcloud.com
 */
package ovn

/*
  1. ovn central ovn.io/role: central
  2. ovn certs
*/

const (
    Version = "v1.0"
    // ovn shard resource
    ConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: ovn-config
  namespace: kube-system
`

    ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovn
  namespace:  kube-system
`

    ClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.k8s.io/system-only: "true"
  name: system:ovn
rules:
  - apiGroups:
      - "ovn.io"
    resources:
      - subnets
      - subnets/status
      - ips
      - vlans
      - networks
    verbs:
      - "*"
  - apiGroups:
      - ""
    resources:
      - pods
      - namespaces
      - nodes
      - configmaps
    verbs:
      - create
      - get
      - list
      - watch
      - patch
      - update
  - apiGroups:
      - ""
      - networking.k8s.io
      - apps
      - extensions
    resources:
      - networkpolicies
      - services
      - endpoints
      - statefulsets
      - daemonsets
      - deployments
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - events
    verbs:
      - create
      - patch
      - update
  - apiGroups:
      - apiextensions.k8s.io
    resources:
      - customresourcedefinitions
    verbs:
      - create
      - update
      - get
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
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ovn
roleRef:
  name: system:ovn
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: ovn
    namespace:  kube-system
`

    // native ovn
    OvnNBService = `
kind: Service
apiVersion: v1
metadata:
  name: ovn-nb
  namespace:  kube-system
spec:
  ports:
    - name: ovn-nb
      protocol: TCP
      port: 6641
      targetPort: 6641
  type: ClusterIP
  selector:
    app: ovn-central
    ovn-nb-leader: "true"
  sessionAffinity: None
`

    OvnSBService = `
kind: Service
apiVersion: v1
metadata:
  name: ovn-sb
  namespace:  kube-system
spec:
  ports:
    - name: ovn-sb
      protocol: TCP
      port: 6642
      targetPort: 6642
  type: ClusterIP
  selector:
    app: ovn-central
    ovn-sb-leader: "true"
  sessionAffinity: None
`

    OvnExporterService = `
kind: Service
apiVersion: v1
metadata:
  name: ovn-exporter
  namespace:  kube-system
  labels:
    app: ovn-exporter
spec:
  ports:
    - name: metrics
      port: 10661
  type: ClusterIP
  selector:
    app: ovn-central
  sessionAffinity: None
`

    OvnCentralDeployment = `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: ovn-central
  namespace:  kube-system
  labels:
    component: ovn
  annotations:
    kubernetes.io/description: |
      OVN components: northd, nb and sb.
spec:
  replicas: 1
  strategy:
    rollingUpdate:
      maxSurge: 0%
      maxUnavailable: 100%
    type: RollingUpdate
  selector:
    matchLabels:
      app: ovn-central
  template:
    metadata:
      labels:
        app: ovn-central
        k8s-app: ovn
        component: ovn
    spec:
      tolerations:
      - operator: Exists
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: ovn-central
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      hostNetwork: true
      shareProcessNamespace: true
      containers:
        - name: ovn-central
          image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
          imagePullPolicy: IfNotPresent
          command: ["/ovn/start-central.sh"]
          securityContext:
            capabilities:
              add: ["SYS_NICE"]
          env:
            - name: SSL
              value: "false"
            - name: NODE_IPS
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          resources:
            limits:
              cpu: 1000m
              memory: 512Mi
            requests:
              cpu: 500m
              memory: 300Mi
          volumeMounts:
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /sys
              name: host-sys
              readOnly: true
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /etc/ovn
              name: host-config-ovn
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /var/run/pki
              name: ovn-certs
          readinessProbe:
            exec:
              command:
                - sh
                - /ovn/ovn-is-leader.sh
            periodSeconds: 3
            timeoutSeconds: 45
          livenessProbe:
            exec:
              command:
                - sh
                - /ovn/ovn-central-healthcheck.sh
            initialDelaySeconds: 30
            periodSeconds: 7
            failureThreshold: 5
            timeoutSeconds: 45
        - name: ovn-exporter
          image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
          imagePullPolicy: IfNotPresent
          command: ["/ovn/start-exporter.sh"]
          env:
          - name: SSL
            value: "false"
          resources:
            limits:
              cpu: 200m
              memory: 100Mi
            requests:
              cpu: 200m
              memory: 100Mi
          volumeMounts:
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /sys
              name: host-sys
              readOnly: true
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /etc/ovn
              name: host-config-ovn
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /var/run/pki
              name: ovn-certs
          readinessProbe:
            exec:
              command:
              - cat
              - /var/run/ovn/ovnnb_db.pid
            periodSeconds: 3
            timeoutSeconds: 45
          livenessProbe:
            exec:
              command:
              - cat
              - /var/run/ovn/ovn-nbctl.pid
            initialDelaySeconds: 30
            periodSeconds: 10
            failureThreshold: 5
            timeoutSeconds: 45
      nodeSelector:
        kubernetes.io/os: "linux"
        ovn.io/role: "central"
      volumes:
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-sys
          hostPath:
            path: /sys
        - name: host-config-openvswitch
          hostPath:
            path: /etc/origin/openvswitch
        - name: host-config-ovn
          hostPath:
            path: /etc/origin/ovn
        - name: host-log-ovs
          hostPath:
            path: /var/log/openvswitch
        - name: host-log-ovn
          hostPath:
            path: /var/log/ovn
        - name: ovn-certs
          secret:
            optional: true
            secretName: ovn-certs
`

    OvnHostDaemonSet = `
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovn-host
  namespace:  kube-system
  labels:
    component: ovn
  annotations:
    kubernetes.io/description: |
      This daemon set launches the openvswitch daemon.
spec:
  selector:
    matchLabels:
      app: ovn-host
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        app: ovn-host
        k8s-app: ovn
        component: ovn
    spec:
      tolerations:
      - operator: Exists
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      hostNetwork: true
      hostPID: true
      containers:
        - name: host
          image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
          imagePullPolicy: IfNotPresent
          command: ["/ovn/start-host.sh"]
          securityContext:
            runAsUser: 0
            privileged: true
          env:
            - name: SSL
              value: "false"
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: HW_OFFLOAD
              value: "false"
            - name: ENCAP
              value: "geneve"
          volumeMounts:
            - mountPath: /lib/modules
              name: host-modules
              readOnly: true
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /sys
              name: host-sys
              readOnly: true
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /etc/ovn
              name: host-config-ovn
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /var/run/pki
              name: ovn-certs
          readinessProbe:
            exec:
              command:
                - sh
                - /ovn/host-healthcheck.sh
            periodSeconds: 5
            timeoutSeconds: 45
          livenessProbe:
            exec:
              command:
                - sh
                - /ovn/host-healthcheck.sh
            initialDelaySeconds: 30
            periodSeconds: 5
            failureThreshold: 5
            timeoutSeconds: 45
          resources:
            requests:
              cpu: 200m
              memory: 300Mi
            limits:
              cpu: 1000m
              memory: 800Mi
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: host-modules
          hostPath:
            path: /lib/modules
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-sys
          hostPath:
            path: /sys
        - name: host-config-openvswitch
          hostPath:
            path: /etc/origin/openvswitch
        - name: host-config-ovn
          hostPath:
            path: /etc/origin/ovn
        - name: host-log-ovs
          hostPath:
            path: /var/log/openvswitch
        - name: host-log-ovn
          hostPath:
            path: /var/log/ovn
        - name: ovn-certs
          secret:
            optional: true
            secretName: ovn-certs
`

    // custom ovn
    OvnControllerService = `
kind: Service
apiVersion: v1
metadata:
  name: ovn-controller
  namespace:  kube-system
  labels:
    app: ovn-controller
spec:
  selector:
    app: ovn-controller
  ports:
    - port: 10660
      name: metrics
`

    OvnDaemonService = `
kind: Service
apiVersion: v1
metadata:
  name: ovn-daemon
  namespace:  kube-system
  labels:
    app: ovn-daemon
spec:
  selector:
    app: ovn-daemon
  ports:
    - port: 10665
      name: metrics
`

    OvnInspectorService = `
kind: Service
apiVersion: v1
metadata:
  name: ovn-inspector
  namespace:  kube-system
  labels:
    app: ovn-inspector
spec:
  selector:
    app: ovn-inspector
  ports:
    - port: 8080
      name: metrics
`

    OvnControllerDeployment = `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: ovn-controller
  namespace:  kube-system
  labels:
    component: ovn
  annotations:
    kubernetes.io/description: |
      ovn controller
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ovn-controller
  strategy:
    rollingUpdate:
      maxSurge: 0%
      maxUnavailable: 100%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ovn-controller
        k8s-app: ovn
        component: ovn
    spec:
      tolerations:
      - operator: Exists
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: ovn-controller
              topologyKey: kubernetes.io/hostname
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      hostNetwork: true
      containers:
        - name: ovn-controller
          image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
          imagePullPolicy: IfNotPresent
          command:
          - /ovn/start-controller.sh
          args:
          - --default-cidr={{ .PodSubnet }}
          - --default-exclude-ips=
          - --node-switch-cidr={{ .NodeSubnet }}
          - --network-type=geneve
          - --default-interface-name=
          - --default-vlan-id=100
          env:
            - name: SSL
              value: "false"
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          readinessProbe:
            exec:
              command:
                - sh
                - /ovn/ovn-controller-healthcheck.sh
            periodSeconds: 3
            timeoutSeconds: 45
          livenessProbe:
            exec:
              command:
                - sh
                - /ovn/ovn-controller-healthcheck.sh
            initialDelaySeconds: 300
            periodSeconds: 7
            failureThreshold: 5
            timeoutSeconds: 45
          volumeMounts:
          - mountPath: /var/run/pki
            name: ovn-certs
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: ovn-certs
          secret:
            optional: true
            secretName: ovn-certs
`

    OvnDaemonDaemonSet = `
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovn-daemon
  namespace:  kube-system
  labels:
    component: ovn
  annotations:
    kubernetes.io/description: |
      This daemon set launches the ovn cni daemon.
spec:
  selector:
    matchLabels:
      app: ovn-daemon
  updateStrategy:
    type: OnDelete
  template:
    metadata:
      labels:
        app: ovn-daemon
        k8s-app: ovn
        component: ovn
    spec:
      tolerations:
      - operator: Exists
      priorityClassName: system-cluster-critical
      serviceAccountName: ovn
      hostNetwork: true
      hostPID: true
      initContainers:
      - name: install-cni
        image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
        imagePullPolicy: IfNotPresent
        command: ["/ovn/install-cni.sh"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
          - mountPath: /opt/cni/bin
            name: cni-bin
      containers:
      - name: cniserver
        image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
        imagePullPolicy: IfNotPresent
        command:
          - sh
          - /ovn/start-daemon.sh
        args:
          - --enable-mirror=false
          - --encap-checksum=true
          - --service-cluster-ip-range={{ .ServiceSubnet }}
          - --iface=
          - --network-type=geneve
          - --default-interface-name=
        securityContext:
          runAsUser: 0
          privileged: true
        env:
          - name: SSL
            value: "false"
          - name: POD_IP
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
          - name: NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
        volumeMounts:
          - mountPath: /etc/cni/net.d
            name: cni-conf
          - mountPath: /run/openvswitch
            name: host-run-ovs
          - mountPath: /run/ovn
            name: host-run-ovn
          - mountPath: /var/run/netns
            name: host-ns
            mountPropagation: HostToContainer
        readinessProbe:
          exec:
            command:
              - nc
              - -z
              - -w3
              - 127.0.0.1
              - "10665"
          periodSeconds: 3
        livenessProbe:
          exec:
            command:
              - nc
              - -z
              - -w3
              - 127.0.0.1
              - "10665"
          initialDelaySeconds: 30
          periodSeconds: 7
          failureThreshold: 5
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: cni-conf
          hostPath:
            path: /etc/cni/net.d
        - name: cni-bin
          hostPath:
            path: /opt/cni/bin
        - name: host-ns
          hostPath:
            path: /var/run/netns
`

    OvnInspectorDaemonSet = `
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovn-inspector
  namespace:  kube-system
  labels:
    component: ovn
  annotations:
    kubernetes.io/description: |
      This daemon set launches the openvswitch daemon.
spec:
  selector:
    matchLabels:
      app: ovn-inspector
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ovn-inspector
        k8s-app: ovn
        component: ovn
    spec:
      tolerations:
        - operator: Exists
      serviceAccountName: ovn
      hostPID: true
      containers:
        - name: inspector
          image: {{ .ImageRepository }}/ovn-{{.Arch}}:{{ .Version }}
          command: ["/ovn/ovn-inspector", "--external-address=114.114.114.114", "--external-dns=tenxcloud.com"]
          imagePullPolicy: IfNotPresent
          securityContext:
            runAsUser: 0
            privileged: false
          env:
            - name: SSL
              value: "false"
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: HOST_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.hostIP
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - mountPath: /lib/modules
              name: host-modules
              readOnly: true
            - mountPath: /run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/openvswitch
              name: host-run-ovs
            - mountPath: /var/run/ovn
              name: host-run-ovn
            - mountPath: /sys
              name: host-sys
              readOnly: true
            - mountPath: /etc/openvswitch
              name: host-config-openvswitch
            - mountPath: /var/log/openvswitch
              name: host-log-ovs
            - mountPath: /var/log/ovn
              name: host-log-ovn
            - mountPath: /var/run/pki
              name: ovn-certs
          resources:
            requests:
              cpu: 100m
              memory: 300Mi
            limits:
              cpu: 200m
              memory: 400Mi
      nodeSelector:
        kubernetes.io/os: "linux"
      volumes:
        - name: host-modules
          hostPath:
            path: /lib/modules
        - name: host-run-ovs
          hostPath:
            path: /run/openvswitch
        - name: host-run-ovn
          hostPath:
            path: /run/ovn
        - name: host-sys
          hostPath:
            path: /sys
        - name: host-config-openvswitch
          hostPath:
            path: /etc/origin/openvswitch
        - name: host-log-ovs
          hostPath:
            path: /var/log/openvswitch
        - name: host-log-ovn
          hostPath:
            path: /var/log/ovn
        - name: ovn-certs
          secret:
            optional: true
            secretName: ovn-certs
`



)