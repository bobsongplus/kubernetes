package haproxy

const (
	Version = "2.2.11"

	DefaultHaproxyCfg = "haproxy.cfg.tmpl"
	DefaultHaproxy  = "haproxy.yaml"

	HAProxyCfg = `
global
  daemon
  stats timeout 30s
  log 127.0.0.1 local2
  ssl-default-bind-ciphers ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:RSA+AESGCM:RSA+AES:!aNULL:!MD5:!DSS
  ssl-default-bind-options no-sslv3

defaults
  log global
  mode  http
  option  httplog
  option  dontlognull
  timeout connect 5000
  timeout client  3600000
  timeout server  3600000
  timeout http-request 15s
  timeout http-keep-alive 15s

frontend liveness
  bind 127.0.0.1:33305
  mode http
  option httplog
  monitor-uri /liveness


listen stats
  bind    *:9000
  mode    http
  stats   enable
  stats   hide-version
  stats   uri       /stats
  stats   refresh   30s
  stats   realm     Haproxy\ Statistics
  stats   auth      admin:Dream001

frontend api
  bind *:[[.Port]]
  mode tcp
  option tcplog
  tcp-request inspect-delay 5s
  default_backend api

backend api
  mode tcp
  option ssl-hello-chk
  balance leastconn
  default-server inter 10s downinter 5s rise 2 fall 2 slowstart 60s maxconn 250 maxqueue 256 weight 100
{{range $i,$s := ls "/masterleases"}}
  server m-{{ $i}} {{ $s }}:16443 check{{end}}


frontend etcd
  bind *:12379
  mode tcp
  option tcplog
  tcp-request inspect-delay 5s
  default_backend etcd

backend etcd
  mode tcp
  option ssl-hello-chk
  balance leastconn
  default-server inter 10s downinter 5s rise 2 fall 2 slowstart 60s maxconn 250 maxqueue 256 weight 100
{{range $i,$s := ls "/masterleases"}}
  server m-{{ $i}} {{ $s }}:2379 check{{end}}

`

	HAProxy = `
apiVersion: v1
kind: Pod
metadata:
  name: kube-haproxy
  namespace: kube-system
  labels:
    k8s-app: haproxy
    tier: control-plane
spec:
  containers:
  - image: {{ .ImageRepository }}/haproxy-{{.Arch}}:{{.Version}}
    imagePullPolicy: IfNotPresent
    name: haproxy
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /liveness
        port: 33305
      initialDelaySeconds: 60
      timeoutSeconds: 15
    volumeMounts:
    - mountPath: /etc/confd/templates
      name: config
    - mountPath: /etc/kubernetes/pki
      name: pki
  hostNetwork: true
  priorityClassName: system-cluster-critical
  volumes:
  - name: config
    hostPath:
      path: /etc/kubernetes/haproxy
      type: DirectoryOrCreate
  - name: pki
    hostPath:
      path: /etc/kubernetes/pki
      type: DirectoryOrCreate
`
)

