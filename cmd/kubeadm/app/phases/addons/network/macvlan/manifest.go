package macvlan

const (
	Version = "v0.9.1"

    DaemonSet = `
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
          image: {{ .ImageRepository }}/dhcp-daemon-{{.Arch}}:{{ .Version }}
          env:
            - name: CNI_CONF_NAME
              value: "50-macvlan.conflist"
            - name: NET_NAME
              value: "ens160"
          volumeMounts:
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
        - name: clean-sock
          image: {{ .ImageRepository }}/dhcp-daemon-{{.Arch}}:{{ .Version }}
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
          image: {{ .ImageRepository }}/dhcp-daemon-{{.Arch}}:{{ .Version }}
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

)

